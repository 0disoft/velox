package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/0disoft/actutum/internal/releasebundle"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("actutum-release", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	cliPath := flags.String("cli", "", "path to the prebuilt Actutum CLI")
	hostPath := flags.String("host", "", "path to the prebuilt Actutum host")
	sourceRoot := flags.String("source-root", ".", "repository source root")
	outputRoot := flags.String("out", "dist/release", "release output root")
	jsonOutput := flags.Bool("json", false, "emit JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *cliPath == "" || *hostPath == "" || flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "actutum-release: --cli and --host are required")
		return 2
	}
	result, err := releasebundle.Build(releasebundle.Options{CLIPath: *cliPath, HostPath: *hostPath, SourceRoot: *sourceRoot, OutputRoot: *outputRoot})
	if err != nil {
		fmt.Fprintf(os.Stderr, "actutum-release: %v\n", err)
		return 6
	}
	if *jsonOutput {
		_ = json.NewEncoder(os.Stdout).Encode(result)
	} else {
		fmt.Printf("Release bundle: %s\nSHA-256: %s\n", result.Archive, result.ArchiveSHA256)
	}
	return 0
}
