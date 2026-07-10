package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	webview2 "github.com/jchv/go-webview2"

	"github.com/0disoft/velox/internal/benchmarker"
	"github.com/0disoft/velox/internal/runtimeconfig"
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

	view := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     *debug,
		DataPath:  dataPath,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  cfg.App.Name,
			Width:  cfg.Window.Width,
			Height: cfg.Window.Height,
			Center: true,
		},
	})
	if view == nil {
		fmt.Fprintln(os.Stderr, "velox-host: WebView2 Runtime is unavailable or initialization failed")
		return 5
	}
	defer view.Destroy()

	if err := view.Bind("__veloxM0Ready", func(phase string) error {
		if err := benchmarker.NotifyReady(phase); err != nil {
			return err
		}
		if os.Getenv("VELOX_BENCH_EXIT_AFTER_READY") == "1" {
			view.Terminate()
		}
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "velox-host: bind ready marker: %v\n", err)
		return 6
	}

	view.Navigate(fileURL(cfg.EntryPath))
	view.Run()
	return 0
}

func fileURL(path string) string {
	slashed := filepath.ToSlash(path)
	if len(slashed) >= 2 && slashed[1] == ':' {
		slashed = "/" + slashed
	}
	return (&url.URL{Scheme: "file", Path: slashed}).String()
}
