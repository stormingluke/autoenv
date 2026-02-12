package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var loadProject string

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Register a project and output export commands for its .env",
	Long:  `Register a project directory and output export commands. Use with eval: eval "$(autoenv load --project /path/to/project)"`,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := bootstrap()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}
		defer b.cc.CloseAll()

		projectPath := loadProject
		if projectPath == "" {
			projectPath, _ = os.Getwd()
		}

		absPath, err := filepath.Abs(projectPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		name := filepath.Base(absPath)

		output, err := b.app.Load.LoadProject("zsh", getShellPID(), absPath, name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "autoenv: registered project %q (%s)\n", name, absPath)
		fmt.Print(output)
	},
}

func init() {
	loadCmd.Flags().StringVarP(&loadProject, "project", "p", "", "Project path (defaults to current directory)")
	rootCmd.AddCommand(loadCmd)
}
