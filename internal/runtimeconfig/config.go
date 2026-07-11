package runtimeconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const Version = 1

type Config struct {
	RuntimeVersion int      `json:"runtimeVersion"`
	App            App      `json:"app"`
	Assets         Assets   `json:"assets"`
	Window         Window   `json:"window"`
	Security       Security `json:"security"`
}

type App struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type Assets struct {
	Root  string `json:"root"`
	Entry string `json:"entry"`
}

type Window struct {
	Width  uint `json:"width"`
	Height uint `json:"height"`
}

type Security struct {
	Permissions []string `json:"permissions"`
}

type Resolved struct {
	Config
	ConfigPath string
	AssetRoot  string
	EntryPath  string
}

func Load(path string) (Resolved, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve config path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return Resolved{}, fmt.Errorf("read runtime config: %w", err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Resolved{}, fmt.Errorf("decode runtime config: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Resolved{}, errors.New("decode runtime config: multiple JSON values")
		}
		return Resolved{}, fmt.Errorf("decode runtime config trailing data: %w", err)
	}
	if err := validate(cfg); err != nil {
		return Resolved{}, err
	}

	configDir := filepath.Dir(absPath)
	assetRoot, err := containedPath(configDir, cfg.Assets.Root)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve asset root: %w", err)
	}
	entryPath, err := containedPath(assetRoot, cfg.Assets.Entry)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve entry point: %w", err)
	}

	rootInfo, err := os.Stat(assetRoot)
	if err != nil {
		return Resolved{}, fmt.Errorf("inspect asset root: %w", err)
	}
	if !rootInfo.IsDir() {
		return Resolved{}, errors.New("asset root is not a directory")
	}
	entryInfo, err := os.Stat(entryPath)
	if err != nil {
		return Resolved{}, fmt.Errorf("inspect entry point: %w", err)
	}
	if !entryInfo.Mode().IsRegular() {
		return Resolved{}, errors.New("entry point is not a regular file")
	}

	return Resolved{
		Config:     cfg,
		ConfigPath: absPath,
		AssetRoot:  assetRoot,
		EntryPath:  entryPath,
	}, nil
}

func validate(cfg Config) error {
	if cfg.RuntimeVersion != Version {
		return fmt.Errorf("unsupported runtimeVersion %d", cfg.RuntimeVersion)
	}
	if strings.TrimSpace(cfg.App.ID) == "" {
		return errors.New("app.id is required")
	}
	if strings.TrimSpace(cfg.App.Name) == "" {
		return errors.New("app.name is required")
	}
	if strings.TrimSpace(cfg.App.Version) == "" {
		return errors.New("app.version is required")
	}
	if cfg.Assets.Root == "" {
		return errors.New("assets.root is required")
	}
	if cfg.Assets.Entry == "" {
		return errors.New("assets.entry is required")
	}
	if cfg.Window.Width < 320 || cfg.Window.Height < 240 {
		return errors.New("window dimensions must be at least 320x240")
	}
	if cfg.Security.Permissions == nil {
		return errors.New("security.permissions is required")
	}
	seen := make(map[string]struct{}, len(cfg.Security.Permissions))
	for _, permission := range cfg.Security.Permissions {
		if permission != "app.info" && permission != "window.basic" {
			return fmt.Errorf("unsupported permission %q", permission)
		}
		if _, exists := seen[permission]; exists {
			return fmt.Errorf("duplicate permission %q", permission)
		}
		seen[permission] = struct{}{}
	}
	return nil
}

func containedPath(root, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", errors.New("absolute paths are not allowed")
	}
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path must stay inside its owner root")
	}

	joined := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, joined)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes its owner root")
	}
	return joined, nil
}
