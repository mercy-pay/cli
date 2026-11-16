package cmd

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/mercy-pay/cli/internal/api"
	"github.com/spf13/cobra"
)

// eventTypes are the types the API can fire in sandbox.
var eventTypes = []string{
	"payment.succeeded", "payment.failed", "payment.refunded", "payment.disputed", "payment.dispute_closed",
	"refund.succeeded", "refund.failed", "refund.canceled",
	"payout.paid", "payout.failed", "payout.canceled",
	"subscription.created", "subscription.updated", "subscription.past_due", "subscription.unpaid",
	"subscription.paused", "subscription.resumed", "subscription.canceled",
	"invoice.created", "invoice.upcoming", "invoice.paid", "invoice.canceled", "invoice.payment_failed",
	"checkout_session.completed", "provider_status.changed", "account.updated",
}

func newTriggerCommand() *cobra.Command {
	var list bool

	command := &cobra.Command{
		Use:   "trigger <type>",
		Short: "Déclencher un événement de test en sandbox",
		Long: "Crée un événement avec des données d'exemple au format de l'objet réel, puis l'envoie à vos endpoints\n" +
			"sandbox et à `mercypay listen`. Rien d'autre n'est créé : les identifiants ne renvoient à aucun objet.",
		Example:   "  mercypay trigger payment.succeeded\n  mercypay trigger --list",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: eventTypes,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if list || len(args) == 0 {
				for _, eventType := range eventTypes {
					fmt.Fprintln(out, eventType)
				}
				return nil
			}
			if !slices.Contains(eventTypes, args[0]) {
				// The API stays the judge: a newer server may know more types.
				fmt.Fprintf(cmd.ErrOrStderr(), "Type inconnu de cette version de la CLI, envoyé quand même : %s\n", args[0])
			}

			client, _, err := authenticatedClient()
			if err != nil {
				return err
			}
			response, err := client.Do(cmd.Context(), api.Request{Method: "POST", Path: "/v1/test_helpers/events", Body: map[string]any{"type": args[0]}})
			if err != nil {
				return err
			}

			var created struct {
				Data struct {
					ID   string `json:"id"`
					Type string `json:"type"`
				} `json:"data"`
			}
			if err := json.Unmarshal(response.Body, &created); err != nil {
				return err
			}
			fmt.Fprintf(out, "Déclenché : %s %s\n", created.Data.Type, created.Data.ID)
			return nil
		},
	}
	command.Flags().BoolVar(&list, "list", false, "lister les types disponibles")
	return command
}
