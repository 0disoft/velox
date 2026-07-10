package webview2

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

var ErrRuntimeUnavailable = errors.New("WebView2 Runtime is unavailable or initialization failed")

type Config struct {
	Title     string
	AppID     string
	Width     uint
	Height    uint
	DataPath  string
	AssetRoot string
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
	return Capabilities{
		VirtualHTTPSOrigin: true,
		PermissionPolicy:   true,
	}
}

func (c Config) validate() error {
	if c.Title == "" {
		return errors.New("window title is required")
	}
	if strings.TrimSpace(c.AppID) == "" {
		return errors.New("app ID is required")
	}
	if c.Width == 0 || c.Height == 0 {
		return errors.New("window dimensions must be positive")
	}
	if !filepath.IsAbs(c.DataPath) {
		return errors.New("WebView2 data path must be absolute")
	}
	if !filepath.IsAbs(c.AssetRoot) {
		return errors.New("asset root must be absolute")
	}
	if !filepath.IsAbs(c.EntryPath) {
		return errors.New("entry path must be absolute")
	}
	resolvedRoot, err := filepath.EvalSymlinks(c.AssetRoot)
	if err != nil {
		return fmt.Errorf("resolve asset root: %w", err)
	}
	resolvedEntry, err := filepath.EvalSymlinks(c.EntryPath)
	if err != nil {
		return fmt.Errorf("resolve entry path: %w", err)
	}
	relativeEntry, err := filepath.Rel(resolvedRoot, resolvedEntry)
	if err != nil {
		return fmt.Errorf("locate entry under asset root: %w", err)
	}
	if relativeEntry == ".." || strings.HasPrefix(relativeEntry, ".."+string(filepath.Separator)) {
		return errors.New("entry path must stay inside asset root")
	}
	return nil
}

func trustedHost(appID string) string {
	digest := sha256.Sum256([]byte(appID))
	return fmt.Sprintf("%x.app.invalid", digest[:16])
}

func virtualEntryURL(appID, assetRoot, entryPath string) (string, error) {
	relativeEntry, err := filepath.Rel(assetRoot, entryPath)
	if err != nil {
		return "", fmt.Errorf("locate virtual entry: %w", err)
	}
	if relativeEntry == ".." || strings.HasPrefix(relativeEntry, ".."+string(filepath.Separator)) {
		return "", errors.New("virtual entry must stay inside asset root")
	}
	return (&url.URL{
		Scheme: "https",
		Host:   trustedHost(appID),
		Path:   "/" + filepath.ToSlash(relativeEntry),
	}).String(), nil
}
