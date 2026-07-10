package webview2

import (
	"errors"
	"fmt"
	"path/filepath"
)

var ErrRuntimeUnavailable = errors.New("WebView2 Runtime is unavailable or initialization failed")

type Config struct {
	Title     string
	Width     uint
	Height    uint
	DataPath  string
	EntryPath string
	Debug     bool
}

type Capabilities struct {
	VirtualHTTPSOrigin    bool
	TrustedOriginMessages bool
	NavigationPolicy      bool
	NewWindowPolicy       bool
	DownloadPolicy        bool
	PermissionPolicy      bool
	CleanShutdown         bool
}

func M0Capabilities() Capabilities {
	return Capabilities{}
}

func (c Config) validate() error {
	if c.Title == "" {
		return errors.New("window title is required")
	}
	if c.Width == 0 || c.Height == 0 {
		return errors.New("window dimensions must be positive")
	}
	if !filepath.IsAbs(c.DataPath) {
		return errors.New("WebView2 data path must be absolute")
	}
	if !filepath.IsAbs(c.EntryPath) {
		return errors.New("entry path must be absolute")
	}
	entryInfo, err := filepath.EvalSymlinks(c.EntryPath)
	if err != nil {
		return fmt.Errorf("resolve entry path: %w", err)
	}
	if entryInfo == "" {
		return errors.New("resolved entry path is empty")
	}
	return nil
}
