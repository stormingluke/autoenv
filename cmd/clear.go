package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Unset all autoenv-loaded vars for current session",
	Long:  `Unset all autoenv-loaded environment variables. Use with eval: eval "$(autoenv clear)"`,
	Run: func(cmd *cobra.Command, args []string) {
		a, cc, err := bootstrapLight()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer cc.CloseAll()

		output, err := a.Clear.Clear("zsh", getShellPID())
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
