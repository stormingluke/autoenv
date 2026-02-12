package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:    "export <shell>",
	Short:  "Emit export/unset commands (called by shell hook)",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		b, err := bootstrap()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			return
		}
		defer b.cc.CloseAll()

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			return
		}

		output, err := b.app.Export.Export(args[0], getShellPID(), cwd)
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
