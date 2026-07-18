package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/0disoft/velox/internal/releaseevidence"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("velox-release-evidence", flag.ContinueOnError)
	directory := flags.String("release-directory", "", "path to the unpacked release directory")
	archive := flags.String("release-archive", "", "path to the deterministic release ZIP")
	output := flags.String("out", "dist/release-evidence", "release evidence output root")
	repository := flags.String("source-repository", "https://github.com/0disoft/velox", "canonical source repository URL")
	commit := flags.String("source-commit", "", "lowercase 40-character source commit")
	invocation := flags.String("invocation-id", "", "stable build invocation identifier")
	created := flags.String("created-at", "", "RFC3339 source timestamp")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *directory == "" || *archive == "" || *commit == "" || *invocation == "" || *created == "" {
		fmt.Fprintln(os.Stderr, "velox-release-evidence: release paths, source commit, invocation ID, and created time are required")
		return 2
	}
	createdAt, err := time.Parse(time.RFC3339, *created)
	if err != nil {
		fmt.Fprintln(os.Stderr, "velox-release-evidence: invalid --created-at")
		return 2
	}
	result, err := releaseevidence.Build(releaseevidence.Options{ReleaseDirectory: *directory, ReleaseArchive: *archive, OutputRoot: *output, SourceRepository: *repository, SourceCommit: *commit, InvocationID: *invocation, CreatedAt: createdAt})
	if err != nil {
		fmt.Fprintln(os.Stderr, "velox-release-evidence:", err)
		return 6
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, "velox-release-evidence:", err)
		return 6
	}
	return 0
}
