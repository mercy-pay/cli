package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/mercy-pay/cli/internal/api"
	"github.com/spf13/cobra"
)

type requestLog struct {
	ID         string  `json:"id"`
	Created    string  `json:"created"`
	Method     string  `json:"method"`
	Path       string  `json:"path"`
	Status     int     `json:"status"`
	ErrorCode  *string `json:"error_code"`
	DurationMS int     `json:"duration_ms"`
}

func newLogsCommand() *cobra.Command {
	logs := &cobra.Command{Use: "logs", Short: "Le journal des appels faits avec vos clés"}

	var status, method, path string
	tail := &cobra.Command{
		Use:     "tail",
		Short:   "Afficher les appels à l'API au fil de l'eau",
		Long:    "Affiche chaque appel fait avec les clés du compte dans cet environnement, quelques secondes après sa réponse.",
		Example: "  mercypay logs tail\n  mercypay logs tail --status error --path payments",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, _, err := authenticatedClient()
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			filters := url.Values{}
			if status != "" {
				filters.Set("status", status)
			}
			if method != "" {
				filters.Set("method", strings.ToUpper(method))
			}
			if path != "" {
				filters.Set("path", path)
			}

			// Start from the newest call, unfiltered, so a filter never replays history.
			cursor := ""
			if first, err := fetchLogs(cmd, client, url.Values{"limit": {"1"}}); err != nil {
				return err
			} else if len(first) > 0 {
				cursor = first[0].ID
			}

			fmt.Fprintln(out, "En attente des appels à l'API. Ctrl+C pour arrêter.")
			for {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(pollInterval):
				}

				query := url.Values{"limit": {"100"}}
				for key, values := range filters {
					query[key] = values
				}
				if cursor != "" {
					query.Set("ending_before", cursor)
				}

				page, err := fetchLogs(cmd, client, query)
				if err != nil {
					if ctx.Err() != nil {
						return nil
					}
					fmt.Fprintf(out, "%s  lecture impossible : %v\n", clock(), err)
					continue
				}
				for i := len(page) - 1; i >= 0; i-- {
					printLog(out, page[i])
				}
				if len(page) > 0 {
					cursor = page[0].ID
				}
			}
		},
	}
	tail.Flags().StringVar(&status, "status", "", "success ou error")
	tail.Flags().StringVar(&method, "method", "", "GET, POST, PATCH ou DELETE")
	tail.Flags().StringVar(&path, "path", "", "partie du chemin, par exemple payments")

	logs.AddCommand(tail)
	return logs
}

func fetchLogs(cmd *cobra.Command, client *api.Client, query url.Values) ([]requestLog, error) {
	response, err := client.Do(cmd.Context(), api.Request{Method: "GET", Path: "/v1/request_logs", Query: query})
	if err != nil {
		return nil, err
	}
	var page struct {
		Data []requestLog `json:"data"`
	}
	return page.Data, json.Unmarshal(response.Body, &page)
}

func printLog(out io.Writer, log requestLog) {
	at := log.Created
	if parsed, err := time.Parse(time.RFC3339, log.Created); err == nil {
		at = parsed.Local().Format("15:04:05")
	}
	line := fmt.Sprintf("%s  %d  %-6s %s  %d ms", at, log.Status, log.Method, log.Path, log.DurationMS)
	if log.ErrorCode != nil {
		line += "  " + *log.ErrorCode
	}
	fmt.Fprintf(out, "%s  %s\n", line, log.ID)
}
