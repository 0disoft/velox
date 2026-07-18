package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/0disoft/actutum/internal/benchmarker"
	"github.com/0disoft/actutum/internal/runtimeconfig"
	"github.com/0disoft/actutum/internal/webview2"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	timeline := benchmarker.NewTimelineRecorder(os.Getenv(benchmarker.PipeEnvironment) != "")
	shutdownTimeline := benchmarker.NewShutdownTimelineRecorder(os.Getenv(benchmarker.PipeEnvironment) != "")
	flags := flag.NewFlagSet("actutum-host", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("config", "actutum.runtime.json", "path to the external runtime configuration")
	debug := flags.Bool("debug", false, "enable WebView2 development tools")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	cfg, err := runtimeconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "actutum-host: %v\n", err)
		return 2
	}
	timeline.Mark("config-loaded")

	dataPath := os.Getenv("ACTUTUM_DATA_DIR")
	if dataPath == "" {
		dataPath, err = defaultDataPath(cfg.App.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "actutum-host: %v\n", err)
			return 6
		}
	}

	var runtime *webview2.Runtime
	audit := newPolicyAudit(os.Getenv("ACTUTUM_BENCH_POLICY_AUDIT") == "1")
	timeline.Mark("runtime-open-started")
	runtime, err = webview2.Open(webview2.Config{
		Title:                   cfg.App.Name,
		AppID:                   cfg.App.ID,
		AppVersion:              cfg.App.Version,
		Permissions:             cfg.Security.Permissions,
		Width:                   cfg.Window.Width,
		Height:                  cfg.Window.Height,
		DataPath:                dataPath,
		BrowserExecutableFolder: os.Getenv("ACTUTUM_BENCH_WEBVIEW2_BROWSER_DIR"),
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
		if os.Getenv("ACTUTUM_BENCH_EXIT_AFTER_READY") == "1" {
			runtime.Close()
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "actutum-host: %v\n", err)
		if errors.Is(err, webview2.ErrRuntimeUnavailable) {
			return 5
		}
		return 6
	}
	timeline.Mark("runtime-opened")
	audit.complete = func() {
		browserProcessID, err := runtime.BrowserProcessID()
		if err != nil {
			fmt.Fprintf(os.Stderr, "actutum-host: policy audit browser process: %v\n", err)
			return
		}
		if err := benchmarker.NotifyReady("security-ok", browserProcessID); err != nil {
			fmt.Fprintf(os.Stderr, "actutum-host: policy audit marker: %v\n", err)
		}
		if os.Getenv("ACTUTUM_BENCH_EXIT_AFTER_READY") == "1" {
			runtime.Close()
		}
	}
	defer runtime.Close()

	runtime.Run()
	if err := shutdownTimeline.Emit(os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "actutum-host: %v\n", err)
		return 6
	}
	return 0
}

func defaultDataPath(appID string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve local application data: %w", err)
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("local application data path is not absolute")
	}
	return filepath.Join(base, "Actutum", "profiles", appID), nil
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
		fmt.Fprintf(os.Stderr, "actutum-policy-blocked: %s\n", kind)
		a.tryComplete()
	}
}

func (a *policyAudit) markIPCReady(phase string) {
	if !a.enabled {
		return
	}
	if phase != "ipc-ok" {
		fmt.Fprintf(os.Stderr, "actutum-ipc-audit: %s\n", phase)
		return
	}
	a.ipcReady = true
	fmt.Fprintln(os.Stderr, "actutum-policy-blocked: ipc-contract-passed")
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
