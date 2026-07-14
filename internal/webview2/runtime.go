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

const (
	PolicyNavigation      = "navigation"
	PolicyFrameNavigation = "frame-navigation"
	PolicyNewWindow       = "new-window"
	PolicyDownload        = "download"
	PolicyPermission      = "permission"
	PolicyMessageSource   = "message-source"
)

type Config struct {
	Title                   string
	AppID                   string
	AppVersion              string
	Permissions             []string
	Width                   uint
	Height                  uint
	DataPath                string
	BrowserExecutableFolder string
	AssetRoot               string
	EntryPath               string
	Debug                   bool
	PolicyBlocked           func(kind string)
	StartupPhase            func(name string)
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

func RuntimeCapabilities() Capabilities {
	return Capabilities{
		VirtualHTTPSOrigin:    true,
		TrustedOriginMessages: true,
		NavigationPolicy:      true,
		NewWindowPolicy:       true,
		DownloadPolicy:        true,
		PermissionPolicy:      true,
		CleanShutdown:         true,
	}
}

func (c Config) validate() error {
	if c.Title == "" {
		return errors.New("window title is required")
	}
	if strings.TrimSpace(c.AppID) == "" {
		return errors.New("app ID is required")
	}
	if strings.TrimSpace(c.AppVersion) == "" {
		return errors.New("app version is required")
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

func trustedOrigin(appID string) string {
	return "https://" + trustedHost(appID)
}

func isTrustedDocument(rawURL, appID string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil {
		return false
	}
	return parsed.Host == trustedHost(appID)
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
