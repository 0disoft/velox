package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/0disoft/velox/internal/cli"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	dependencies := cli.Dependencies{}
	if len(args) > 0 && args[0] == "build" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		dependencies.BuildContext = ctx
	}
	return cli.Run(args, dependencies)
}
