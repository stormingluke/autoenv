package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Manage autoenv defaults",
	Long: `Manage autoenv defaults that are stored in Turso and synced across machines.

Examples:
  autoenv configure set github.default_owner stormingluke
  autoenv configure get github.default_owner
  autoenv configure list`,
}

var configureSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a default value",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		b, err := bootstrap()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer b.cc.CloseAll()

		if err := b.app.Configure.Set(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s = %s\n", args[0], args[1])
	},
}

var configureGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a default value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		b, err := bootstrap()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer b.cc.CloseAll()

		value, err := b.app.Configure.Get(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(value)
	},
}

var configureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all defaults",
	Run: func(cmd *cobra.Command, args []string) {
		b, err := bootstrap()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer b.cc.CloseAll()

		settings, err := b.app.Configure.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		if len(settings) == 0 {
			fmt.Println("No defaults configured.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "KEY\tVALUE")
		for _, s := range settings {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", s.Key, s.Value)
		}
		_ = w.Flush()
	},
}

func init() {
	configureCmd.AddCommand(configureSetCmd)
	configureCmd.AddCommand(configureGetCmd)
	configureCmd.AddCommand(configureListCmd)
	rootCmd.AddCommand(configureCmd)
}
