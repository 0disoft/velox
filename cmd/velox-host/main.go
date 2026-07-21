package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/0disoft/velox/internal/benchmarker"
	"github.com/0disoft/velox/internal/runtimeconfig"
	"github.com/0disoft/velox/internal/webview2"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	benchmark := benchmarkOptionsFromEnvironment(os.Getenv)
	timeline := benchmarker.NewTimelineRecorder(benchmark.pipeConfigured)
	shutdownTimeline := benchmarker.NewShutdownTimelineRecorder(benchmark.pipeConfigured)
	configDefault, err := defaultConfigPath(os.Executable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "velox-host: %v\n", err)
		return 6
	}
	flags := flag.NewFlagSet("velox-host", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("config", configDefault, "path to the external runtime configuration")
	debug := flags.Bool("debug", false, "enable WebView2 development tools")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	cfg, err := runtimeconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "velox-host: %v\n", err)
		return 2
	}
	timeline.Mark("config-loaded")

	dataPath := os.Getenv("VELOX_DATA_DIR")
	if dataPath == "" {
		dataPath, err = defaultDataPath(cfg.App.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "velox-host: %v\n", err)
			return 6
		}
	}

	var runtime *webview2.Runtime
	audit := newPolicyAudit(benchmark.policyAudit)
	timeline.Mark("runtime-open-started")
	runtime, err = webview2.Open(webview2.Config{
		Title:                   cfg.App.Name,
		AppID:                   cfg.App.ID,
		AppVersion:              cfg.App.Version,
		Permissions:             cfg.Security.Permissions,
		Width:                   cfg.Window.Width,
		Height:                  cfg.Window.Height,
		DataPath:                dataPath,
		BrowserExecutableFolder: benchmark.browserExecutableFolder,
		AssetRoot:               cfg.AssetRoot,
		EntryPath:               cfg.EntryPath,
		Debug:                   *debug,
		PolicyBlocked:           audit.record,
		StartupPhase:            timeline.Mark,
		ShutdownPhase:           shutdownTimeline.Mark,
	}, func(phase string) error {
		if audit.enabled {
			audit.markIPCReady(phase)
			return nil
		}
		if !benchmark.enabled {
			return nil
		}
		browserProcessID, err := runtime.BrowserProcessID()
		if err != nil {
			return fmt.Errorf("read WebView2 browser process ID: %w", err)
		}
		timeline.Mark(phase)
		if err := benchmarker.NotifyReady(phase, browserProcessID); err != nil {
			return err
		}
		if err := timeline.Emit(os.Stderr); err != nil {
			return err
		}
		if benchmark.exitAfterReady {
			runtime.Close()
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "velox-host: %v\n", err)
		if errors.Is(err, webview2.ErrRuntimeUnavailable) {
			return 5
		}
		return 6
	}
	timeline.Mark("runtime-opened")
	audit.complete = func() {
		browserProcessID, err := runtime.BrowserProcessID()
		if err != nil {
			fmt.Fprintf(os.Stderr, "velox-host: policy audit browser process: %v\n", err)
			return
		}
		if err := benchmarker.NotifyReady("security-ok", browserProcessID); err != nil {
			fmt.Fprintf(os.Stderr, "velox-host: policy audit marker: %v\n", err)
		}
		if benchmark.exitAfterReady {
			runtime.Close()
		}
	}
	defer runtime.Close()

	runtime.Run()
	if err := shutdownTimeline.Emit(os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "velox-host: %v\n", err)
		return 6
	}
	return 0
}

func defaultConfigPath(executable func() (string, error)) (string, error) {
	executablePath, err := executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	if !filepath.IsAbs(executablePath) {
		return "", errors.New("executable path is not absolute")
	}
	return filepath.Join(filepath.Dir(executablePath), "velox.runtime.json"), nil
}

type benchmarkOptions struct {
	enabled                 bool
	pipeConfigured          bool
	policyAudit             bool
	exitAfterReady          bool
	browserExecutableFolder string
}

func benchmarkOptionsFromEnvironment(getenv func(string) string) benchmarkOptions {
	if getenv("VELOX_BENCH_MODE") != "1" {
		return benchmarkOptions{}
	}
	return benchmarkOptions{
		enabled:                 true,
		pipeConfigured:          getenv(benchmarker.PipeEnvironment) != "",
		policyAudit:             getenv("VELOX_BENCH_POLICY_AUDIT") == "1",
		exitAfterReady:          getenv("VELOX_BENCH_EXIT_AFTER_READY") == "1",
		browserExecutableFolder: getenv("VELOX_BENCH_WEBVIEW2_BROWSER_DIR"),
	}
}

func defaultDataPath(appID string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve local application data: %w", err)
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("local application data path is not absolute")
	}
	return filepath.Join(base, "Velox", "profiles", appID), nil
}

type policyAudit struct {
	enabled  bool
	blocked  map[string]bool
	ipcReady bool
	complete func()
	reported bool
}

func newPolicyAudit(enabled bool) *policyAudit {
	return &policyAudit{enabled: enabled, blocked: make(map[string]bool)}
}

func (a *policyAudit) record(kind string) {
	if a.enabled {
		a.blocked[kind] = true
		fmt.Fprintf(os.Stderr, "velox-policy-blocked: %s\n", kind)
		a.tryComplete()
	}
}

func (a *policyAudit) markIPCReady(phase string) {
	if !a.enabled {
		return
	}
	if phase != "ipc-ok" {
		fmt.Fprintf(os.Stderr, "velox-ipc-audit: %s\n", phase)
		return
	}
	a.ipcReady = true
	fmt.Fprintln(os.Stderr, "velox-policy-blocked: ipc-contract-passed")
	a.tryComplete()
}

func (a *policyAudit) tryComplete() {
	if !a.reported && len(a.missing()) == 0 && a.complete != nil {
		a.reported = true
		a.complete()
	}
}

func (a *policyAudit) missing() []string {
	required := []string{
		webview2.PolicyNavigation,
		webview2.PolicyFrameNavigation,
		webview2.PolicyNewWindow,
		webview2.PolicyDownload,
		webview2.PolicyPermission,
	}
	missing := make([]string, 0, len(required))
	for _, kind := range required {
		if !a.blocked[kind] {
			missing = append(missing, kind)
		}
	}
	if !a.ipcReady {
		missing = append(missing, "ipc-contract")
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return missing
}
