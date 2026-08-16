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
	"time"

	"github.com/0disoft/velox/internal/archive"
	"github.com/0disoft/velox/internal/assettree"
	"github.com/0disoft/velox/internal/buildphase"
	"github.com/0disoft/velox/internal/buildplan"
	"github.com/0disoft/velox/internal/buildreport"
	"github.com/0disoft/velox/internal/ipc"
	"github.com/0disoft/velox/internal/outputpair"
	"github.com/0disoft/velox/internal/runtimeconfig"
	"github.com/0disoft/velox/internal/safefs"
)

type Result struct {
	Report        buildreport.Report
	DirectoryPath string
	ArchivePath   string
	PortableBytes int64
	ArchiveSize   int64
	ArchiveSHA256 string
}

func Build(plan buildplan.Plan) (Result, error) {
	return BuildObserved(plan, nil)
}

func BuildObserved(plan buildplan.Plan, observer buildphase.Observer) (Result, error) {
	totalStarted := time.Now()
	defer buildphase.Record(observer, "build.total", totalStarted)
	snapshot := plan.Snapshot()
	if err := os.MkdirAll(snapshot.OutputRoot, 0o755); err != nil {
		return Result{}, fmt.Errorf("create output root: %w", err)
	}
	revalidateStarted := time.Now()
	if err := plan.RevalidateInputs(); err != nil {
		return Result{}, err
	}
	buildphase.Record(observer, "inputs.revalidate", revalidateStarted)
	stageDirectory := filepath.Join(snapshot.OutputRoot, "."+snapshot.ApplicationKey+".staging")
	stageArchive := filepath.Join(snapshot.OutputRoot, "."+snapshot.ApplicationKey+".zip.staging")
	if exists(stageDirectory) || exists(stageArchive) {
		return Result{}, errors.New("owned staging output already exists; remove it after confirming no build is active")
	}
	stageStarted := time.Now()
	if err := os.Mkdir(stageDirectory, 0o755); err != nil {
		return Result{}, fmt.Errorf("create staging directory: %w", err)
	}
	buildphase.Record(observer, "stage.create", stageStarted)
	success := false
	defer func() {
		if !success {
			os.RemoveAll(stageDirectory)
			os.Remove(stageArchive)
		}
	}()
	archiveStarted := time.Now()
	archiveStream, err := archive.NewStream(stageArchive)
	if err != nil {
		return Result{}, err
	}
	defer archiveStream.Abort()
	var archiveEntryDuration time.Duration
	archiveRoot := snapshot.ApplicationKey + "/"
	createArchiveEntry := func(name string) (io.Writer, error) {
		started := time.Now()
		entry, createErr := archiveStream.CreateEntry(name, 0o644)
		archiveEntryDuration += time.Since(started)
		return entry, createErr
	}

	hostName := snapshot.ApplicationKey + ".exe"
	hostArchiveEntry, err := createArchiveEntry(archiveRoot + hostName)
	if err != nil {
		return Result{}, err
	}
	hostStarted := time.Now()
	if _, err := copyVerified(snapshot.HostPath, filepath.Join(stageDirectory, hostName), 0o755, snapshot.HostSize, 0, snapshot.HostSHA256, observedWriter{writer: hostArchiveEntry, duration: &archiveEntryDuration}); err != nil {
		return Result{}, fmt.Errorf("copy host template: %w", err)
	}
	buildphase.Record(observer, "host.copy", hostStarted)
	webRoot := filepath.Join(stageDirectory, "web")
	copiedAssets := make([]assettree.File, 0, len(snapshot.Assets.Files))
	assetsStarted := time.Now()
	for _, asset := range snapshot.Assets.Files {
		destination := filepath.Join(webRoot, filepath.FromSlash(asset.RelativePath))
		assetArchiveEntry, err := createArchiveEntry(archiveRoot + "web/" + asset.RelativePath)
		if err != nil {
			return Result{}, err
		}
		digest, err := copyVerified(asset.SourcePath, destination, 0o644, asset.Size, asset.ModifiedUnixNano, asset.SHA256, observedWriter{writer: assetArchiveEntry, duration: &archiveEntryDuration})
		if err != nil {
			return Result{}, fmt.Errorf("copy asset %s: %w", asset.RelativePath, err)
		}
		asset.SHA256 = digest
		copiedAssets = append(copiedAssets, asset)
	}
	verifiedAssets := assettree.Summarize(copiedAssets)
	buildphase.Record(observer, "assets.copy", assetsStarted)

	runtimeValue := runtimeconfig.FromManifest(snapshot.Manifest, "web")
	runtimeArchiveEntry, err := createArchiveEntry(archiveRoot + "velox.runtime.json")
	if err != nil {
		return Result{}, err
	}
	runtimeStarted := time.Now()
	runtimeBytes, err := writeJSON(filepath.Join(stageDirectory, "velox.runtime.json"), runtimeValue, observedWriter{writer: runtimeArchiveEntry, duration: &archiveEntryDuration})
	if err != nil {
		return Result{}, err
	}
	buildphase.Record(observer, "runtime.write", runtimeStarted)
	report := buildreport.Report{
		SchemaVersion:  buildreport.SchemaVersion,
		ReleaseVersion: snapshot.HostMetadata.ReleaseVersion,
		App:            buildreport.App{ID: snapshot.Manifest.App.ID, Name: snapshot.Manifest.App.Name, Version: snapshot.Manifest.App.Version},
		Target:         snapshot.Target,
		Contracts:      buildreport.Contracts{Manifest: 1, Runtime: runtimeconfig.Version, Host: snapshot.HostMetadata.Contracts.Host, IPC: ipc.Version},
		Host:           buildreport.File{File: hostName, Bytes: snapshot.HostSize, SHA256: snapshot.HostSHA256},
		Assets:         buildreport.Assets{Files: len(verifiedAssets.Files), Bytes: verifiedAssets.TotalBytes, SHA256: verifiedAssets.Digest},
		Permissions:    append([]string{}, snapshot.Manifest.Security.Permissions...),
		Outputs:        buildreport.OutputCounts{PortableFiles: len(snapshot.Assets.Files) + 3},
	}
	reportArchiveEntry, err := createArchiveEntry(archiveRoot + "build-result.json")
	if err != nil {
		return Result{}, err
	}
	reportStarted := time.Now()
	reportBytes, err := writeJSON(filepath.Join(stageDirectory, "build-result.json"), report, observedWriter{writer: reportArchiveEntry, duration: &archiveEntryDuration})
	if err != nil {
		return Result{}, err
	}
	buildphase.Record(observer, "report.write", reportStarted)
	buildphase.Emit(observer, "archive.entries", archiveEntryDuration)
	archiveResult, err := archiveStream.CloseObserved(observer)
	if err != nil {
		return Result{}, err
	}
	buildphase.Record(observer, "archive.total", archiveStarted)
	if archiveResult.FileCount != report.Outputs.PortableFiles {
		return Result{}, fmt.Errorf("archive file count %d does not match build report %d", archiveResult.FileCount, report.Outputs.PortableFiles)
	}
	promoteStarted := time.Now()
	if err := promote(snapshot, stageDirectory, stageArchive); err != nil {
		return Result{}, err
	}
	buildphase.Record(observer, "output.promote", promoteStarted)
	success = true
	return Result{
		Report: report, DirectoryPath: snapshot.AppDirectory, ArchivePath: snapshot.ArchivePath,
		PortableBytes: snapshot.HostSize + verifiedAssets.TotalBytes + runtimeBytes + reportBytes,
		ArchiveSize:   archiveResult.Size, ArchiveSHA256: archiveResult.SHA256,
	}, nil
}

func promote(plan buildplan.Snapshot, stageDirectory, stageArchive string) error {
	return outputpair.Promote(plan.AppDirectory, plan.ArchivePath, stageDirectory, stageArchive)
}

func copyVerified(source, destination string, mode os.FileMode, expectedSize, expectedModifiedUnixNano int64, expectedSHA256 string, mirrors ...io.Writer) (string, error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", err
	}
	input, info, err := safefs.OpenVerifiedRegular(source)
	if err != nil {
		return "", err
	}
	defer input.Close()
	if info.Size() != expectedSize || (expectedModifiedUnixNano != 0 && info.ModTime().UnixNano() != expectedModifiedUnixNano) {
		return "", errors.New("source changed after build planning")
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	writers := []io.Writer{output, hash}
	writers = append(writers, mirrors...)
	written, err := io.Copy(io.MultiWriter(writers...), input)
	if err != nil {
		output.Close()
		return "", err
	}
	after, statErr := input.Stat()
	actualSHA256 := hex.EncodeToString(hash.Sum(nil))
	if statErr != nil || written != expectedSize || after.Size() != expectedSize ||
		(expectedModifiedUnixNano != 0 && after.ModTime().UnixNano() != expectedModifiedUnixNano) ||
		(expectedSHA256 != "" && actualSHA256 != expectedSHA256) {
		output.Close()
		os.Remove(destination)
		return "", errors.New("source changed after build planning")
	}
	if err := output.Close(); err != nil {
		return "", err
	}
	return actualSHA256, nil
}

func writeJSON(path string, value any, mirrors ...io.Writer) (int64, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return 0, fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	for _, mirror := range mirrors {
		written, err := mirror.Write(data)
		if err != nil {
			return 0, fmt.Errorf("archive %s: %w", filepath.Base(path), err)
		}
		if written != len(data) {
			return 0, fmt.Errorf("archive %s: short write", filepath.Base(path))
		}
	}
	return int64(len(data)), nil
}

type observedWriter struct {
	writer   io.Writer
	duration *time.Duration
}

func (writer observedWriter) Write(value []byte) (int, error) {
	started := time.Now()
	written, err := writer.writer.Write(value)
	*writer.duration += time.Since(started)
	return written, err
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}
