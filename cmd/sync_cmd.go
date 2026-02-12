package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stormingluke/autoenv/internal/config"
	"github.com/stormingluke/autoenv/internal/store"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Force sync projects database with Turso cloud",
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

		if err := turso.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: sync failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Sync complete.")
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
