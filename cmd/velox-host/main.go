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
	flags := flag.NewFlagSet("velox-host", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("config", "velox.runtime.json", "path to the external runtime configuration")
	debug := flags.Bool("debug", false, "enable WebView2 development tools")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	cfg, err := runtimeconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "velox-host: %v\n", err)
		return 2
	}

	dataPath := os.Getenv("VELOX_DATA_DIR")
	if dataPath == "" {
		dataPath = filepath.Join(os.TempDir(), "velox", cfg.App.ID)
	}

	var runtime *webview2.Runtime
	audit := newPolicyAudit(os.Getenv("VELOX_BENCH_POLICY_AUDIT") == "1")
	runtime, err = webview2.Open(webview2.Config{
		Title:                   cfg.App.Name,
		AppID:                   cfg.App.ID,
		Width:                   cfg.Window.Width,
		Height:                  cfg.Window.Height,
		DataPath:                dataPath,
		BrowserExecutableFolder: os.Getenv("VELOX_BENCH_WEBVIEW2_BROWSER_DIR"),
		AssetRoot:               cfg.AssetRoot,
		EntryPath:               cfg.EntryPath,
		Debug:                   *debug,
		PolicyBlocked:           audit.record,
	}, func(phase string) error {
		if audit.enabled {
			return nil
		}
		if err := benchmarker.NotifyReady(phase); err != nil {
			return err
		}
		if os.Getenv("VELOX_BENCH_EXIT_AFTER_READY") == "1" {
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
	audit.complete = func() {
		if err := benchmarker.NotifyPolicyAudit(); err != nil {
			fmt.Fprintf(os.Stderr, "velox-host: policy audit marker: %v\n", err)
		}
		if os.Getenv("VELOX_BENCH_EXIT_AFTER_READY") == "1" {
			runtime.Close()
		}
	}
	defer runtime.Close()

	runtime.Run()
	return 0
}

type policyAudit struct {
	enabled  bool
	blocked  map[string]bool
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
		if !a.reported && len(a.missing()) == 0 && a.complete != nil {
			a.reported = true
			a.complete()
		}
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
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return missing
}
