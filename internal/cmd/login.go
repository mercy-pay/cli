package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mercy-pay/cli/internal/api"
	"github.com/mercy-pay/cli/internal/config"
	"github.com/spf13/cobra"
)

func newLoginCommand() *cobra.Command {
	var noBrowser bool

	command := &cobra.Command{
		Use:   "login",
		Short: "Connecter la CLI à votre compte",
		Long: "Ouvre votre navigateur pour autoriser la CLI : elle reçoit une clé de test pour la boutique choisie.\n" +
			"Avec --api-key, enregistre plutôt la clé donnée (sandbox ou live).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if flags.apiKey != "" {
				return saveGivenKey(cmd, cfg, flags.apiKey)
			}
			return browserLogin(cmd, cfg, !noBrowser)
		},
	}
	command.Flags().BoolVar(&noBrowser, "no-browser", false, "afficher le lien sans ouvrir le navigateur")
	return command
}

func browserLogin(cmd *cobra.Command, cfg *config.Config, openBrowser bool) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()
	profile := cfg.Profile(flags.profile)
	client := api.New(baseURL(profile), "", userAgent())

	hostname, _ := os.Hostname()
	started, err := client.Do(ctx, api.Request{Method: "POST", Path: "/api/cli/login", Body: map[string]any{"device_name": hostname}})
	if err != nil {
		return err
	}

	var login struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURL string `json:"verification_url"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	if err := json.Unmarshal(started.Body, &login); err != nil {
		return err
	}

	fmt.Fprintf(out, "Votre code : %s\n", login.UserCode)
	fmt.Fprintf(out, "Autorisez la CLI ici : %s\n", login.VerificationURL)
	if openBrowser && openURL(login.VerificationURL) == nil {
		fmt.Fprintln(out, "Le navigateur s'est ouvert. En attente de votre accord…")
	} else {
		fmt.Fprintln(out, "En attente de votre accord…")
	}

	interval := time.Duration(max(login.Interval, 1)) * time.Second
	deadline, cancel := context.WithTimeout(ctx, time.Duration(max(login.ExpiresIn, 60))*time.Second)
	defer cancel()

	for {
		select {
		case <-deadline.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("le code a expiré : relancez `mercypay login`")
		case <-time.After(interval):
		}

		polled, err := client.Do(deadline, api.Request{Method: "POST", Path: "/api/cli/login/poll", Body: map[string]any{"device_code": login.DeviceCode}})
		if err != nil {
			return err
		}

		var result struct {
			Status   string `json:"status"`
			APIKey   string `json:"api_key"`
			Merchant struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"merchant"`
		}
		if err := json.Unmarshal(polled.Body, &result); err != nil {
			return err
		}

		switch result.Status {
		case "pending":
			continue
		case "denied":
			return errors.New("connexion refusée dans le navigateur")
		case "expired":
			return errors.New("le code a expiré : relancez `mercypay login`")
		case "approved":
			profile.MerchantID = result.Merchant.ID
			profile.MerchantName = result.Merchant.Name
			profile.BaseURL = baseURLToKeep()
			inKeychain, err := cfg.StoreKey(flags.profile, result.APIKey)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Connecté à %s, en sandbox.\n", result.Merchant.Name)
			reportKeyStorage(cmd, inKeychain)
			return nil
		default:
			return fmt.Errorf("réponse inattendue : %s", result.Status)
		}
	}
}

func saveGivenKey(cmd *cobra.Command, cfg *config.Config, key string) error {
	profile := cfg.Profile(flags.profile)
	client := api.New(baseURL(profile), key, userAgent())

	response, err := client.Do(cmd.Context(), api.Request{Method: "GET", Path: "/v1/account"})
	if err != nil {
		return err
	}

	var account struct {
		Data struct {
			Merchant struct {
				UUID string `json:"uuid"`
				Name string `json:"name"`
			} `json:"merchant"`
			APIKey struct {
				Environment string `json:"environment"`
			} `json:"api_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body, &account); err != nil {
		return err
	}

	profile.MerchantID = account.Data.Merchant.UUID
	profile.MerchantName = account.Data.Merchant.Name
	profile.BaseURL = baseURLToKeep()
	inKeychain, err := cfg.StoreKey(flags.profile, key)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Connecté à %s, en %s.\n", account.Data.Merchant.Name, account.Data.APIKey.Environment)
	reportKeyStorage(cmd, inKeychain)
	return nil
}

// baseURLToKeep remembers a non-default address (a local server) for the next commands.
func baseURLToKeep() string {
	if url := baseURL(nil); url != api.DefaultBaseURL {
		return url
	}
	return ""
}

func reportKeyStorage(cmd *cobra.Command, inKeychain bool) {
	if inKeychain {
		fmt.Fprintln(cmd.OutOrStdout(), "Clé enregistrée dans le trousseau du système.")
		return
	}
	dir, _ := config.Dir()
	fmt.Fprintf(cmd.OutOrStdout(), "Pas de trousseau disponible : clé enregistrée dans %s/config.json, lisible par vous seul.\n", dir)
}

func newLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Oublier la clé de ce profil sur cet ordinateur",
		Long:  "Supprime la clé enregistrée localement. Pour la révoquer, supprimez-la dans Développeurs, onglet Clés API.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := cfg.Forget(flags.profile); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Déconnecté. La clé reste valable tant qu'elle n'est pas supprimée dans Développeurs, onglet Clés API.")
			return nil
		},
	}
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Voir le compte et l'environnement de la clé utilisée",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := authenticatedClient()
			if err != nil {
				return err
			}
			response, err := client.Do(cmd.Context(), api.Request{Method: "GET", Path: "/v1/account"})
			if err != nil {
				return err
			}
			var account struct {
				Data struct {
					Merchant struct{ Name string } `json:"merchant"`
					APIKey   struct {
						Environment string `json:"environment"`
						KeyPrefix   string `json:"key_prefix"`
					} `json:"api_key"`
				} `json:"data"`
			}
			if err := json.Unmarshal(response.Body, &account); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Compte : %s\nEnvironnement : %s\nClé : %s…\nAPI : %s\n",
				account.Data.Merchant.Name, account.Data.APIKey.Environment, account.Data.APIKey.KeyPrefix, client.BaseURL)
			return nil
		},
	}
}

func openURL(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return errors.New("adresse refusée")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
