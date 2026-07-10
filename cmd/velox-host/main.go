package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
		dataPath = filepath.Join(os.TempDir(), "velox-m0", cfg.App.ID)
	}

	var runtime *webview2.M0Runtime
	runtime, err = webview2.OpenM0(webview2.Config{
		Title:     cfg.App.Name,
		Width:     cfg.Window.Width,
		Height:    cfg.Window.Height,
		DataPath:  dataPath,
		EntryPath: cfg.EntryPath,
		Debug:     *debug,
	}, func(phase string) error {
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
	defer runtime.Close()

	runtime.Run()
	return 0
}
