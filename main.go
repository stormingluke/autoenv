package main

import (
	"fmt"

	"github.com/stormingluke/autoenv/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	cmd.Execute()
}
