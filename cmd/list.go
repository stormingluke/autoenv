package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/stormingluke/autoenv/internal/config"
	"github.com/stormingluke/autoenv/internal/store"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered projects",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		if err := cfg.EnsureDir(); err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		turso, err := store.OpenTurso(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer turso.Close()

		repo := store.NewProjectRepo(turso.DB)
		projects, err := repo.ListAll()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		if len(projects) == 0 {
			fmt.Println("No registered projects.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPATH\tCREATED")
		for _, p := range projects {
			name := p.Name
			if name == "" {
				name = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", name, p.Path, p.CreatedAt)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
