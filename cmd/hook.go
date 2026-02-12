package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stormingluke/autoenv/internal/adapter/shell"
)

var hookCmd = &cobra.Command{
	Use:   "hook <shell>",
	Short: "Output shell hook code to add to your shell config",
	Long:  `Output shell hook code. Add eval "$(autoenv hook zsh)" to your .zshrc.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		script, err := shell.HookScript(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(script)
	},
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
