// Package cmd holds the mercypay commands.
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mercy-pay/cli/internal/api"
	"github.com/mercy-pay/cli/internal/config"
	"github.com/spf13/cobra"
)

// Version is set at build time.
var Version = "dev"

type globals struct {
	apiKey  string
	profile string
	baseURL string
}

var flags globals

// NewRoot builds the command tree.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "mercypay",
		Short:         "La ligne de commande de Mercy Pay",
		Long:          "Connectez-vous, recevez vos webhooks en local, déclenchez des événements de test et appelez l'API.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}

	root.PersistentFlags().StringVar(&flags.apiKey, "api-key", "", "clé API à utiliser (sinon MERCYPAY_API_KEY, sinon celle enregistrée par login)")
	root.PersistentFlags().StringVarP(&flags.profile, "profile", "p", "default", "profil à utiliser")
	root.PersistentFlags().StringVar(&flags.baseURL, "base-url", "", "adresse de l'API (sinon MERCYPAY_API_BASE)")
	_ = root.PersistentFlags().MarkHidden("base-url")

	root.AddGroup(
		&cobra.Group{ID: "tools", Title: "Développer en local :"},
		&cobra.Group{ID: "resources", Title: "Appeler l'API :"},
	)
	for _, command := range []*cobra.Command{newLoginCommand(), newLogoutCommand(), newStatusCommand(), newListenCommand(), newTriggerCommand(), newLogsCommand()} {
		command.GroupID = "tools"
		root.AddCommand(command)
	}
	for _, command := range newRawCommands() {
		command.GroupID = "resources"
		root.AddCommand(command)
	}
	addResourceCommands(root)

	return root
}

// Execute runs the CLI and returns the exit code.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := NewRoot().ExecuteContext(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return 130
		}
		fmt.Fprintln(os.Stderr, "Erreur :", err)
		var apiError *api.Error
		if errors.As(err, &apiError) && apiError.RequestID != "" {
			fmt.Fprintln(os.Stderr, "Request-Id :", apiError.RequestID)
		}
		return 1
	}
	return 0
}

func userAgent() string {
	return "mercypay-cli/" + Version
}

func baseURL(profile *config.Profile) string {
	switch {
	case flags.baseURL != "":
		return flags.baseURL
	case os.Getenv("MERCYPAY_API_BASE") != "":
		return os.Getenv("MERCYPAY_API_BASE")
	case profile != nil && profile.BaseURL != "":
		return profile.BaseURL
	}
	return api.DefaultBaseURL
}

// authenticatedClient finds the key: --api-key, then MERCYPAY_API_KEY, then the profile.
func authenticatedClient() (*api.Client, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	profile := cfg.Profiles[flags.profile]

	key := flags.apiKey
	if key == "" {
		key = os.Getenv("MERCYPAY_API_KEY")
	}
	if key == "" {
		key = cfg.Key(flags.profile)
	}
	if key == "" {
		return nil, nil, errors.New("aucune clé : lancez `mercypay login` ou passez --api-key")
	}

	return api.New(baseURL(profile), key, userAgent()), cfg, nil
}

// printJSON writes the body indented and readable (accents and slashes as they are), or as
// is when it is not JSON.
func printJSON(out io.Writer, body []byte) {
	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") != nil {
		out.Write(body)
		fmt.Fprintln(out)
		return
	}
	fmt.Fprintln(out, readable(pretty.String()))
}

var unicodeEscape = regexp.MustCompile(`\\u[0-9a-fA-F]{4}(\\u[0-9a-fA-F]{4})?`)

// readable undoes the escapes PHP adds by default (\/ and \u00e9) and that JSON does not need.
// A quote or a control character stays escaped, so the output is still valid JSON.
func readable(text string) string {
	text = strings.ReplaceAll(text, `\/`, "/")
	return unicodeEscape.ReplaceAllStringFunc(text, func(escape string) string {
		var decoded string
		err := json.Unmarshal([]byte(`"`+escape+`"`), &decoded)
		if err != nil || strings.ContainsRune(decoded, utf8.RuneError) || strings.ContainsAny(decoded, "\"\\") || strings.IndexFunc(decoded, unicode.IsControl) >= 0 {
			return escape
		}
		return decoded
	})
}
