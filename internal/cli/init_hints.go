package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
)

func printInitNextSteps(writer io.Writer, directory, goos string) {
	if goos == "" {
		goos = runtime.GOOS
	}
	config := filepath.ToSlash(filepath.Join(directory, "velox.json"))
	shell := "POSIX shell"
	escaped := strings.ReplaceAll(config, "'", "'\"'\"'")
	if goos == "windows" {
		shell = "PowerShell"
		escaped = strings.ReplaceAll(config, "'", "''")
	}
	argument := "'" + escaped + "'"
	fmt.Fprintf(writer, "\nNext steps (%s):\n  velox run --config %s\n  velox build --config %s\n", shell, argument, argument)
}
