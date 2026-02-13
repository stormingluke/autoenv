package cmd

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/stormingluke/autoenv/internal/adapter/envfile"
	"github.com/stormingluke/autoenv/internal/adapter/shell"
)

// Version is set by main via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "autoenv",
	Short: "Automatically load .env files into shell sessions",
	Long:  "Autoenv loads .env files when you enter registered project directories and unsets them when you leave.\n\nRun without arguments to load .env from the current directory.",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		loader := envfile.NewLoader()
		envFile, err := loader.Load(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		if envFile == nil {
			fmt.Fprintf(os.Stderr, "autoenv: no .env file in %s\n", cwd)
			os.Exit(1)
		}

		_ = godotenv.Load(envFile.Path)

		renderer := shell.NewRenderer()
		fmt.Print(renderer.FormatExports("zsh", envFile.Values))
		fmt.Fprintf(os.Stderr, "autoenv: loaded %d variables from %s\n", len(envFile.Values), cwd)
	},
}

func Execute() {
	rootCmd.Version = Version
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
