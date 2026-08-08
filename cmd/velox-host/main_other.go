//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "velox-host: unsupported platform; the production host requires Windows x64")
	os.Exit(5)
}
