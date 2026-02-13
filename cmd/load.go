package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/stormingluke/autoenv/internal/adapter/envfile"
	"github.com/stormingluke/autoenv/internal/adapter/shell"
)

var loadProject string

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load .env from a directory and output export commands",
	Long:  `Load environment variables from a .env file. Use with eval: eval "$(autoenv load -p /path/to/dir)"`,
	Run: func(cmd *cobra.Command, args []string) {
		path := loadProject
		if path == "" {
			path, _ = os.Getwd()
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		loader := envfile.NewLoader()
		envFile, err := loader.Load(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		if envFile == nil {
			fmt.Fprintf(os.Stderr, "autoenv: no .env file in %s\n", absPath)
			os.Exit(1)
		}

		renderer := shell.NewRenderer()
		fmt.Print(renderer.FormatExports("zsh", envFile.Values))
		fmt.Fprintf(os.Stderr, "autoenv: loaded %d variables from %s\n", len(envFile.Values), absPath)
	},
}

func init() {
	loadCmd.Flags().StringVarP(&loadProject, "project", "p", "", "Path to directory containing .env (defaults to current directory)")
	rootCmd.AddCommand(loadCmd)
}
