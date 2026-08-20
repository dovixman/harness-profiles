package main

import (
	"os"

	"github.com/dovixman/harness-profiles/internal/adapters/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
