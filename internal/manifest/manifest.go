package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const Version = 1

var appIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)+$`)

type Manifest struct {
	Schema        string   `json:"$schema,omitempty"`
	SchemaVersion int      `json:"schemaVersion"`
	App           App      `json:"app"`
	Assets        Assets   `json:"assets"`
	Window        Window   `json:"window"`
	Security      Security `json:"security"`
}

type App struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
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
	Manifest
	ConfigPath  string
	ProjectRoot string
	AssetRoot   string
	EntryPath   string
}

func Load(path string) (Resolved, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve manifest path: %w", err)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return Resolved{}, fmt.Errorf("read manifest: %w", err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var value Manifest
	if err := decoder.Decode(&value); err != nil {
		return Resolved{}, fmt.Errorf("decode manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Resolved{}, errors.New("decode manifest: multiple JSON values")
		}
		return Resolved{}, fmt.Errorf("decode manifest trailing data: %w", err)
	}
	applyDefaults(&value)
	if err := validate(value); err != nil {
		return Resolved{}, err
	}

	projectRoot := filepath.Dir(absPath)
	assetRoot, err := resolveContained(projectRoot, value.Assets.Root)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve assets.root: %w", err)
	}
	entryPath, err := resolveContained(assetRoot, value.Assets.Entry)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve assets.entry: %w", err)
	}
	return Resolved{
		Manifest:    value,
		ConfigPath:  absPath,
		ProjectRoot: projectRoot,
		AssetRoot:   assetRoot,
		EntryPath:   entryPath,
	}, nil
}

func applyDefaults(value *Manifest) {
	if value.Assets.Root == "" {
		value.Assets.Root = "web"
	}
	if value.Assets.Entry == "" {
		value.Assets.Entry = "index.html"
	}
	if value.Window.Width == 0 {
		value.Window.Width = 960
	}
	if value.Window.Height == 0 {
		value.Window.Height = 640
	}
	if value.Security.Permissions == nil {
		value.Security.Permissions = []string{}
	}
}

func validate(value Manifest) error {
	if value.SchemaVersion != Version {
		return fmt.Errorf("unsupported schemaVersion %d", value.SchemaVersion)
	}
	if !appIDPattern.MatchString(value.App.ID) {
		return errors.New("app.id must be lowercase reverse-domain ASCII")
	}
	if strings.TrimSpace(value.App.Name) == "" {
		return errors.New("app.name is required")
	}
	if strings.TrimSpace(value.App.Version) == "" {
		return errors.New("app.version is required")
	}
	if value.Window.Width < 320 || value.Window.Height < 240 {
		return errors.New("window dimensions must be at least 320x240")
	}
	seen := make(map[string]struct{}, len(value.Security.Permissions))
	for _, permission := range value.Security.Permissions {
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

func resolveContained(root, relative string) (string, error) {
	if relative == "" {
		return "", errors.New("path is required")
	}
	if filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" {
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
