package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/mercy-pay/cli/internal/catalog"
	"github.com/spf13/cobra"
)

// operationExtras are the options a few commands add on top of the plain API call.
type operationExtras struct {
	wait     bool
	timeout  time.Duration
	open     bool
	test     string
	provider string
	amount   int
	currency string
}

// The pause before the first new read; shorter in tests.
var waitFirstDelay = 2 * time.Second

var (
	finalPaymentStatuses = []string{"succeeded", "failed", "canceled", "expired", "refunded", "disputed"}
	finalSessionStatuses = []string{"completed", "expired", "cancelled"}

	// What a sandbox number can produce, for --test.
	testOutcomes = []string{"succeeded", "pending", "insufficient_balance", "payer_not_found", "payer_limit_reached", "payment_not_approved", "failed"}
)

func addOperationExtras(command *cobra.Command, op operation, extras *operationExtras) {
	waitable := (op.Resource == "payments" && (op.Verb == "get" || op.Verb == "create")) ||
		(op.Resource == "checkout-sessions" && (op.Verb == "get" || op.Verb == "create"))
	if waitable {
		command.Flags().BoolVarP(&extras.wait, "wait", "w", false, "attendre le statut final, puis l'afficher")
		command.Flags().DurationVar(&extras.timeout, "timeout", 2*time.Minute, "durée maximale d'attente avec --wait")
	}

	if op.Resource == "checkout-sessions" && op.Verb == "create" {
		command.Flags().BoolVar(&extras.open, "open", false, "ouvrir la page de paiement dans le navigateur")
	}

	if op.Resource == "payments" && op.Verb == "create" {
		command.Flags().StringVar(&extras.test, "test", "", "sandbox : payer avec le numéro de test qui produit ce résultat ("+strings.Join(testOutcomes, ", ")+")")
		command.Flags().StringVar(&extras.provider, "provider", "", "opérateur du paiement de test, par exemple MTN_MOMO_CMR")
		command.Flags().IntVar(&extras.amount, "amount", 1000, "montant du paiement de test, en unités mineures")
		command.Flags().StringVar(&extras.currency, "currency", "XAF", "devise du paiement de test")
		_ = command.RegisterFlagCompletionFunc("test", cobra.FixedCompletions(testOutcomes, cobra.ShellCompDirectiveNoFileComp))
		_ = command.RegisterFlagCompletionFunc("provider", cobra.FixedCompletions(catalog.Providers(), cobra.ShellCompDirectiveNoFileComp))
		command.Example = "  mercypay payments create --test succeeded --provider MTN_MOMO_CMR --wait\n" +
			"  mercypay payments create --test insufficient_balance --provider ORANGE_CMR --wait"
	}
}

func runOperation(cmd *cobra.Command, op operation, path string, data []string, idempotencyKey string, extras operationExtras) error {
	if extras.test != "" {
		testData, err := testPaymentData(cmd, extras)
		if err != nil {
			return err
		}
		data = append(testData, data...)
	}

	response, err := send(cmd, op.Method, path, data, idempotencyKey, !extras.wait)
	if err != nil {
		return err
	}

	object := dataOf(response.Body)

	if extras.open {
		checkoutURL, _ := object["checkout_url"].(string)
		if checkoutURL == "" {
			return errors.New("la session n'a pas de checkout_url")
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Ouverture de", checkoutURL)
		if err := openURL(checkoutURL); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Ouvrez cette adresse :", checkoutURL)
		}
	}

	if !extras.wait {
		return nil
	}

	id, _ := object["id"].(string)
	if id == "" {
		return errors.New("aucun id à suivre dans la réponse")
	}
	readPath, final := "/v1/payments/"+id, finalPaymentStatuses
	if op.Resource == "checkout-sessions" {
		readPath, final = "/v1/checkout_sessions/"+id, finalSessionStatuses
	}
	return waitForFinal(cmd, readPath, final, extras.timeout, object)
}

// waitForFinal reads the object again, more and more spaced, until its status is final.
func waitForFinal(cmd *cobra.Command, path string, final []string, timeout time.Duration, current map[string]any) error {
	deadline := time.Now().Add(timeout)
	delay := waitFirstDelay
	lastStatus := ""

	for {
		status, _ := current["status"].(string)
		if status != lastStatus {
			fmt.Fprintf(cmd.ErrOrStderr(), "Statut : %s\n", status)
			lastStatus = status
		}
		if slices.Contains(final, status) {
			encoded, _ := json.Marshal(current)
			render(cmd.OutOrStdout(), encoded, flags.output)
			return nil
		}
		if time.Now().Add(delay).After(deadline) {
			encoded, _ := json.Marshal(current)
			render(cmd.OutOrStdout(), encoded, flags.output)
			return fmt.Errorf("pas de statut final après %s (statut : %s)", timeout, status)
		}

		select {
		case <-cmd.Context().Done():
			return cmd.Context().Err()
		case <-time.After(delay):
		}
		delay = min(delay*3/2, 10*time.Second)

		response, err := send(cmd, http.MethodGet, path, nil, "", false)
		if err != nil {
			return err
		}
		current = dataOf(response.Body)
	}
}

// testPaymentData is the body of a sandbox payment that gives the asked outcome.
func testPaymentData(cmd *cobra.Command, extras operationExtras) ([]string, error) {
	if !strings.HasPrefix(resolvedKey(), "mp_sandbox_") {
		return nil, errors.New("--test ne marche qu'avec une clé sandbox (mp_sandbox_…)")
	}
	if extras.provider == "" {
		return nil, errors.New("--test demande --provider, par exemple --provider MTN_MOMO_CMR")
	}

	query := url.Values{"provider": {strings.ToUpper(extras.provider)}}
	response, err := send(cmd, http.MethodGet, "/v1/test_helpers/mobile_money_numbers?"+query.Encode(), nil, "", false)
	if err != nil {
		return nil, err
	}

	var list struct {
		Data []struct {
			Country     string  `json:"country"`
			Provider    string  `json:"provider"`
			Phone       string  `json:"phone"`
			AppliesTo   string  `json:"applies_to"`
			Outcome     string  `json:"outcome"`
			FailureCode *string `json:"failure_code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body, &list); err != nil {
		return nil, err
	}
	if len(list.Data) == 0 {
		return nil, fmt.Errorf("aucun numéro de test pour %s", extras.provider)
	}

	for _, number := range list.Data {
		code := ""
		if number.FailureCode != nil {
			code = *number.FailureCode
		}
		if number.AppliesTo == "payment" && (number.Outcome == extras.test || code == extras.test) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Numéro de test %s (%s) : %s\n", number.Phone, number.Provider, extras.test)
			return []string{
				"amount:=" + fmt.Sprint(extras.amount),
				"currency=" + strings.ToUpper(extras.currency),
				"country=" + number.Country,
				"provider=" + number.Provider,
				"phone=" + number.Phone,
				"metadata[mercypay_cli_test]=" + extras.test,
			}, nil
		}
	}
	return nil, fmt.Errorf("%s n'a pas de numéro de test pour « %s »", extras.provider, extras.test)
}

// dataOf reads {"data": {...}} or the object itself.
func dataOf(body []byte) map[string]any {
	var decoded map[string]any
	_ = json.Unmarshal(body, &decoded)
	if inner, ok := decoded["data"].(map[string]any); ok {
		return inner
	}
	return decoded
}
