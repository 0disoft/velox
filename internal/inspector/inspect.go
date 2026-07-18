package inspector

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/actutum/internal/assettree"
	"github.com/0disoft/actutum/internal/buildreport"
	"github.com/0disoft/actutum/internal/runtimeconfig"
)

const (
	maxArchiveFiles       = 100_000
	maxMetadataBytes      = 1 << 20
	maxArchiveEntryBytes  = 512 << 20
	maxArchiveTotalBytes  = 1 << 30
	maxArchiveExpandRatio = 1_000
)

type Result struct {
	Kind           string                `json:"kind"`
	ReleaseVersion string                `json:"releaseVersion"`
	App            buildreport.App       `json:"app"`
	Target         string                `json:"target"`
	Contracts      buildreport.Contracts `json:"contracts"`
	Permissions    []string              `json:"permissions"`
	PortableFiles  int                   `json:"portableFiles"`
	PortableBytes  int64                 `json:"portableBytes"`
	HostSHA256     string                `json:"hostSha256"`
	AssetSHA256    string                `json:"assetSha256"`
}

func Inspect(inputPath string) (Result, error) {
	absolute, err := filepath.Abs(inputPath)
	if err != nil {
		return Result{}, fmt.Errorf("resolve artifact path: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return Result{}, fmt.Errorf("inspect artifact path: %w", err)
	}
	if info.IsDir() {
		return inspectDirectory(absolute)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return Result{}, errors.New("artifact must be a regular directory or ZIP file")
	}
	if !strings.EqualFold(filepath.Ext(absolute), ".zip") {
		return Result{}, errors.New("artifact file must be a ZIP archive")
	}
	return inspectZIP(absolute)
}

func inspectDirectory(root string) (Result, error) {
	reportFile, err := os.Open(filepath.Join(root, "build-result.json"))
	if err != nil {
		return Result{}, fmt.Errorf("open build result: %w", err)
	}
	report, decodeErr := buildreport.Decode(io.LimitReader(reportFile, maxMetadataBytes+1))
	closeErr := reportFile.Close()
	if decodeErr != nil {
		return Result{}, decodeErr
	}
	if closeErr != nil {
		return Result{}, closeErr
	}

	all, err := assettree.Scan(root)
	if err != nil {
		return Result{}, err
	}
	if len(all.Files) != report.Outputs.PortableFiles {
		return Result{}, fmt.Errorf("portable file count %d does not match report %d", len(all.Files), report.Outputs.PortableFiles)
	}
	files := make(map[string]assettree.File, len(all.Files))
	for _, file := range all.Files {
		files[file.RelativePath] = file
		if file.RelativePath != report.Host.File && file.RelativePath != "actutum.runtime.json" && file.RelativePath != "build-result.json" && !strings.HasPrefix(file.RelativePath, "web/") {
			return Result{}, fmt.Errorf("unexpected portable file %q", file.RelativePath)
		}
	}
	host, exists := files[report.Host.File]
	if !exists || host.Size != report.Host.Bytes || host.SHA256 != report.Host.SHA256 {
		return Result{}, errors.New("host artifact does not match build result")
	}
	assets, err := assettree.Scan(filepath.Join(root, "web"))
	if err != nil {
		return Result{}, err
	}
	if err := validateAssets(report, assets); err != nil {
		return Result{}, err
	}
	runtime, err := runtimeconfig.Load(filepath.Join(root, "actutum.runtime.json"))
	if err != nil {
		return Result{}, err
	}
	if err := validateRuntime(report, runtime.Config); err != nil {
		return Result{}, err
	}
	return result("directory", report, all.TotalBytes), nil
}

func inspectZIP(archivePath string) (Result, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return Result{}, fmt.Errorf("open artifact ZIP: %w", err)
	}
	defer reader.Close()
	if len(reader.File) == 0 || len(reader.File) > maxArchiveFiles {
		return Result{}, fmt.Errorf("artifact ZIP file count %d is outside limits", len(reader.File))
	}
	if err := validateArchiveBudget(reader.File); err != nil {
		return Result{}, err
	}

	entries := make(map[string]*zip.File, len(reader.File))
	caseNames := make(map[string]string, len(reader.File))
	root := ""
	for _, entry := range reader.File {
		name := entry.Name
		if entry.FileInfo().IsDir() || strings.Contains(name, "\\") || strings.Contains(name, ":") || strings.HasPrefix(name, "/") || path.Clean(name) != name {
			return Result{}, fmt.Errorf("unsafe ZIP entry %q", name)
		}
		parts := strings.SplitN(name, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return Result{}, fmt.Errorf("ZIP entry lacks one application root: %q", name)
		}
		if root == "" {
			root = parts[0]
		} else if parts[0] != root {
			return Result{}, errors.New("ZIP contains multiple application roots")
		}
		key := strings.ToLower(name)
		if previous, exists := caseNames[key]; exists {
			return Result{}, fmt.Errorf("case-colliding ZIP entries %q and %q", previous, name)
		}
		caseNames[key] = name
		entries[parts[1]] = entry
	}

	reportEntry, exists := entries["build-result.json"]
	if !exists {
		return Result{}, errors.New("ZIP is missing build-result.json")
	}
	reportData, err := readBounded(reportEntry, maxMetadataBytes)
	if err != nil {
		return Result{}, fmt.Errorf("read build result: %w", err)
	}
	report, err := buildreport.Decode(bytes.NewReader(reportData))
	if err != nil {
		return Result{}, err
	}
	if len(entries) != report.Outputs.PortableFiles {
		return Result{}, fmt.Errorf("portable file count %d does not match report %d", len(entries), report.Outputs.PortableFiles)
	}

	var totalBytes int64
	assetFiles := make([]assettree.File, 0, report.Assets.Files)
	for relative, entry := range entries {
		if entry.UncompressedSize64 > uint64(^uint64(0)>>1) {
			return Result{}, fmt.Errorf("ZIP entry is too large: %s", relative)
		}
		size := int64(entry.UncompressedSize64)
		if size > 0 && totalBytes > int64(^uint64(0)>>1)-size {
			return Result{}, errors.New("ZIP byte count overflow")
		}
		totalBytes += size
		if strings.HasPrefix(relative, "web/") {
			digest, err := hashZIPFile(entry)
			if err != nil {
				return Result{}, fmt.Errorf("hash asset %s: %w", relative, err)
			}
			assetFiles = append(assetFiles, assettree.File{RelativePath: strings.TrimPrefix(relative, "web/"), Size: size, SHA256: digest})
		} else if relative != report.Host.File && relative != "actutum.runtime.json" && relative != "build-result.json" {
			return Result{}, fmt.Errorf("unexpected portable file %q", relative)
		}
	}
	hostEntry, exists := entries[report.Host.File]
	if !exists {
		return Result{}, errors.New("ZIP is missing the reported host")
	}
	hostDigest, err := hashZIPFile(hostEntry)
	if err != nil {
		return Result{}, err
	}
	if int64(hostEntry.UncompressedSize64) != report.Host.Bytes || hostDigest != report.Host.SHA256 {
		return Result{}, errors.New("host artifact does not match build result")
	}
	assets := assettree.Summarize(assetFiles)
	if err := validateAssets(report, assets); err != nil {
		return Result{}, err
	}
	runtimeEntry, exists := entries["actutum.runtime.json"]
	if !exists {
		return Result{}, errors.New("ZIP is missing actutum.runtime.json")
	}
	runtimeData, err := readBounded(runtimeEntry, maxMetadataBytes)
	if err != nil {
		return Result{}, fmt.Errorf("read runtime config: %w", err)
	}
	runtime, err := runtimeconfig.Parse(runtimeData)
	if err != nil {
		return Result{}, err
	}
	if err := validateRuntime(report, runtime); err != nil {
		return Result{}, err
	}
	entryKey := "web/" + path.Clean(runtime.Assets.Entry)
	if _, exists := entries[entryKey]; !exists || strings.HasPrefix(runtime.Assets.Entry, "../") || path.IsAbs(runtime.Assets.Entry) {
		return Result{}, errors.New("runtime entry point is missing or unsafe")
	}
	return result("zip", report, totalBytes), nil
}

func validateArchiveBudget(files []*zip.File) error {
	var total uint64
	for _, file := range files {
		if file.UncompressedSize64 > maxArchiveEntryBytes {
			return fmt.Errorf("ZIP entry exceeds uncompressed size limit: %s", file.Name)
		}
		if total > maxArchiveTotalBytes-file.UncompressedSize64 {
			return errors.New("artifact ZIP exceeds total uncompressed size limit")
		}
		total += file.UncompressedSize64
		if file.UncompressedSize64 > 0 && file.CompressedSize64 == 0 {
			return fmt.Errorf("ZIP entry has an invalid compression size: %s", file.Name)
		}
		if file.CompressedSize64 > 0 && file.UncompressedSize64/file.CompressedSize64 > maxArchiveExpandRatio {
			return fmt.Errorf("ZIP entry exceeds compression ratio limit: %s", file.Name)
		}
	}
	return nil
}

func validateAssets(report buildreport.Report, assets assettree.Tree) error {
	if len(assets.Files) != report.Assets.Files || assets.TotalBytes != report.Assets.Bytes || assets.Digest != report.Assets.SHA256 {
		return errors.New("asset tree does not match build result")
	}
	return nil
}

func validateRuntime(report buildreport.Report, runtime runtimeconfig.Config) error {
	if runtime.RuntimeVersion != report.Contracts.Runtime || runtime.App.ID != report.App.ID || runtime.App.Name != report.App.Name || runtime.App.Version != report.App.Version || !slices.Equal(runtime.Security.Permissions, report.Permissions) {
		return errors.New("runtime configuration does not match build result")
	}
	if runtime.Assets.Root != "web" {
		return errors.New("runtime asset root is not web")
	}
	return nil
}

func result(kind string, report buildreport.Report, bytes int64) Result {
	return Result{
		Kind: kind, ReleaseVersion: report.ReleaseVersion, App: report.App,
		Target: report.Target, Contracts: report.Contracts,
		Permissions:   append([]string{}, report.Permissions...),
		PortableFiles: report.Outputs.PortableFiles, PortableBytes: bytes,
		HostSHA256: report.Host.SHA256, AssetSHA256: report.Assets.SHA256,
	}
}

func readBounded(file *zip.File, limit int64) ([]byte, error) {
	if file.UncompressedSize64 > uint64(limit) {
		return nil, errors.New("metadata file exceeds size limit")
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, limit+1))
}

func hashZIPFile(file *zip.File) (string, error) {
	reader, err := file.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
