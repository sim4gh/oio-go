package main

import (
	"github.com/sim4gh/oio-go/internal/cli"
	"github.com/sim4gh/oio-go/internal/config"
)

func main() {
	// Load config on startup
	config.Load()

	// Execute CLI
	cli.Execute()
}
