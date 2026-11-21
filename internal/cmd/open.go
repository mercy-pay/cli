package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mercy-pay/cli/internal/config"
	"github.com/spf13/cobra"
)

const docsURL = "https://docs.mercy-pay.com"

// Pages `mercypay open` knows, besides an API resource.
var openTargets = map[string]string{
	"docs":      docsURL,
	"dashboard": "https://mercy-pay.com/dashboard",
	"api":       docsURL + "/api/checkout-sessions/create",
	"cli":       docsURL + "/developers/cli",
	"webhooks":  docsURL + "/developers/webhooks/overview",
	"sandbox":   docsURL + "/start/environments",
	"status":    "https://status.mercy-pay.com",
	"changelog": docsURL + "/changelog",
}

func newOpenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "open <page>",
		Short: "Ouvrir la doc, le tableau de bord ou la référence d'une ressource",
		Long: "Ouvre une page dans le navigateur : docs, dashboard, api, cli, webhooks, sandbox, status, changelog, " +
			"ou la référence d'une ressource (payments, refunds, payouts…).",
		Example: "  mercypay open dashboard\n  mercypay open payments",
		Args:    cobra.ExactArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return openChoices(), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := openTarget(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "Ouverture de", target)
			if openURL(target) != nil {
				fmt.Fprintln(cmd.OutOrStdout(), target)
			}
			return nil
		},
	}
}

// openTarget is a known page, or the reference of a resource: "payments" opens its list page.
func openTarget(name string) (string, error) {
	name = strings.ToLower(name)
	if target, ok := openTargets[name]; ok {
		return target, nil
	}
	for _, op := range operations {
		if op.Resource == name {
			return fmt.Sprintf("%s/api/%s/%s", docsURL, name, referencePage(name)), nil
		}
	}
	return "", fmt.Errorf("page inconnue « %s » : essayez %s", name, strings.Join(openChoices()[:8], ", "))
}

// referencePage is the first page of a resource in the reference.
func referencePage(resource string) string {
	switch resource {
	case "account", "provider-status":
		return "get"
	case "fees":
		return "quote"
	case "billing-portal":
		return "create-session"
	case "test-helpers":
		return "mobile-money-numbers"
	}
	return "list"
}

func openChoices() []string {
	var choices []string
	for name := range openTargets {
		choices = append(choices, name)
	}
	sort.Strings(choices)
	seen := map[string]bool{}
	for _, op := range operations {
		if !seen[op.Resource] {
			seen[op.Resource] = true
			choices = append(choices, op.Resource)
		}
	}
	return choices
}

func newConfigCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Voir et choisir les profils (une boutique par profil)",
	}

	command.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Lister les profils enregistrés",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Profiles) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Aucun profil : lancez `mercypay login`.")
				return nil
			}
			names := make([]string, 0, len(cfg.Profiles))
			for name := range cfg.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				marker := "  "
				if name == currentProfile(cfg) {
					marker = "* "
				}
				profile := cfg.Profiles[name]
				shop := profile.MerchantName
				if shop == "" {
					shop = "boutique inconnue"
				}
				environment := "pas de clé"
				if key := cfg.Key(name); key != "" {
					environment = map[bool]string{true: "sandbox", false: "live"}[strings.HasPrefix(key, "mp_sandbox_")]
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s%s\t%s\t%s\n", marker, name, shop, environment)
			}
			return nil
		},
	})

	command.AddCommand(&cobra.Command{
		Use:   "use <profil>",
		Short: "Choisir le profil utilisé par défaut",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			cfg, err := config.Load()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var names []string
			for name := range cfg.Profiles {
				names = append(names, name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[args[0]]; !ok {
				return fmt.Errorf("profil inconnu « %s » : `mercypay config list` montre les profils", args[0])
			}
			cfg.Current = args[0]
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Profil par défaut : %s\n", args[0])
			return nil
		},
	})

	return command
}

// currentProfile is the one `config use` chose, "default" otherwise.
func currentProfile(cfg *config.Config) string {
	if cfg.Current != "" {
		return cfg.Current
	}
	return "default"
}
