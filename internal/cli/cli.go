package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/0disoft/velox/internal/builder"
	"github.com/0disoft/velox/internal/buildinfo"
	"github.com/0disoft/velox/internal/buildplan"
	"github.com/0disoft/velox/internal/doctor"
	"github.com/0disoft/velox/internal/hostmeta"
	"github.com/0disoft/velox/internal/initializer"
	"github.com/0disoft/velox/internal/inspector"
	"github.com/0disoft/velox/internal/manifest"
	"github.com/0disoft/velox/internal/runtimeconfig"
	"github.com/0disoft/velox/internal/webview2"
)

type Dependencies struct {
	Stdout               io.Writer
	Stderr               io.Writer
	HostPath             string
	GOOS                 string
	GOARCH               string
	WebView2VersionProbe func() (string, error)
}

type Envelope struct {
	SchemaVersion int          `json:"schemaVersion"`
	OK            bool         `json:"ok"`
	Command       string       `json:"command"`
	Result        any          `json:"result,omitempty"`
	Error         *ErrorResult `json:"error,omitempty"`
	Diagnostics   []Diagnostic `json:"diagnostics"`
}

type ErrorResult struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Category string `json:"category"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type ValidateResult struct {
	ReleaseVersion string   `json:"releaseVersion"`
	AppID          string   `json:"appId"`
	AppVersion     string   `json:"appVersion"`
	Target         string   `json:"target"`
	AssetFiles     int      `json:"assetFiles"`
	AssetBytes     int64    `json:"assetBytes"`
	AssetSHA256    string   `json:"assetSha256"`
	HostSHA256     string   `json:"hostSha256"`
	Permissions    []string `json:"permissions"`
}

type BuildResult struct {
	ReleaseVersion string `json:"releaseVersion"`
	AppID          string `json:"appId"`
	Target         string `json:"target"`
	Directory      string `json:"directory"`
	Archive        string `json:"archive"`
	ArchiveBytes   int64  `json:"archiveBytes"`
	ArchiveSHA256  string `json:"archiveSha256"`
}

type VersionResult struct {
	Version          string   `json:"version"`
	ManifestVersions []int    `json:"manifestVersions"`
	RuntimeVersions  []int    `json:"runtimeVersions"`
	HostContracts    []int    `json:"hostContracts"`
	Targets          []string `json:"targets"`
}

func Run(args []string, dependencies Dependencies) int {
	if dependencies.Stdout == nil {
		dependencies.Stdout = os.Stdout
	}
	if dependencies.Stderr == nil {
		dependencies.Stderr = os.Stderr
	}
	if len(args) == 0 {
		printUsage(dependencies.Stderr)
		return 2
	}
	if args[0] == "--version" {
		fmt.Fprintln(dependencies.Stdout, buildinfo.Version)
		return 0
	}
	switch args[0] {
	case "validate":
		return runValidate(args[1:], dependencies)
	case "build":
		return runBuild(args[1:], dependencies)
	case "version":
		return runVersion(args[1:], dependencies)
	case "inspect":
		return runInspect(args[1:], dependencies)
	case "init":
		return runInit(args[1:], dependencies)
	case "doctor":
		return runDoctor(args[1:], dependencies)
	case "help", "--help", "-h":
		printUsage(dependencies.Stdout)
		return 0
	default:
		fmt.Fprintf(dependencies.Stderr, "velox: unknown command %q\n", args[0])
		printUsage(dependencies.Stderr)
		return 2
	}
}

func runInit(args []string, dependencies Dependencies) int {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(dependencies.Stderr)
	jsonOutput := flags.Bool("json", false, "emit one JSON document")
	quiet := flags.Bool("quiet", false, "suppress successful human output")
	if jsonRequested(args) {
		flags.SetOutput(io.Discard)
	}
	if err := flags.Parse(reorderPositionalArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return emitFailure(dependencies, "init", *jsonOutput || jsonRequested(args), 2, "USAGE_INVALID", "Command arguments are invalid.", err)
	}
	if flags.NArg() > 1 {
		return emitFailure(dependencies, "init", *jsonOutput, 2, "USAGE_INVALID", "Init accepts at most one directory.", errors.New("too many project directories"))
	}
	directory := "."
	if flags.NArg() == 1 {
		directory = flags.Arg(0)
	}
	result, err := initializer.Create(directory)
	if err != nil {
		return emitFailure(dependencies, "init", *jsonOutput, 6, "INIT_FAILED", "Project initialization failed.", err)
	}
	if *jsonOutput {
		emitJSON(dependencies.Stdout, Envelope{SchemaVersion: 1, OK: true, Command: "init", Result: result, Diagnostics: []Diagnostic{}})
	} else if !*quiet {
		fmt.Fprintf(dependencies.Stdout, "Initialized %s in %s\n", result.AppID, result.Directory)
	}
	return 0
}

type commonOptions struct {
	config  string
	target  string
	out     string
	json    bool
	quiet   bool
	verbose bool
}

func newFlagSet(command string, stderr io.Writer) (*flag.FlagSet, *commonOptions) {
	options := &commonOptions{}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.config, "config", "velox.json", "project manifest path")
	flags.StringVar(&options.target, "target", buildplan.TargetWindowsX64, "build target")
	flags.StringVar(&options.out, "out", "dist", "output root relative to the project")
	flags.BoolVar(&options.json, "json", false, "emit one JSON document")
	flags.BoolVar(&options.quiet, "quiet", false, "suppress successful human output")
	flags.BoolVar(&options.verbose, "verbose", false, "emit bounded diagnostic detail")
	return flags, options
}

func runDoctor(args []string, dependencies Dependencies) int {
	flags, options := newFlagSet("doctor", dependencies.Stderr)
	if jsonRequested(args) {
		flags.SetOutput(io.Discard)
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return emitFailure(dependencies, "doctor", options.json || jsonRequested(args), 2, "USAGE_INVALID", "Command arguments are invalid.", err)
	}
	if flags.NArg() != 0 {
		return emitFailure(dependencies, "doctor", options.json, 2, "USAGE_INVALID", "Doctor does not accept positional arguments.", errors.New("unexpected positional arguments"))
	}

	goos, goarch := dependencies.GOOS, dependencies.GOARCH
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	probe := dependencies.WebView2VersionProbe
	if probe == nil {
		probe = webview2.InstalledVersion
	}
	version, probeErr := probe()
	plan, planErr := createPlan(*options, dependencies.HostPath)
	result, failure := doctor.Evaluate(doctor.Evidence{
		GOOS: goos, GOARCH: goarch, WebView2Version: version,
		WebView2ProbeError: probeErr, Plan: plan, PlanError: planErr,
	})
	if failure != nil {
		if options.json {
			emitJSON(dependencies.Stdout, Envelope{
				SchemaVersion: 1, OK: false, Command: "doctor", Result: result,
				Error:       &ErrorResult{Code: failure.Code, Message: failure.Message},
				Diagnostics: []Diagnostic{{Code: failure.Code, Severity: "error", Category: category(failure.Code), Message: failure.Message}},
			})
		} else {
			printDoctor(dependencies.Stdout, result)
			fmt.Fprintf(dependencies.Stderr, "velox: %s: %s\n", failure.Code, failure.Message)
		}
		return failure.ExitCode
	}
	if options.json {
		emitJSON(dependencies.Stdout, Envelope{SchemaVersion: 1, OK: true, Command: "doctor", Result: result, Diagnostics: []Diagnostic{}})
	} else if !options.quiet {
		printDoctor(dependencies.Stdout, result)
	}
	return 0
}

func printDoctor(writer io.Writer, result doctor.Result) {
	for _, check := range result.Checks {
		fmt.Fprintf(writer, "%-7s %-9s %s\n", check.Status, check.Name, check.Message)
	}
}

func runValidate(args []string, dependencies Dependencies) int {
	flags, options := newFlagSet("validate", dependencies.Stderr)
	if jsonRequested(args) {
		flags.SetOutput(io.Discard)
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return emitFailure(dependencies, "validate", options.json || jsonRequested(args), 2, "USAGE_INVALID", "Command arguments are invalid.", err)
	}
	if flags.NArg() != 0 {
		return emitFailure(dependencies, "validate", options.json, 2, "USAGE_INVALID", "Validate does not accept positional arguments.", errors.New("unexpected positional arguments"))
	}
	plan, err := createPlan(*options, dependencies.HostPath)
	if err != nil {
		return emitPlanError(dependencies, "validate", options.json, err)
	}
	result := validateResult(plan)
	if options.json {
		emitJSON(dependencies.Stdout, Envelope{SchemaVersion: 1, OK: true, Command: "validate", Result: result, Diagnostics: []Diagnostic{}})
	} else if !options.quiet {
		fmt.Fprintf(dependencies.Stdout, "Valid: %s (%d assets, %d bytes)\n", result.AppID, result.AssetFiles, result.AssetBytes)
	}
	return 0
}

func runBuild(args []string, dependencies Dependencies) int {
	flags, options := newFlagSet("build", dependencies.Stderr)
	if jsonRequested(args) {
		flags.SetOutput(io.Discard)
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return emitFailure(dependencies, "build", options.json || jsonRequested(args), 2, "USAGE_INVALID", "Command arguments are invalid.", err)
	}
	if flags.NArg() != 0 {
		return emitFailure(dependencies, "build", options.json, 2, "USAGE_INVALID", "Build does not accept positional arguments.", errors.New("unexpected positional arguments"))
	}
	plan, err := createPlan(*options, dependencies.HostPath)
	if err != nil {
		return emitPlanError(dependencies, "build", options.json, err)
	}
	result, err := builder.Build(plan)
	if err != nil {
		return emitFailure(dependencies, "build", options.json, 6, "PACKAGING_FAILED", "Application packaging failed.", err)
	}
	snapshot := plan.Snapshot()
	presented := BuildResult{
		ReleaseVersion: result.Report.ReleaseVersion, AppID: result.Report.App.ID, Target: result.Report.Target,
		Directory:    safePath(snapshot.Manifest.ProjectRoot, result.DirectoryPath),
		Archive:      safePath(snapshot.Manifest.ProjectRoot, result.ArchivePath),
		ArchiveBytes: result.ArchiveSize, ArchiveSHA256: result.ArchiveSHA256,
	}
	if options.json {
		emitJSON(dependencies.Stdout, Envelope{SchemaVersion: 1, OK: true, Command: "build", Result: presented, Diagnostics: []Diagnostic{}})
	} else if !options.quiet {
		fmt.Fprintf(dependencies.Stdout, "Built %s\nArchive: %s\nSHA-256: %s\n", presented.AppID, presented.Archive, presented.ArchiveSHA256)
	}
	return 0
}

func runVersion(args []string, dependencies Dependencies) int {
	flags := flag.NewFlagSet("version", flag.ContinueOnError)
	flags.SetOutput(dependencies.Stderr)
	jsonOutput := flags.Bool("json", false, "emit one JSON document")
	quiet := flags.Bool("quiet", false, "suppress successful human output")
	if jsonRequested(args) {
		flags.SetOutput(io.Discard)
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return emitFailure(dependencies, "version", *jsonOutput || jsonRequested(args), 2, "USAGE_INVALID", "Command arguments are invalid.", err)
	}
	if flags.NArg() != 0 {
		return emitFailure(dependencies, "version", *jsonOutput, 2, "USAGE_INVALID", "Version does not accept positional arguments.", errors.New("unexpected positional arguments"))
	}
	result := VersionResult{
		Version: buildinfo.Version, ManifestVersions: []int{manifest.Version},
		RuntimeVersions: []int{runtimeconfig.Version}, HostContracts: []int{hostmeta.ContractVersion}, Targets: []string{buildplan.TargetWindowsX64},
	}
	if *jsonOutput {
		emitJSON(dependencies.Stdout, Envelope{SchemaVersion: 1, OK: true, Command: "version", Result: result, Diagnostics: []Diagnostic{}})
	} else if !*quiet {
		fmt.Fprintf(dependencies.Stdout, "Velox %s\nManifest: v%d\nRuntime: v%d\nTarget: %s\n", buildinfo.Version, manifest.Version, runtimeconfig.Version, buildplan.TargetWindowsX64)
	}
	return 0
}

func runInspect(args []string, dependencies Dependencies) int {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(dependencies.Stderr)
	jsonOutput := flags.Bool("json", false, "emit one JSON document")
	quiet := flags.Bool("quiet", false, "suppress successful human output")
	if jsonRequested(args) {
		flags.SetOutput(io.Discard)
	}
	if err := flags.Parse(reorderPositionalArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return emitFailure(dependencies, "inspect", *jsonOutput || jsonRequested(args), 2, "USAGE_INVALID", "Command arguments are invalid.", err)
	}
	if flags.NArg() != 1 {
		return emitFailure(dependencies, "inspect", *jsonOutput, 2, "USAGE_INVALID", "Inspect requires one artifact path.", errors.New("expected one artifact path"))
	}
	result, err := inspector.Inspect(flags.Arg(0))
	if err != nil {
		return emitFailure(dependencies, "inspect", *jsonOutput, 3, "ARTIFACT_INVALID", "Artifact validation failed.", err)
	}
	if *jsonOutput {
		emitJSON(dependencies.Stdout, Envelope{SchemaVersion: 1, OK: true, Command: "inspect", Result: result, Diagnostics: []Diagnostic{}})
	} else if !*quiet {
		fmt.Fprintf(dependencies.Stdout, "Valid %s artifact: %s %s (%d files, %d bytes)\n", result.Kind, result.App.ID, result.App.Version, result.PortableFiles, result.PortableBytes)
	}
	return 0
}

func createPlan(options commonOptions, hostPath string) (buildplan.Plan, error) {
	if hostPath == "" {
		executable, err := os.Executable()
		if err != nil {
			return buildplan.Plan{}, &buildplan.Error{Kind: buildplan.ErrorHost, Err: fmt.Errorf("locate Velox executable: %w", err)}
		}
		hostPath = filepath.Join(filepath.Dir(executable), "velox-host.exe")
	}
	return buildplan.Create(buildplan.Options{
		ManifestPath:     options.config,
		HostPath:         hostPath,
		HostMetadataPath: filepath.Join(filepath.Dir(hostPath), "velox-host.json"),
		OutputRoot:       options.out,
		Target:           options.target,
	})
}

func validateResult(plan buildplan.Plan) ValidateResult {
	snapshot := plan.Snapshot()
	return ValidateResult{
		ReleaseVersion: snapshot.HostMetadata.ReleaseVersion, AppID: snapshot.Manifest.App.ID, AppVersion: snapshot.Manifest.App.Version, Target: snapshot.Target,
		AssetFiles: len(snapshot.Assets.Files), AssetBytes: snapshot.Assets.TotalBytes,
		AssetSHA256: snapshot.Assets.Digest, HostSHA256: snapshot.HostSHA256,
		Permissions: append([]string{}, snapshot.Manifest.Security.Permissions...),
	}
}

func emitPlanError(dependencies Dependencies, command string, jsonOutput bool, err error) int {
	var planError *buildplan.Error
	if !errors.As(err, &planError) {
		return emitFailure(dependencies, command, jsonOutput, 10, "INTERNAL", "Unexpected internal failure.", err)
	}
	switch planError.Kind {
	case buildplan.ErrorManifest, buildplan.ErrorConfig:
		return emitFailure(dependencies, command, jsonOutput, 2, "MANIFEST_INVALID", "Project manifest is invalid.", err)
	case buildplan.ErrorAsset:
		return emitFailure(dependencies, command, jsonOutput, 3, "ASSET_INVALID", "Project assets are invalid.", err)
	case buildplan.ErrorHost:
		return emitFailure(dependencies, command, jsonOutput, 4, "HOST_INCOMPATIBLE", "Host template is unavailable or incompatible.", err)
	default:
		return emitFailure(dependencies, command, jsonOutput, 10, "INTERNAL", "Unexpected internal failure.", err)
	}
}

func emitFailure(dependencies Dependencies, command string, jsonOutput bool, exitCode int, code, message string, detail error) int {
	if jsonOutput {
		emitJSON(dependencies.Stdout, Envelope{
			SchemaVersion: 1, OK: false, Command: command,
			Error:       &ErrorResult{Code: code, Message: message},
			Diagnostics: []Diagnostic{{Code: code, Severity: "error", Category: category(code), Message: message}},
		})
	} else {
		fmt.Fprintf(dependencies.Stderr, "velox: %s: %s\n", code, safeMessage(detail))
	}
	return exitCode
}

func emitJSON(writer io.Writer, value Envelope) {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}

func category(code string) string {
	switch {
	case strings.HasPrefix(code, "MANIFEST"), strings.HasPrefix(code, "USAGE"):
		return "configuration"
	case strings.HasPrefix(code, "ASSET"):
		return "asset"
	case strings.HasPrefix(code, "ARTIFACT"):
		return "artifact"
	case strings.HasPrefix(code, "INIT"):
		return "filesystem"
	case strings.HasPrefix(code, "RUNTIME"):
		return "runtime"
	case strings.HasPrefix(code, "HOST"):
		return "host"
	case strings.HasPrefix(code, "PACKAGING"):
		return "packaging"
	default:
		return "internal"
	}
}

func safeMessage(err error) string {
	if err == nil {
		return "operation failed"
	}
	message := err.Error()
	if volume := filepath.VolumeName(message); volume != "" {
		return "operation failed at a local path"
	}
	return message
}

func safePath(projectRoot, path string) string {
	relative, err := filepath.Rel(projectRoot, path)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(relative)
	}
	return filepath.Base(path)
}

func jsonRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || strings.HasPrefix(arg, "--json=") {
			return true
		}
	}
	return false
}

func reorderPositionalArgs(args []string) []string {
	ordered := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			ordered = append(ordered, arg)
		}
	}
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			ordered = append(ordered, arg)
		}
	}
	return ordered
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "Usage: velox <init|validate|doctor|build|inspect|version> [options]")
}
