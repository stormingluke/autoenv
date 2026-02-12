package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stormingluke/autoenv/internal/config"
	handler "github.com/stormingluke/autoenv/internal/export"
	"github.com/stormingluke/autoenv/internal/store"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Unset all autoenv-loaded vars for current session",
	Long:  `Unset all autoenv-loaded environment variables. Use with eval: eval "$(autoenv clear)"`,
	Run: func(cmd *cobra.Command, args []string) {
		shellPID := getShellPID()

		cfg := config.Load()
		if err := cfg.EnsureDir(); err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		sessDB, err := store.OpenSessionsDB(cfg.SessionsDBPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer sessDB.Close()

		// ProjectRepo not needed for clear, but handler requires it
		turso, err := store.OpenTurso(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer turso.Close()

		h := handler.NewHandler(
			store.NewProjectRepo(turso.DB),
			store.NewSessionRepo(sessDB),
		)

		output, err := h.Clear("zsh", shellPID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
