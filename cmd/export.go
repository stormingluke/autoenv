package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/stormingluke/autoenv/internal/config"
	handler "github.com/stormingluke/autoenv/internal/export"
	"github.com/stormingluke/autoenv/internal/store"
)

var exportCmd = &cobra.Command{
	Use:    "export <shell>",
	Short:  "Emit export/unset commands (called by shell hook)",
	Args:   cobra.ExactArgs(1),
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		shellType := args[0]
		shellPID := getShellPID()

		cfg := config.Load()
		if err := cfg.EnsureDir(); err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			return
		}

		turso, err := store.OpenTurso(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			return
		}
		defer turso.Close()

		sessDB, err := store.OpenSessionsDB(cfg.SessionsDBPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "autoenv: %v\n", err)
			return
		}
		defer sessDB.Close()

		h := handler.NewHandler(
			store.NewProjectRepo(turso.DB),
			store.NewSessionRepo(sessDB),
		)

		output, err := h.Export(shellType, shellPID)
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
