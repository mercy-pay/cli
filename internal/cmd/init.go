package cmd

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

//go:embed templates
var templates embed.FS

// The stacks `mercypay init` knows, with what they make.
var stacks = map[string]string{
	"node":    "une boutique Node.js (Express) avec le paiement intégré et les webhooks",
	"next":    "une boutique Next.js (App Router) avec le paiement intégré et les webhooks",
	"php":     "une boutique PHP sans framework, avec le paiement intégré et les webhooks",
	"laravel": "les fichiers à ajouter à une app Laravel existante",
	"django":  "une app à ajouter à un projet Django existant",
	"flutter": "un écran de paiement à ajouter à une app Flutter",
}

// Files whose name can't carry their leading dot in the embedded tree.
var renamed = map[string]string{
	"gitignore":   ".gitignore",
	"env":         ".env",
	"env.local":   ".env.local",
	"env.example": ".env.example",
}

func newInitCommand() *cobra.Command {
	var force bool

	command := &cobra.Command{
		Use:   "init <stack> [dossier]",
		Short: "Créer un projet d'exemple prêt à lancer",
		Long: "Crée un projet d'exemple : une session de paiement intégrée, la route des webhooks, et la clé sandbox " +
			"du profil dans le fichier d'environnement.\n\nStacks : " + strings.Join(stackNames(), ", ") + ".",
		Example: "  mercypay init node\n  mercypay init next ma-boutique",
		Args:    cobra.RangeArgs(1, 2),
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveFilterDirs
			}
			var choices []string
			for _, name := range stackNames() {
				choices = append(choices, name+"\t"+stacks[name])
			}
			return choices, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			stack := strings.ToLower(args[0])
			if _, ok := stacks[stack]; !ok {
				return fmt.Errorf("stack inconnue « %s » : %s", args[0], strings.Join(stackNames(), ", "))
			}
			target := "mercy-pay-" + stack
			if len(args) == 2 {
				target = args[1]
			}

			key := resolvedKey()
			if !strings.HasPrefix(key, "mp_sandbox_") {
				// Never write a live key in a sample project.
				key = "mp_sandbox_..."
			}

			written, err := writeStack(stack, target, key, force)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s créé dans %s (%d fichiers).\n", stacks[stack], target, written)
			if key == "mp_sandbox_..." {
				fmt.Fprintln(out, "Mettez votre clé sandbox dans le fichier d'environnement, ou lancez d'abord `mercypay login`.")
			}
			fmt.Fprintf(out, "La suite : %s/README.md\n", target)
			return nil
		},
	}
	command.Flags().BoolVar(&force, "force", false, "écrire même si le dossier n'est pas vide")
	return command
}

func stackNames() []string {
	names := make([]string, 0, len(stacks))
	for name := range stacks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// writeStack copies one stack's templates into target, filling in the name and the key.
func writeStack(stack, target, key string, force bool) (int, error) {
	if entries, err := os.ReadDir(target); err == nil && len(entries) > 0 && !force {
		return 0, fmt.Errorf("%s existe et n'est pas vide : choisissez un autre dossier, ou --force", target)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}

	values := map[string]string{"Name": filepath.Base(target), "APIKey": key}
	root := path.Join("templates", stack)
	written := 0

	err := fs.WalkDir(templates, root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := templates.ReadFile(name)
		if err != nil {
			return err
		}
		filled, err := fill(name, content, values)
		if err != nil {
			return err
		}

		relative := strings.TrimPrefix(name, root+"/")
		dir, base := path.Split(relative)
		if dotted, ok := renamed[base]; ok {
			base = dotted
		}
		destination := filepath.Join(target, filepath.FromSlash(dir), base)

		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(base, ".env") {
			mode = 0o600
		}
		written++
		return os.WriteFile(destination, filled, mode)
	})
	return written, err
}

func fill(name string, content []byte, values map[string]string) ([]byte, error) {
	parsed, err := template.New(name).Delims("[[", "]]").Parse(string(content))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := parsed.Execute(&out, values); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
