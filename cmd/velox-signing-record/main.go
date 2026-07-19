package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/0disoft/velox/internal/authenticode"
	"github.com/0disoft/velox/internal/signingrecord"
)

var verifyAuthenticodeDirectory = authenticode.VerifyDirectory

type commonFlags struct {
	unsignedDirectory *string
	signingInput      *string
	signedDirectory   *string
	releaseDirectory  *string
	releaseArchive    *string
	evidenceDirectory *string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "velox-signing-record: expected prepare, authenticode, dry-run, or verify")
		return 2
	}
	switch args[0] {
	case "prepare":
		return runPrepare(args[1:], stdout, stderr)
	case "authenticode":
		return runAuthenticode(args[1:], stdout, stderr)
	case "dry-run":
		return runDryRun(args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "velox-signing-record: unknown command %q\n", args[0])
		return 2
	}
}

func runAuthenticode(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("velox-signing-record authenticode", flag.ContinueOnError)
	flags.SetOutput(stderr)
	signedDirectory := flags.String("signed-dir", "", "directory containing provider-output velox.exe and velox-host.exe")
	expectedSubject := flags.String("expected-subject", "", "exact approved Authenticode publisher subject")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *signedDirectory == "" || *expectedSubject == "" {
		fmt.Fprintln(stderr, "velox-signing-record: authenticode requires --signed-dir and --expected-subject")
		return 2
	}
	result, err := verifyAuthenticodeDirectory(*signedDirectory, *expectedSubject)
	if err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	return 0
}

func runPrepare(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("velox-signing-record prepare", flag.ContinueOnError)
	flags.SetOutput(stderr)
	unsignedDirectory := flags.String("unsigned-dir", "", "directory containing unsigned velox.exe and velox-host.exe")
	out := flags.String("out", "", "output velox-signing-input.zip path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *unsignedDirectory == "" || *out == "" {
		fmt.Fprintln(stderr, "velox-signing-record: prepare requires --unsigned-dir and --out")
		return 2
	}
	result, err := signingrecord.PrepareSigningInput(*unsignedDirectory, *out)
	if err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	if err := json.NewEncoder(stdout).Encode(struct {
		SchemaVersion string                           `json:"schemaVersion"`
		Command       string                           `json:"command"`
		Publishable   bool                             `json:"publishable"`
		Result        signingrecord.SigningInputResult `json:"result"`
	}{SchemaVersion: "velox.signing-record-result/v1", Command: "prepare", Publishable: false, Result: result}); err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	return 0
}

func runDryRun(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("velox-signing-record dry-run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("out", "", "output signing record path")
	releaseVersion := flags.String("release-version", "", "Velox release version")
	repository := flags.String("source-repository", "https://github.com/0disoft/velox", "canonical source repository URL")
	commit := flags.String("source-commit", "", "lowercase 40-character source commit")
	tag := flags.String("source-tag", "", "immutable release tag")
	workflow := flags.String("source-workflow", "", "workflow identity including immutable ref")
	runID := flags.String("source-run-id", "", "GitHub Actions run identifier")
	providerName := flags.String("provider", signingrecord.ProviderSignPath, "signing provider identity")
	providerProject := flags.String("provider-project", "", "provider project identity")
	artifactConfiguration := flags.String("artifact-configuration", "", "provider artifact configuration identity")
	signingPolicy := flags.String("signing-policy", "", "provider signing policy identity")
	requestID := flags.String("request-id", "", "dry-run request identity")
	paths := addCommonFlags(flags)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *out == "" || *releaseVersion == "" || *commit == "" || *tag == "" || *workflow == "" || *runID == "" || *providerProject == "" || *artifactConfiguration == "" || *signingPolicy == "" || *requestID == "" || !paths.complete() {
		fmt.Fprintln(stderr, "velox-signing-record: dry-run metadata, evidence paths, and --out are required")
		return 2
	}
	record, err := signingrecord.BuildDryRun(signingrecord.DryRunOptions{
		ReleaseVersion: *releaseVersion,
		Source: signingrecord.Source{
			Repository: *repository,
			Commit:     *commit,
			Tag:        *tag,
			Workflow:   *workflow,
			RunID:      *runID,
		},
		Provider: signingrecord.Provider{
			Name:                  *providerName,
			Project:               *providerProject,
			ArtifactConfiguration: *artifactConfiguration,
			SigningPolicy:         *signingPolicy,
			RequestID:             *requestID,
		},
		Files: paths.files(),
	})
	if err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	result, err := signingrecord.Write(*out, record)
	if err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	if err := json.NewEncoder(stdout).Encode(struct {
		SchemaVersion string                    `json:"schemaVersion"`
		Command       string                    `json:"command"`
		Publishable   bool                      `json:"publishable"`
		Result        signingrecord.WriteResult `json:"result"`
	}{SchemaVersion: "velox.signing-record-result/v1", Command: "dry-run", Publishable: false, Result: result}); err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	return 0
}

func runVerify(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("velox-signing-record verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	recordPath := flags.String("record", "", "signing record path")
	paths := addCommonFlags(flags)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *recordPath == "" || !paths.complete() {
		fmt.Fprintln(stderr, "velox-signing-record: --record and all evidence paths are required")
		return 2
	}
	record, err := signingrecord.DecodeFile(*recordPath)
	if err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	if err := signingrecord.VerifyFiles(record, paths.files()); err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	if err := json.NewEncoder(stdout).Encode(struct {
		SchemaVersion string `json:"schemaVersion"`
		Command       string `json:"command"`
		Mode          string `json:"mode"`
		Publishable   bool   `json:"publishable"`
		Valid         bool   `json:"valid"`
	}{SchemaVersion: "velox.signing-record-result/v1", Command: "verify", Mode: record.Mode, Publishable: record.Publishable, Valid: true}); err != nil {
		fmt.Fprintln(stderr, "velox-signing-record:", err)
		return 6
	}
	return 0
}

func addCommonFlags(flags *flag.FlagSet) commonFlags {
	return commonFlags{
		unsignedDirectory: flags.String("unsigned-dir", "", "directory containing unsigned velox.exe and velox-host.exe"),
		signingInput:      flags.String("signing-input", "", "velox-signing-input.zip path"),
		signedDirectory:   flags.String("signed-dir", "", "directory containing provider-output velox.exe and velox-host.exe"),
		releaseDirectory:  flags.String("release-dir", "", "final release directory containing release-manifest.json"),
		releaseArchive:    flags.String("release-archive", "", "final velox-windows-x64.zip path"),
		evidenceDirectory: flags.String("evidence-dir", "", "directory containing checksums.sha256 and the final SPDX SBOM"),
	}
}

func (flags commonFlags) complete() bool {
	return *flags.unsignedDirectory != "" && *flags.signingInput != "" && *flags.signedDirectory != "" && *flags.releaseDirectory != "" && *flags.releaseArchive != "" && *flags.evidenceDirectory != ""
}

func (flags commonFlags) files() signingrecord.Files {
	return signingrecord.Files{
		UnsignedCLI:     filepath.Join(*flags.unsignedDirectory, "velox.exe"),
		UnsignedHost:    filepath.Join(*flags.unsignedDirectory, "velox-host.exe"),
		SigningInput:    *flags.signingInput,
		SignedCLI:       filepath.Join(*flags.signedDirectory, "velox.exe"),
		SignedHost:      filepath.Join(*flags.signedDirectory, "velox-host.exe"),
		ReleaseArchive:  *flags.releaseArchive,
		ReleaseManifest: filepath.Join(*flags.releaseDirectory, "release-manifest.json"),
		Checksums:       filepath.Join(*flags.evidenceDirectory, "checksums.sha256"),
		SBOM:            filepath.Join(*flags.evidenceDirectory, "velox-windows-x64.spdx.json"),
	}
}
