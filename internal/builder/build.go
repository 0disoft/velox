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
	"github.com/0disoft/velox/internal/buildreport"
	"github.com/0disoft/velox/internal/ipc"
	"github.com/0disoft/velox/internal/runtimeconfig"
)

type Result struct {
	Report        buildreport.Report
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
	if err := plan.RevalidateInputs(); err != nil {
		return Result{}, err
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

	runtimeValue := runtimeconfig.FromManifest(snapshot.Manifest, "web")
	if err := writeJSON(filepath.Join(stageDirectory, "velox.runtime.json"), runtimeValue); err != nil {
		return Result{}, err
	}
	report := buildreport.Report{
		SchemaVersion:  buildreport.SchemaVersion,
		ReleaseVersion: snapshot.HostMetadata.ReleaseVersion,
		App:            buildreport.App{ID: snapshot.Manifest.App.ID, Name: snapshot.Manifest.App.Name, Version: snapshot.Manifest.App.Version},
		Target:         snapshot.Target,
		Contracts:      buildreport.Contracts{Manifest: 1, Runtime: runtimeconfig.Version, Host: snapshot.HostMetadata.Contracts.Host, IPC: ipc.Version},
		Host:           buildreport.File{File: hostName, Bytes: snapshot.HostSize, SHA256: snapshot.HostSHA256},
		Assets:         buildreport.Assets{Files: len(snapshot.Assets.Files), Bytes: snapshot.Assets.TotalBytes, SHA256: snapshot.Assets.Digest},
		Permissions:    append([]string{}, snapshot.Manifest.Security.Permissions...),
		Outputs:        buildreport.OutputCounts{PortableFiles: len(snapshot.Assets.Files) + 3},
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
	return promoteWithOperations(plan, stageDirectory, stageArchive, promotionOperations{
		rename:    os.Rename,
		remove:    os.Remove,
		removeAll: os.RemoveAll,
	})
}

type promotionOperations struct {
	rename    func(string, string) error
	remove    func(string) error
	removeAll func(string) error
}

func promoteWithOperations(plan buildplan.Snapshot, stageDirectory, stageArchive string, operations promotionOperations) error {
	backupDirectory := plan.AppDirectory + ".previous"
	backupArchive := plan.ArchivePath + ".previous"
	if exists(backupDirectory) || exists(backupArchive) {
		return errors.New("previous-output backup already exists; refusing to overwrite recovery data")
	}
	if err := validateExistingOutput(plan.AppDirectory, true); err != nil {
		return fmt.Errorf("validate previous app directory: %w", err)
	}
	if err := validateExistingOutput(plan.ArchivePath, false); err != nil {
		return fmt.Errorf("validate previous archive: %w", err)
	}
	directoryBackedUp := false
	archiveBackedUp := false
	if exists(plan.AppDirectory) {
		if err := operations.rename(plan.AppDirectory, backupDirectory); err != nil {
			return fmt.Errorf("backup previous app directory: %w", err)
		}
		directoryBackedUp = true
	}
	if exists(plan.ArchivePath) {
		if err := operations.rename(plan.ArchivePath, backupArchive); err != nil {
			backupErr := fmt.Errorf("backup previous archive: %w", err)
			if directoryBackedUp {
				if restoreErr := operations.rename(backupDirectory, plan.AppDirectory); restoreErr != nil {
					return errors.Join(backupErr, fmt.Errorf("restore previous app directory after archive backup failure: %w", restoreErr))
				}
			}
			return backupErr
		}
		archiveBackedUp = true
	}
	rollback := func() error {
		var rollbackErr error
		if err := operations.removeAll(plan.AppDirectory); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove partially promoted app directory: %w", err))
		}
		if err := operations.remove(plan.ArchivePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("remove partially promoted archive: %w", err))
		}
		if directoryBackedUp {
			if err := operations.rename(backupDirectory, plan.AppDirectory); err != nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore previous app directory: %w", err))
			}
		}
		if archiveBackedUp {
			if err := operations.rename(backupArchive, plan.ArchivePath); err != nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("restore previous archive: %w", err))
			}
		}
		return rollbackErr
	}
	if err := operations.rename(stageDirectory, plan.AppDirectory); err != nil {
		return errors.Join(fmt.Errorf("promote app directory: %w", err), rollback())
	}
	if err := operations.rename(stageArchive, plan.ArchivePath); err != nil {
		return errors.Join(fmt.Errorf("promote archive: %w", err), rollback())
	}
	var cleanupErr error
	if directoryBackedUp {
		if err := operations.removeAll(backupDirectory); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove previous app directory after promotion: %w", err))
		}
	}
	if archiveBackedUp {
		if err := operations.remove(backupArchive); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove previous archive after promotion: %w", err))
		}
	}
	return cleanupErr
}

func validateExistingOutput(path string, wantDirectory bool) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("output must not be a symbolic link")
	}
	if wantDirectory && !info.IsDir() {
		return errors.New("output is not a directory")
	}
	if !wantDirectory && !info.Mode().IsRegular() {
		return errors.New("output is not a regular file")
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
