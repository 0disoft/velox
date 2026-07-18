package hostmeta

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	SchemaVersion   = "actutum.host/v1"
	ContractVersion = 1
)

type Metadata struct {
	SchemaVersion  string    `json:"schemaVersion"`
	ReleaseVersion string    `json:"releaseVersion"`
	Target         string    `json:"target"`
	Contracts      Contracts `json:"contracts"`
	Host           Artifact  `json:"host"`
}

type Contracts struct {
	Host    int `json:"host"`
	Runtime int `json:"runtime"`
	IPC     int `json:"ipc"`
}

type Artifact struct {
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func Load(path string) (Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, fmt.Errorf("read host metadata: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var metadata Metadata
	if err := decoder.Decode(&metadata); err != nil {
		return Metadata{}, fmt.Errorf("decode host metadata: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Metadata{}, errors.New("decode host metadata: multiple JSON values")
		}
		return Metadata{}, fmt.Errorf("decode host metadata trailing data: %w", err)
	}
	if err := validate(metadata); err != nil {
		return Metadata{}, err
	}
	return metadata, nil
}

func (metadata Metadata) ValidateArtifact(hostPath, target, releaseVersion string, runtimeVersion, ipcVersion int, size int64, sha256 string) error {
	if metadata.Target != target {
		return fmt.Errorf("host target %q does not match requested target %q", metadata.Target, target)
	}
	if metadata.ReleaseVersion != releaseVersion {
		return fmt.Errorf("host release %q does not match CLI release %q", metadata.ReleaseVersion, releaseVersion)
	}
	if metadata.Contracts.Host != ContractVersion {
		return fmt.Errorf("unsupported host contract %d", metadata.Contracts.Host)
	}
	if metadata.Contracts.Runtime != runtimeVersion {
		return fmt.Errorf("host runtime contract %d does not match CLI runtime contract %d", metadata.Contracts.Runtime, runtimeVersion)
	}
	if metadata.Contracts.IPC != ipcVersion {
		return fmt.Errorf("host IPC contract %d does not match CLI IPC contract %d", metadata.Contracts.IPC, ipcVersion)
	}
	if metadata.Host.File != filepath.Base(hostPath) {
		return fmt.Errorf("host metadata names %q, found %q", metadata.Host.File, filepath.Base(hostPath))
	}
	if metadata.Host.Bytes != size {
		return fmt.Errorf("host size %d does not match metadata size %d", size, metadata.Host.Bytes)
	}
	if metadata.Host.SHA256 != sha256 {
		return errors.New("host SHA-256 does not match metadata")
	}
	return nil
}

func validate(metadata Metadata) error {
	if metadata.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported host metadata schema %q", metadata.SchemaVersion)
	}
	if strings.TrimSpace(metadata.ReleaseVersion) == "" {
		return errors.New("host releaseVersion is required")
	}
	if metadata.Target == "" {
		return errors.New("host target is required")
	}
	if filepath.Base(metadata.Host.File) != metadata.Host.File || metadata.Host.File == "." || metadata.Host.File == "" {
		return errors.New("host file must be a basename")
	}
	if metadata.Host.Bytes <= 0 {
		return errors.New("host bytes must be positive")
	}
	if len(metadata.Host.SHA256) != 64 || strings.ToLower(metadata.Host.SHA256) != metadata.Host.SHA256 {
		return errors.New("host sha256 must be 64 lowercase hexadecimal characters")
	}
	for _, character := range metadata.Host.SHA256 {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return errors.New("host sha256 must be 64 lowercase hexadecimal characters")
		}
	}
	return nil
}
