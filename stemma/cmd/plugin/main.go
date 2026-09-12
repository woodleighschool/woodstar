// Plugin serves the woodstar.munki reconciliation operation.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/woodleighschool/stemma/plugin"
	"github.com/woodleighschool/woodstar/stemma/internal/destination"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	registry := plugin.New("woodstar", version)
	if err := destination.Register(registry); err != nil {
		return err
	}
	return plugin.Serve(ctx, os.Stdin, os.Stdout, registry)
}
