package buildreport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const SchemaVersion = "actutum.build-result/v1"

type Report struct {
	SchemaVersion  string       `json:"schemaVersion"`
	ReleaseVersion string       `json:"releaseVersion"`
	App            App          `json:"app"`
	Target         string       `json:"target"`
	Contracts      Contracts    `json:"contracts"`
	Host           File         `json:"host"`
	Assets         Assets       `json:"assets"`
	Permissions    []string     `json:"permissions"`
	Outputs        OutputCounts `json:"outputs"`
}

type App struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Contracts struct {
	Manifest int `json:"manifest"`
	Runtime  int `json:"runtime"`
	Host     int `json:"host"`
	IPC      int `json:"ipc"`
}

type File struct {
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type Assets struct {
	Files  int    `json:"files"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type OutputCounts struct {
	PortableFiles int `json:"portableFiles"`
}

func Decode(reader io.Reader) (Report, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var report Report
	if err := decoder.Decode(&report); err != nil {
		return Report{}, fmt.Errorf("decode build result: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Report{}, errors.New("decode build result: multiple JSON values")
		}
		return Report{}, fmt.Errorf("decode build result trailing data: %w", err)
	}
	if err := report.Validate(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (report Report) Validate() error {
	if report.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported build result schema %q", report.SchemaVersion)
	}
	if strings.TrimSpace(report.ReleaseVersion) == "" {
		return errors.New("build result releaseVersion is required")
	}
	if report.App.ID == "" || report.App.Name == "" || report.App.Version == "" {
		return errors.New("build result application identity is incomplete")
	}
	if report.Target != "windows-x64" {
		return fmt.Errorf("unsupported build result target %q", report.Target)
	}
	if report.Contracts.Manifest != 1 || report.Contracts.Runtime != 1 || report.Contracts.Host != 1 || report.Contracts.IPC != 1 {
		return errors.New("unsupported build result contract versions")
	}
	if report.Host.File == "" || strings.ContainsAny(report.Host.File, `/\\:`) || report.Host.Bytes <= 0 || !validDigest(report.Host.SHA256) {
		return errors.New("invalid build result host artifact")
	}
	if report.Assets.Files < 1 || report.Assets.Bytes < 0 || !validDigest(report.Assets.SHA256) {
		return errors.New("invalid build result asset summary")
	}
	if report.Outputs.PortableFiles != report.Assets.Files+3 {
		return errors.New("build result portable file count is inconsistent")
	}
	seen := make(map[string]struct{}, len(report.Permissions))
	for _, permission := range report.Permissions {
		if permission != "app.info" && permission != "window.basic" {
			return fmt.Errorf("unsupported build result permission %q", permission)
		}
		if _, exists := seen[permission]; exists {
			return fmt.Errorf("duplicate build result permission %q", permission)
		}
		seen[permission] = struct{}{}
	}
	return nil
}

func validDigest(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}
