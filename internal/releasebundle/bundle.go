package releasebundle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/0disoft/velox/internal/archive"
	"github.com/0disoft/velox/internal/buildinfo"
	"github.com/0disoft/velox/internal/hostmeta"
	"github.com/0disoft/velox/internal/ipc"
	"github.com/0disoft/velox/internal/manifest"
	"github.com/0disoft/velox/internal/runtimeconfig"
)

const (
	SchemaVersion    = "velox.release/v1"
	TargetWindowsX64 = "windows-x64"
)

var releaseSchemaFiles = []string{
	"build-result-v1.schema.json",
	"consumer-clean-v1.schema.json",
	"host-metadata-v1.schema.json",
	"ipc-v1.schema.json",
	"public-preview-verification-v1.schema.json",
	"release-manifest-v1.schema.json",
	"runtime-config-v1.schema.json",
	"velox-v1.schema.json",
}

type Options struct {
	CLIPath    string
	HostPath   string
	SourceRoot string
	OutputRoot string
}

type Manifest struct {
	SchemaVersion  string     `json:"schemaVersion"`
	ReleaseVersion string     `json:"releaseVersion"`
	Target         string     `json:"target"`
	Contracts      Contracts  `json:"contracts"`
	Artifacts      []Artifact `json:"artifacts"`
}

type Contracts struct {
	Manifest    int `json:"manifest"`
	Runtime     int `json:"runtime"`
	Host        int `json:"host"`
	IPC         int `json:"ipc"`
	BuildResult int `json:"buildResult"`
}

type Artifact struct {
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type Result struct {
	Directory     string `json:"directory"`
	Archive       string `json:"archive"`
	ArchiveBytes  int64  `json:"archiveBytes"`
	ArchiveSHA256 string `json:"archiveSha256"`
}

func Build(options Options) (Result, error) {
	if options.SourceRoot == "" || options.OutputRoot == "" {
		return Result{}, errors.New("source and output roots are required")
	}
	outputRoot, err := filepath.Abs(options.OutputRoot)
	if err != nil {
		return Result{}, fmt.Errorf("resolve release output: %w", err)
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return Result{}, fmt.Errorf("create release output: %w", err)
	}
	name := "velox-windows-x64"
	stageDirectory := filepath.Join(outputRoot, "."+name+".staging")
	stageArchive := filepath.Join(outputRoot, "."+name+".zip.staging")
	finalDirectory := filepath.Join(outputRoot, name)
	finalArchive := filepath.Join(outputRoot, name+".zip")
	if exists(stageDirectory) || exists(stageArchive) {
		return Result{}, errors.New("release staging output already exists")
	}
	if err := os.Mkdir(stageDirectory, 0o755); err != nil {
		return Result{}, fmt.Errorf("create release staging: %w", err)
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(stageDirectory)
			_ = os.Remove(stageArchive)
		}
	}()

	cliArtifact, err := copyArtifact(options.CLIPath, filepath.Join(stageDirectory, "velox.exe"), "velox.exe")
	if err != nil {
		return Result{}, fmt.Errorf("package CLI: %w", err)
	}
	hostArtifact, err := copyArtifact(options.HostPath, filepath.Join(stageDirectory, "velox-host.exe"), "velox-host.exe")
	if err != nil {
		return Result{}, fmt.Errorf("package host: %w", err)
	}
	hostMetadata := hostmeta.Metadata{
		SchemaVersion: hostmeta.SchemaVersion, ReleaseVersion: buildinfo.Version, Target: TargetWindowsX64,
		Contracts: hostmeta.Contracts{Host: hostmeta.ContractVersion, Runtime: runtimeconfig.Version, IPC: ipc.Version},
		Host:      hostmeta.Artifact{File: hostArtifact.File, Bytes: hostArtifact.Bytes, SHA256: hostArtifact.SHA256},
	}
	if err := writeJSON(filepath.Join(stageDirectory, "velox-host.json"), hostMetadata); err != nil {
		return Result{}, err
	}

	artifacts := []Artifact{cliArtifact, hostArtifact}
	schemaRoot := filepath.Join(options.SourceRoot, "schema")
	for _, schemaFile := range releaseSchemaFiles {
		relative := filepath.ToSlash(filepath.Join("schema", schemaFile))
		artifact, err := copyArtifact(filepath.Join(schemaRoot, schemaFile), filepath.Join(stageDirectory, filepath.FromSlash(relative)), relative)
		if err != nil {
			return Result{}, fmt.Errorf("package schema %s: %w", schemaFile, err)
		}
		artifacts = append(artifacts, artifact)
	}
	notices, err := copyArtifact(filepath.Join(options.SourceRoot, "THIRD_PARTY_NOTICES.md"), filepath.Join(stageDirectory, "THIRD_PARTY_NOTICES.md"), "THIRD_PARTY_NOTICES.md")
	if err != nil {
		return Result{}, fmt.Errorf("package third-party notices: %w", err)
	}
	artifacts = append(artifacts, notices)
	hostMetadataArtifact, err := inspectArtifact(filepath.Join(stageDirectory, "velox-host.json"), "velox-host.json")
	if err != nil {
		return Result{}, err
	}
	artifacts = append(artifacts, hostMetadataArtifact)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].File < artifacts[j].File })
	releaseManifest := Manifest{
		SchemaVersion: SchemaVersion, ReleaseVersion: buildinfo.Version, Target: TargetWindowsX64,
		Contracts: Contracts{Manifest: manifest.Version, Runtime: runtimeconfig.Version, Host: hostmeta.ContractVersion, IPC: ipc.Version, BuildResult: 1},
		Artifacts: artifacts,
	}
	if err := writeJSON(filepath.Join(stageDirectory, "release-manifest.json"), releaseManifest); err != nil {
		return Result{}, err
	}

	archiveResult, err := archive.Create(stageDirectory, stageArchive, name)
	if err != nil {
		return Result{}, err
	}
	if err := promote(finalDirectory, finalArchive, stageDirectory, stageArchive); err != nil {
		return Result{}, err
	}
	success = true
	return Result{Directory: finalDirectory, Archive: finalArchive, ArchiveBytes: archiveResult.Size, ArchiveSHA256: archiveResult.SHA256}, nil
}

func copyArtifact(source, destination, relative string) (Artifact, error) {
	info, err := os.Lstat(source)
	if err != nil {
		return Artifact{}, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return Artifact{}, errors.New("release input must be a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return Artifact{}, err
	}
	input, err := os.Open(source)
	if err != nil {
		return Artifact{}, err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(output, hash), input)
	closeErr := output.Close()
	if copyErr != nil {
		return Artifact{}, copyErr
	}
	if closeErr != nil {
		return Artifact{}, closeErr
	}
	if written != info.Size() {
		return Artifact{}, errors.New("release input changed while copying")
	}
	return Artifact{File: filepath.ToSlash(relative), Bytes: written, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func inspectArtifact(path, relative string) (Artifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Artifact{}, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return Artifact{}, err
	}
	return Artifact{File: filepath.ToSlash(relative), Bytes: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func promote(finalDirectory, finalArchive, stageDirectory, stageArchive string) error {
	backupDirectory := finalDirectory + ".previous"
	backupArchive := finalArchive + ".previous"
	if exists(backupDirectory) || exists(backupArchive) {
		return errors.New("release recovery output already exists")
	}
	directoryBackedUp := false
	archiveBackedUp := false
	if exists(finalDirectory) {
		if err := os.Rename(finalDirectory, backupDirectory); err != nil {
			return fmt.Errorf("backup release directory: %w", err)
		}
		directoryBackedUp = true
	}
	if exists(finalArchive) {
		if err := os.Rename(finalArchive, backupArchive); err != nil {
			if directoryBackedUp {
				_ = os.Rename(backupDirectory, finalDirectory)
			}
			return fmt.Errorf("backup release archive: %w", err)
		}
		archiveBackedUp = true
	}
	if err := os.Rename(stageDirectory, finalDirectory); err != nil {
		if directoryBackedUp {
			_ = os.Rename(backupDirectory, finalDirectory)
		}
		if archiveBackedUp {
			_ = os.Rename(backupArchive, finalArchive)
		}
		return fmt.Errorf("promote release directory: %w", err)
	}
	if err := os.Rename(stageArchive, finalArchive); err != nil {
		_ = os.RemoveAll(finalDirectory)
		if directoryBackedUp {
			_ = os.Rename(backupDirectory, finalDirectory)
		}
		if archiveBackedUp {
			_ = os.Rename(backupArchive, finalArchive)
		}
		return fmt.Errorf("promote release archive: %w", err)
	}
	if directoryBackedUp {
		_ = os.RemoveAll(backupDirectory)
	}
	if archiveBackedUp {
		_ = os.Remove(backupArchive)
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}
