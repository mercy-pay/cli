package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mercy-pay/cli/internal/api"
	"github.com/mercy-pay/cli/internal/params"
	"github.com/spf13/cobra"
)

// operation is one call of the API, as `mercypay <resource> <verb> [ids…]`.
type operation struct {
	Resource    string
	Verb        string
	Method      string
	Path        string // {} for each id given as an argument, in order
	Short       string
	OperationID string // the name in the API reference, kept in step by a test
}

var operations = []operation{
	{"account", "get", "GET", "/v1/account", "Lire le compte, ses frais et ses limites", "getAccount"},
	{"fees", "quote", "POST", "/v1/fees/quote", "Calculer les frais d'un montant", "createFeeQuote"},
	{"provider-status", "get", "GET", "/v1/provider_status", "État des opérateurs", "getProviderStatus"},

	{"checkout-sessions", "create", "POST", "/v1/checkout_sessions", "Créer une session de paiement", "createCheckoutSession"},
	{"checkout-sessions", "list", "GET", "/v1/checkout_sessions", "Lister les sessions", "listCheckoutSessions"},
	{"checkout-sessions", "get", "GET", "/v1/checkout_sessions/{}", "Lire une session", "getCheckoutSession"},
	{"checkout-sessions", "expire", "POST", "/v1/checkout_sessions/{}/expire", "Faire expirer une session", "expireCheckoutSession"},

	{"payments", "create", "POST", "/v1/payments", "Créer une charge directe", "createPayment"},
	{"payments", "list", "GET", "/v1/payments", "Lister les paiements", "listPayments"},
	{"payments", "get", "GET", "/v1/payments/{}", "Lire un paiement", "getPayment"},
	{"payments", "refund", "POST", "/v1/payments/{}/refunds", "Rembourser un paiement", "createRefund"},

	{"refunds", "list", "GET", "/v1/refunds", "Lister les remboursements", "listRefunds"},
	{"refunds", "get", "GET", "/v1/refunds/{}", "Lire un remboursement", "getRefund"},
	{"refunds", "cancel", "POST", "/v1/refunds/{}/cancel", "Annuler un remboursement", "cancelRefund"},

	{"wallets", "list", "GET", "/v1/wallets", "Lister les soldes", "listWallets"},
	{"wallet-transfers", "list", "GET", "/v1/wallet_transfers", "Lister les transferts entre soldes", "listWalletTransfers"},
	{"wallet-transfers", "create", "POST", "/v1/wallet_transfers", "Transférer entre deux soldes", "createWalletTransfer"},

	{"payouts", "create", "POST", "/v1/payouts", "Demander un retrait", "createPayout"},
	{"payouts", "list", "GET", "/v1/payouts", "Lister les retraits", "listPayouts"},
	{"payouts", "get", "GET", "/v1/payouts/{}", "Lire un retrait", "getPayout"},

	{"payment-links", "create", "POST", "/v1/payment_links", "Créer un lien de paiement", "createPaymentLink"},
	{"payment-links", "list", "GET", "/v1/payment_links", "Lister les liens", "listPaymentLinks"},
	{"payment-links", "get", "GET", "/v1/payment_links/{}", "Lire un lien", "getPaymentLink"},
	{"payment-links", "update", "PATCH", "/v1/payment_links/{}", "Modifier un lien", "updatePaymentLink"},
	{"payment-links", "delete", "DELETE", "/v1/payment_links/{}", "Supprimer un lien", "deletePaymentLink"},
	{"payment-links", "archive", "POST", "/v1/payment_links/{}/archive", "Archiver un lien", "archivePaymentLink"},
	{"payment-links", "payments", "GET", "/v1/payment_links/{}/payments", "Lister les paiements d'un lien", "listPaymentLinkPayments"},

	{"plans", "create", "POST", "/v1/plans", "Créer un plan", "createPlan"},
	{"plans", "list", "GET", "/v1/plans", "Lister les plans", "listPlans"},
	{"plans", "get", "GET", "/v1/plans/{}", "Lire un plan", "getPlan"},
	{"plans", "update", "PATCH", "/v1/plans/{}", "Modifier un plan", "updatePlan"},
	{"plans", "delete", "DELETE", "/v1/plans/{}", "Supprimer un plan", "deletePlan"},
	{"plans", "archive", "POST", "/v1/plans/{}/archive", "Archiver un plan", "archivePlan"},
	{"plans", "restore", "POST", "/v1/plans/{}/restore", "Réactiver un plan archivé", "restorePlan"},

	{"customers", "create", "POST", "/v1/customers", "Créer un client", "createCustomer"},
	{"customers", "list", "GET", "/v1/customers", "Lister les clients", "listCustomers"},
	{"customers", "get", "GET", "/v1/customers/{}", "Lire un client", "getCustomer"},
	{"customers", "update", "PATCH", "/v1/customers/{}", "Modifier un client", "updateCustomer"},
	{"customers", "delete", "DELETE", "/v1/customers/{}", "Supprimer un client", "deleteCustomer"},
	{"customers", "payment-methods", "GET", "/v1/customers/{}/payment_methods", "Lister les moyens de paiement d'un client", "listCustomerPaymentMethods"},
	{"customers", "set-default-payment-method", "POST", "/v1/customers/{}/payment_methods/{}/default", "Choisir le moyen de paiement par défaut", "setDefaultPaymentMethod"},
	{"customers", "remove-payment-method", "DELETE", "/v1/customers/{}/payment_methods/{}", "Retirer un moyen de paiement", "removePaymentMethod"},

	{"subscriptions", "create", "POST", "/v1/subscriptions", "Créer un abonnement", "createSubscription"},
	{"subscriptions", "list", "GET", "/v1/subscriptions", "Lister les abonnements", "listSubscriptions"},
	{"subscriptions", "get", "GET", "/v1/subscriptions/{}", "Lire un abonnement", "getSubscription"},
	{"subscriptions", "update", "PATCH", "/v1/subscriptions/{}", "Modifier un abonnement", "updateSubscription"},
	{"subscriptions", "cancel", "POST", "/v1/subscriptions/{}/cancel", "Annuler un abonnement", "cancelSubscription"},
	{"subscriptions", "reactivate", "POST", "/v1/subscriptions/{}/reactivate", "Réactiver un abonnement", "reactivateSubscription"},
	{"subscriptions", "advance", "POST", "/v1/subscriptions/{}/advance", "Passer à l'échéance suivante (sandbox)", "advanceSubscription"},
	{"subscriptions", "pause", "POST", "/v1/subscriptions/{}/pause", "Mettre en pause", "pauseSubscription"},
	{"subscriptions", "resume", "POST", "/v1/subscriptions/{}/resume", "Reprendre", "resumeSubscription"},
	{"subscriptions", "invoices", "GET", "/v1/subscriptions/{}/invoices", "Lister les factures d'un abonnement", "listSubscriptionInvoices"},

	{"invoices", "list", "GET", "/v1/invoices", "Lister les factures", "listInvoices"},
	{"invoices", "get", "GET", "/v1/invoices/{}", "Lire une facture", "getInvoice"},
	{"invoices", "pay", "POST", "/v1/invoices/{}/pay", "Encaisser une facture", "payInvoice"},
	{"invoices", "send", "POST", "/v1/invoices/{}/send", "Envoyer une facture au client", "sendInvoice"},

	{"billing-portal", "create-session", "POST", "/v1/billing_portal/sessions", "Ouvrir l'espace client d'un client", "createBillingPortalSession"},

	{"events", "list", "GET", "/v1/events", "Lister les événements", "listEvents"},
	{"events", "get", "GET", "/v1/events/{}", "Lire un événement", "getEvent"},
	{"events", "deliveries", "GET", "/v1/events/{}/deliveries", "Lister les envois d'un événement", "listEventDeliveries"},
	{"events", "resend", "POST", "/v1/events/{}/resend", "Renvoyer un événement", "resendEvent"},

	{"webhook-endpoints", "create", "POST", "/v1/webhook_endpoints", "Créer un endpoint", "createWebhookEndpoint"},
	{"webhook-endpoints", "list", "GET", "/v1/webhook_endpoints", "Lister les endpoints", "listWebhookEndpoints"},
	{"webhook-endpoints", "get", "GET", "/v1/webhook_endpoints/{}", "Lire un endpoint", "getWebhookEndpoint"},
	{"webhook-endpoints", "update", "PATCH", "/v1/webhook_endpoints/{}", "Modifier un endpoint", "updateWebhookEndpoint"},
	{"webhook-endpoints", "delete", "DELETE", "/v1/webhook_endpoints/{}", "Supprimer un endpoint", "deleteWebhookEndpoint"},
	{"webhook-endpoints", "rotate-secret", "POST", "/v1/webhook_endpoints/{}/rotate_secret", "Changer le secret", "rotateWebhookEndpointSecret"},
	{"webhook-endpoints", "test", "POST", "/v1/webhook_endpoints/{}/test", "Envoyer un événement webhook.test", "testWebhookEndpoint"},

	{"request-logs", "list", "GET", "/v1/request_logs", "Lister les appels faits avec vos clés", "listRequestLogs"},

	{"embed-domains", "create", "POST", "/v1/embed_domains", "Déclarer un site pour le paiement intégré", "createEmbedDomain"},
	{"embed-domains", "list", "GET", "/v1/embed_domains", "Lister les sites autorisés", "listEmbedDomains"},
	{"embed-domains", "get", "GET", "/v1/embed_domains/{}", "Lire un site autorisé", "getEmbedDomain"},
	{"embed-domains", "verify", "POST", "/v1/embed_domains/{}/verify", "Vérifier l'enregistrement TXT d'un site", "verifyEmbedDomain"},
	{"embed-domains", "delete", "DELETE", "/v1/embed_domains/{}", "Retirer un site", "deleteEmbedDomain"},

	{"test-helpers", "mobile-money-numbers", "GET", "/v1/test_helpers/mobile_money_numbers", "Lister les numéros de test et ce que chacun produit", "listTestMobileMoneyNumbers"},
}

// Operations that exist in the API but are a dedicated command instead.
var dedicatedOperations = map[string]string{"createTestEvent": "trigger"}

func addResourceCommands(root *cobra.Command) {
	groups := map[string]*cobra.Command{}

	for _, op := range operations {
		group, ok := groups[op.Resource]
		if !ok {
			group = &cobra.Command{Use: op.Resource, Short: "Appels à /v1/" + strings.ReplaceAll(op.Resource, "-", "_"), GroupID: "resources"}
			groups[op.Resource] = group
			root.AddCommand(group)
		}
		group.AddCommand(operationCommand(op))
	}
}

func operationCommand(op operation) *cobra.Command {
	var data []string
	var idempotencyKey string
	var extras operationExtras
	ids := strings.Count(op.Path, "{}")

	use := op.Verb
	for i := 0; i < ids; i++ {
		use += " <id>"
	}

	command := &cobra.Command{
		Use:   use,
		Short: op.Short,
		Long:  fmt.Sprintf("%s\n\n%s %s", op.Short, op.Method, "/api"+op.Path),
		Args:  cobra.ExactArgs(ids),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := op.Path
			for _, id := range args {
				path = strings.Replace(path, "{}", id, 1)
			}
			return runOperation(cmd, op, path, data, idempotencyKey, extras)
		},
	}
	addDataFlags(command, &data, &idempotencyKey, op.Method)
	addOperationExtras(command, op, &extras)
	return command
}

func addDataFlags(command *cobra.Command, data *[]string, idempotencyKey *string, method string) {
	command.Flags().StringArrayVarP(data, "data", "d", nil, "paramètre cle=valeur (texte), cle:=json (nombre, booléen…), metadata[cle]=v, liste[]=v")
	if method != http.MethodGet {
		command.Flags().StringVarP(idempotencyKey, "idempotency-key", "i", "", "Idempotency-Key à envoyer (générée sinon)")
	}
}

// call sends one request and prints the answer in the chosen format.
func call(cmd *cobra.Command, method, path string, data []string, idempotencyKey string) error {
	_, err := send(cmd, method, path, data, idempotencyKey, true)
	return err
}

// send sends one request, prints the answer when asked, and returns it.
func send(cmd *cobra.Command, method, path string, data []string, idempotencyKey string, print bool) (*api.Response, error) {
	client, _, err := authenticatedClient()
	if err != nil {
		return nil, err
	}

	request := api.Request{Method: method, Path: path, IdempotencyKey: idempotencyKey}
	if method == http.MethodGet || method == http.MethodDelete {
		query, err := params.Query(data)
		if err != nil {
			return nil, err
		}
		request.Query = query
	} else if len(data) > 0 {
		body, err := params.Parse(data)
		if err != nil {
			return nil, err
		}
		request.Body = body
	} else {
		request.Body = map[string]any{}
	}

	response, err := client.Do(cmd.Context(), request)
	if response != nil && (print || err != nil) {
		render(cmd.OutOrStdout(), response.Body, flags.output)
	}
	return response, err
}

// newRawCommands are `mercypay get|post|patch|delete <path>` for anything else.
func newRawCommands() []*cobra.Command {
	var commands []*cobra.Command
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
		method := method
		var data []string
		var idempotencyKey string
		command := &cobra.Command{
			Use:     strings.ToLower(method) + " <chemin>",
			Short:   method + " sur n'importe quel chemin de l'API",
			Example: "  mercypay " + strings.ToLower(method) + " /v1/payments",
			Args:    cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return call(cmd, method, args[0], data, idempotencyKey)
			},
		}
		addDataFlags(command, &data, &idempotencyKey, method)
		commands = append(commands, command)
	}
	return commands
}
