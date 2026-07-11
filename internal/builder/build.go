package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/0disoft/velox/internal/archive"
	"github.com/0disoft/velox/internal/buildplan"
	"github.com/0disoft/velox/internal/runtimeconfig"
)

const ReportSchema = "velox.build-result/v1"

type Report struct {
	SchemaVersion  string             `json:"schemaVersion"`
	ReleaseVersion string             `json:"releaseVersion"`
	App            ReportApp          `json:"app"`
	Target         string             `json:"target"`
	Contracts      ReportContracts    `json:"contracts"`
	Host           ReportFile         `json:"host"`
	Assets         ReportAssets       `json:"assets"`
	Permissions    []string           `json:"permissions"`
	Outputs        ReportOutputCounts `json:"outputs"`
}

type ReportApp struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ReportContracts struct {
	Manifest int `json:"manifest"`
	Runtime  int `json:"runtime"`
	Host     int `json:"host"`
}

type ReportFile struct {
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type ReportAssets struct {
	Files  int    `json:"files"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type ReportOutputCounts struct {
	PortableFiles int `json:"portableFiles"`
}

type Result struct {
	Report        Report
	DirectoryPath string
	ArchivePath   string
	ArchiveSize   int64
	ArchiveSHA256 string
}

func Build(plan buildplan.Plan) (Result, error) {
	snapshot := plan.Snapshot()
	if err := os.MkdirAll(snapshot.OutputRoot, 0o755); err != nil {
		return Result{}, fmt.Errorf("create output root: %w", err)
	}
	stageDirectory := filepath.Join(snapshot.OutputRoot, "."+snapshot.ApplicationKey+".staging")
	stageArchive := filepath.Join(snapshot.OutputRoot, "."+snapshot.ApplicationKey+".zip.staging")
	if exists(stageDirectory) || exists(stageArchive) {
		return Result{}, errors.New("owned staging output already exists; remove it after confirming no build is active")
	}
	if err := os.Mkdir(stageDirectory, 0o755); err != nil {
		return Result{}, fmt.Errorf("create staging directory: %w", err)
	}
	success := false
	defer func() {
		if !success {
			os.RemoveAll(stageDirectory)
			os.Remove(stageArchive)
		}
	}()

	hostName := snapshot.ApplicationKey + ".exe"
	if err := copyVerified(snapshot.HostPath, filepath.Join(stageDirectory, hostName), 0o755, snapshot.HostSize, snapshot.HostSHA256); err != nil {
		return Result{}, fmt.Errorf("copy host template: %w", err)
	}
	webRoot := filepath.Join(stageDirectory, "web")
	for _, asset := range snapshot.Assets.Files {
		destination := filepath.Join(webRoot, filepath.FromSlash(asset.RelativePath))
		if err := copyVerified(asset.SourcePath, destination, 0o644, asset.Size, asset.SHA256); err != nil {
			return Result{}, fmt.Errorf("copy asset %s: %w", asset.RelativePath, err)
		}
	}

	runtimeValue := runtimeconfig.Config{
		RuntimeVersion: runtimeconfig.Version,
		App: runtimeconfig.App{
			ID: snapshot.Manifest.App.ID, Name: snapshot.Manifest.App.Name, Version: snapshot.Manifest.App.Version,
		},
		Assets:   runtimeconfig.Assets{Root: "web", Entry: filepath.ToSlash(snapshot.Manifest.Assets.Entry)},
		Window:   runtimeconfig.Window{Width: snapshot.Manifest.Window.Width, Height: snapshot.Manifest.Window.Height},
		Security: runtimeconfig.Security{Permissions: append([]string(nil), snapshot.Manifest.Security.Permissions...)},
	}
	if err := writeJSON(filepath.Join(stageDirectory, "velox.runtime.json"), runtimeValue); err != nil {
		return Result{}, err
	}
	report := Report{
		SchemaVersion:  ReportSchema,
		ReleaseVersion: snapshot.HostMetadata.ReleaseVersion,
		App:            ReportApp{ID: snapshot.Manifest.App.ID, Name: snapshot.Manifest.App.Name, Version: snapshot.Manifest.App.Version},
		Target:         snapshot.Target,
		Contracts:      ReportContracts{Manifest: 1, Runtime: runtimeconfig.Version, Host: snapshot.HostMetadata.Contracts.Host},
		Host:           ReportFile{File: hostName, Bytes: snapshot.HostSize, SHA256: snapshot.HostSHA256},
		Assets:         ReportAssets{Files: len(snapshot.Assets.Files), Bytes: snapshot.Assets.TotalBytes, SHA256: snapshot.Assets.Digest},
		Permissions:    append([]string(nil), snapshot.Manifest.Security.Permissions...),
		Outputs:        ReportOutputCounts{PortableFiles: len(snapshot.Assets.Files) + 3},
	}
	if err := writeJSON(filepath.Join(stageDirectory, "build-result.json"), report); err != nil {
		return Result{}, err
	}
	archiveResult, err := archive.Create(stageDirectory, stageArchive, snapshot.ApplicationKey)
	if err != nil {
		return Result{}, err
	}
	if archiveResult.FileCount != report.Outputs.PortableFiles {
		return Result{}, fmt.Errorf("archive file count %d does not match build report %d", archiveResult.FileCount, report.Outputs.PortableFiles)
	}
	if err := promote(snapshot, stageDirectory, stageArchive); err != nil {
		return Result{}, err
	}
	success = true
	return Result{
		Report: report, DirectoryPath: snapshot.AppDirectory, ArchivePath: snapshot.ArchivePath,
		ArchiveSize: archiveResult.Size, ArchiveSHA256: archiveResult.SHA256,
	}, nil
}

func promote(plan buildplan.Snapshot, stageDirectory, stageArchive string) error {
	backupDirectory := plan.AppDirectory + ".previous"
	backupArchive := plan.ArchivePath + ".previous"
	if exists(backupDirectory) || exists(backupArchive) {
		return errors.New("previous-output backup already exists; refusing to overwrite recovery data")
	}
	directoryBackedUp := false
	archiveBackedUp := false
	if exists(plan.AppDirectory) {
		if err := os.Rename(plan.AppDirectory, backupDirectory); err != nil {
			return fmt.Errorf("backup previous app directory: %w", err)
		}
		directoryBackedUp = true
	}
	if exists(plan.ArchivePath) {
		if err := os.Rename(plan.ArchivePath, backupArchive); err != nil {
			if directoryBackedUp {
				_ = os.Rename(backupDirectory, plan.AppDirectory)
			}
			return fmt.Errorf("backup previous archive: %w", err)
		}
		archiveBackedUp = true
	}
	rollback := func() {
		_ = os.RemoveAll(plan.AppDirectory)
		_ = os.Remove(plan.ArchivePath)
		if directoryBackedUp {
			_ = os.Rename(backupDirectory, plan.AppDirectory)
		}
		if archiveBackedUp {
			_ = os.Rename(backupArchive, plan.ArchivePath)
		}
	}
	if err := os.Rename(stageDirectory, plan.AppDirectory); err != nil {
		rollback()
		return fmt.Errorf("promote app directory: %w", err)
	}
	if err := os.Rename(stageArchive, plan.ArchivePath); err != nil {
		rollback()
		return fmt.Errorf("promote archive: %w", err)
	}
	if directoryBackedUp {
		if err := os.RemoveAll(backupDirectory); err != nil {
			return fmt.Errorf("remove previous app backup: %w", err)
		}
	}
	if archiveBackedUp {
		if err := os.Remove(backupArchive); err != nil {
			return fmt.Errorf("remove previous archive backup: %w", err)
		}
	}
	return nil
}

func copyVerified(source, destination string, mode os.FileMode, expectedSize int64, expectedSHA256 string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(output, hash), input)
	if err != nil {
		output.Close()
		return err
	}
	actualSHA256 := hex.EncodeToString(hash.Sum(nil))
	if written != expectedSize || actualSHA256 != expectedSHA256 {
		output.Close()
		os.Remove(destination)
		return errors.New("source changed after build planning")
	}
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}
