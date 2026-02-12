package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "autoenv",
	Short: "Automatically load .env files into shell sessions",
	Long:  "Autoenv loads .env files when you enter registered project directories and unsets them when you leave.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
