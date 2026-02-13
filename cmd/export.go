package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:    "export <shell>",
	Short:  "Emit export/unset commands (called by shell hook)",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			return
		}

		// Fast path: no .env and no active session → nothing to do
		_, statErr := os.Stat(filepath.Join(cwd, ".env"))
		hasEnv := statErr == nil
		hasSession := os.Getenv("_AUTOENV_ACTIVE") != ""
		if !hasEnv && !hasSession {
			return
		}

		// Lightweight bootstrap — sessions DB only, no Turso
		a, cc, err := bootstrapLight()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			return
		}
		defer cc.CloseAll()

		output, err := a.Export.Export(args[0], getShellPID(), cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			return
		}
		fmt.Print(output)
	},
}

func getShellPID() int {
	pid, _ := strconv.Atoi(os.Getenv("AUTOENV_SHELL_PID"))
	if pid != 0 {
		return pid
	}
	return os.Getppid()
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
