package main

import "github.com/stormingluke/autoenv/cmd"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute()
}
