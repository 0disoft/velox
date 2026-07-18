package main

import (
	"os"

	"github.com/0disoft/actutum/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Dependencies{}))
}
