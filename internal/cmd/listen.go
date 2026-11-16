package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/mercy-pay/cli/internal/api"
	"github.com/mercy-pay/cli/internal/signature"
	"github.com/spf13/cobra"
)

// pollInterval is how often new events are asked for.
const pollInterval = 2 * time.Second

type event struct {
	ID       string          `json:"id"`
	Object   string          `json:"object"`
	Type     string          `json:"type"`
	Created  string          `json:"created"`
	Livemode bool            `json:"livemode"`
	Data     json.RawMessage `json:"data"`
}

type eventPage struct {
	HasMore bool    `json:"has_more"`
	Data    []event `json:"data"`
}

func newListenCommand() *cobra.Command {
	var forwardTo string
	var types []string
	var printSecret bool

	command := &cobra.Command{
		Use:   "listen",
		Short: "Recevoir les événements du compte, et les relayer à votre serveur local",
		Long: "Affiche chaque nouvel événement du compte. Avec --forward-to, l'envoie aussi à votre serveur,\n" +
			"signé comme un vrai webhook (en-tête MercyPay-Signature) avec le secret affiché au démarrage.\n" +
			"Les événements sont lus toutes les 2 secondes.",
		Example: "  mercypay listen --forward-to localhost:8000/webhooks/mercy-pay\n" +
			"  mercypay listen --events payment.succeeded,payment.failed",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, cfg, err := authenticatedClient()
			if err != nil {
				return err
			}

			profile := cfg.Profile(flags.profile)
			if profile.ListenSecret == "" {
				secret, err := signature.NewSecret()
				if err != nil {
					return err
				}
				profile.ListenSecret = secret
				if err := cfg.Save(); err != nil {
					return err
				}
			}

			if printSecret {
				fmt.Fprintln(cmd.OutOrStdout(), profile.ListenSecret)
				return nil
			}

			target := ""
			if forwardTo != "" {
				target, err = normalizeTarget(forwardTo)
				if err != nil {
					return err
				}
			}

			listener := &listener{
				client:  client,
				target:  target,
				secret:  profile.ListenSecret,
				types:   types,
				out:     cmd.OutOrStdout(),
				http:    &http.Client{Timeout: 10 * time.Second},
				poll:    pollInterval,
				nowUnix: func() int64 { return time.Now().Unix() },
			}
			return listener.run(cmd.Context())
		},
	}

	command.Flags().StringVarP(&forwardTo, "forward-to", "f", "", "adresse locale qui reçoit les événements, par exemple localhost:8000/webhooks")
	command.Flags().StringSliceVarP(&types, "events", "e", nil, "types à relayer, séparés par des virgules (tous par défaut)")
	command.Flags().BoolVar(&printSecret, "print-secret", false, "afficher seulement le secret de signature, puis quitter")
	return command
}

type listener struct {
	client  *api.Client
	target  string
	secret  string
	types   []string
	out     io.Writer
	http    *http.Client
	poll    time.Duration
	nowUnix func() int64
	cursor  string
}

func (l *listener) run(ctx context.Context) error {
	if err := l.start(ctx); err != nil {
		return err
	}

	fmt.Fprintf(l.out, "Prêt. Secret de signature : %s\n", l.secret)
	if l.target != "" {
		fmt.Fprintf(l.out, "Les événements sont relayés à %s. Ctrl+C pour arrêter.\n", l.target)
	} else {
		fmt.Fprintln(l.out, "Les événements s'affichent ici. Ctrl+C pour arrêter.")
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(l.poll):
		}

		if err := l.tick(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			fmt.Fprintf(l.out, "%s  lecture impossible : %v (nouvel essai dans %s)\n", clock(), err, l.poll)
		}
	}
}

// start remembers the newest event so only what happens next is shown.
func (l *listener) start(ctx context.Context) error {
	response, err := l.client.Do(ctx, api.Request{Method: "GET", Path: "/v1/events", Query: url.Values{"limit": {"1"}}})
	if err != nil {
		return err
	}
	var page eventPage
	if err := json.Unmarshal(response.Body, &page); err != nil {
		return err
	}
	if len(page.Data) > 0 {
		l.cursor = page.Data[0].ID
	}
	return nil
}

// tick reads every event newer than the cursor, oldest first, and handles each.
func (l *listener) tick(ctx context.Context) error {
	for {
		query := url.Values{"limit": {"100"}}
		if l.cursor != "" {
			query.Set("ending_before", l.cursor)
		}

		response, err := l.client.Do(ctx, api.Request{Method: "GET", Path: "/v1/events", Query: query})
		if err != nil {
			return err
		}
		var page eventPage
		if err := json.Unmarshal(response.Body, &page); err != nil {
			return err
		}
		if len(page.Data) == 0 {
			return nil
		}

		// The page is newest first: handle it in the order things happened.
		for i := len(page.Data) - 1; i >= 0; i-- {
			l.handle(ctx, page.Data[i])
		}
		l.cursor = page.Data[0].ID

		// Without a cursor the first page is the newest one; with one, more may be waiting.
		if !page.HasMore || query.Get("ending_before") == "" {
			return nil
		}
	}
}

func (l *listener) handle(ctx context.Context, e event) {
	if len(l.types) > 0 && !slices.Contains(l.types, e.Type) {
		return
	}

	line := fmt.Sprintf("%s  %-30s %s", clock(), e.Type, e.ID)
	if l.target == "" {
		fmt.Fprintln(l.out, line)
		return
	}

	status, err := l.forward(ctx, e)
	switch {
	case err != nil:
		fmt.Fprintf(l.out, "%s  [échec : %v]\n", line, err)
	default:
		fmt.Fprintf(l.out, "%s  [%d]\n", line, status)
	}
}

// forward posts the event the way Mercy Pay's servers deliver a webhook.
func (l *listener) forward(ctx context.Context, e event) (int, error) {
	body, err := json.Marshal(e)
	if err != nil {
		return 0, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, l.target, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "MercyPay-Webhooks/1.0 (CLI)")
	request.Header.Set("MercyPay-Signature", signature.Header(l.secret, body, l.nowUnix()))
	request.Header.Set("MercyPay-Event-Id", e.ID)
	request.Header.Set("MercyPay-Event-Type", e.Type)
	request.Header.Set("MercyPay-Delivery-Id", "whd_cli_"+strings.TrimPrefix(e.ID, "evt_"))

	response, err := l.http.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode, nil
}

// normalizeTarget accepts "localhost:3000/hook", ":3000/hook" or a full URL.
func normalizeTarget(raw string) (string, error) {
	if strings.HasPrefix(raw, ":") {
		raw = "localhost" + raw
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("--forward-to : adresse invalide, par exemple localhost:8000/webhooks")
	}
	return parsed.String(), nil
}

func clock() string {
	return time.Now().Format("15:04:05")
}
