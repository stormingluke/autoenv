package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var syncDB bool

var syncCmd = &cobra.Command{
	Use:   "sync [target]",
	Short: "Sync secrets to external targets or force Turso DB sync",
	Long: `Sync .env secrets to external targets like GitHub Actions.

Examples:
  autoenv sync github.com/stormingluke/stormingplatform   # full target
  autoenv sync stormingplatform                            # uses default owner
  autoenv sync --db                                        # Turso cloud sync`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := bootstrap()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer b.cc.CloseAll()

		if syncDB {
			if err := b.turso.Sync(); err != nil {
				fmt.Fprintf(os.Stderr, "autoenv: sync failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Turso sync complete.")
			return
		}

		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "autoenv: target required (e.g., autoenv sync github.com/owner/repo) or use --db for Turso sync")
			os.Exit(1)
		}

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		if err := b.app.Sync.SyncSecrets(cwd, args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: sync failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Secrets synced to %s\n", args[0])
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncDB, "db", false, "Force Turso cloud database sync")
	rootCmd.AddCommand(syncCmd)
}
