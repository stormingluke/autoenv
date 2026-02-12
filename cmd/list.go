package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered projects",
	Run: func(cmd *cobra.Command, args []string) {
		b, err := bootstrap()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer b.cc.CloseAll()

		projects, err := b.app.List.ListProjects()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		if len(projects) == 0 {
			fmt.Println("No registered projects.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tPATH\tCREATED")
		for _, p := range projects {
			name := p.Name
			if name == "" {
				name = "-"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", name, p.Path, p.CreatedAt)
		}
		_ = w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
