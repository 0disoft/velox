package buildplan

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0disoft/velox/internal/assettree"
	"github.com/0disoft/velox/internal/manifest"
)

const TargetWindowsX64 = "windows-x64"

type Options struct {
	ManifestPath string
	HostPath     string
	OutputRoot   string
	Target       string
}

type ErrorKind string

const (
	ErrorManifest ErrorKind = "manifest"
	ErrorAsset    ErrorKind = "asset"
	ErrorHost     ErrorKind = "host"
	ErrorConfig   ErrorKind = "config"
)

type Error struct {
	Kind ErrorKind
	Err  error
}

func (err *Error) Error() string { return err.Err.Error() }
func (err *Error) Unwrap() error { return err.Err }

func fail(kind ErrorKind, err error) error {
	return &Error{Kind: kind, Err: err}
}

type Plan struct {
	manifest       manifest.Resolved
	assets         assettree.Tree
	hostPath       string
	hostSHA256     string
	hostSize       int64
	target         string
	outputRoot     string
	appDirectory   string
	archivePath    string
	applicationKey string
}

type Snapshot struct {
	Manifest       manifest.Resolved
	Assets         assettree.Tree
	HostPath       string
	HostSHA256     string
	HostSize       int64
	Target         string
	OutputRoot     string
	AppDirectory   string
	ArchivePath    string
	ApplicationKey string
}

func Create(options Options) (Plan, error) {
	if options.Target == "" {
		options.Target = TargetWindowsX64
	}
	if options.Target != TargetWindowsX64 {
		return Plan{}, fail(ErrorConfig, fmt.Errorf("unsupported target %q", options.Target))
	}
	resolved, err := manifest.Load(options.ManifestPath)
	if err != nil {
		return Plan{}, fail(ErrorManifest, err)
	}
	assets, err := assettree.Scan(resolved.AssetRoot)
	if err != nil {
		return Plan{}, fail(ErrorAsset, err)
	}
	entryRelative, err := filepath.Rel(resolved.AssetRoot, resolved.EntryPath)
	if err != nil {
		return Plan{}, fail(ErrorAsset, fmt.Errorf("resolve entry point: %w", err))
	}
	entryKey := filepath.ToSlash(entryRelative)
	entryFound := false
	for _, file := range assets.Files {
		if file.RelativePath == entryKey {
			entryFound = true
			break
		}
	}
	if !entryFound {
		return Plan{}, fail(ErrorAsset, fmt.Errorf("entry point is missing or not a regular asset: %s", entryKey))
	}

	if options.HostPath == "" {
		return Plan{}, fail(ErrorHost, errors.New("host template path is required"))
	}
	hostPath, err := filepath.Abs(options.HostPath)
	if err != nil {
		return Plan{}, fail(ErrorHost, fmt.Errorf("resolve host template: %w", err))
	}
	if err := rejectRedirectedPath(hostPath); err != nil {
		return Plan{}, fail(ErrorHost, fmt.Errorf("validate host template path: %w", err))
	}
	hostInfo, err := os.Lstat(hostPath)
	if err != nil {
		return Plan{}, fail(ErrorHost, fmt.Errorf("inspect host template: %w", err))
	}
	if !hostInfo.Mode().IsRegular() || hostInfo.Mode()&os.ModeSymlink != 0 {
		return Plan{}, fail(ErrorHost, errors.New("host template must be a regular file"))
	}
	hostDigest, err := hashFile(hostPath)
	if err != nil {
		return Plan{}, fail(ErrorHost, fmt.Errorf("hash host template: %w", err))
	}

	outputCandidate := options.OutputRoot
	if outputCandidate == "" {
		outputCandidate = "dist"
	}
	if !filepath.IsAbs(outputCandidate) {
		outputCandidate = filepath.Join(resolved.ProjectRoot, outputCandidate)
	}
	outputRoot, err := filepath.Abs(outputCandidate)
	if err != nil {
		return Plan{}, fail(ErrorConfig, fmt.Errorf("resolve output root: %w", err))
	}
	if err := rejectRedirectedPath(outputRoot); err != nil {
		return Plan{}, fail(ErrorConfig, fmt.Errorf("validate output root: %w", err))
	}
	if containsPath(resolved.AssetRoot, outputRoot) || containsPath(outputRoot, resolved.AssetRoot) {
		return Plan{}, fail(ErrorConfig, errors.New("output root and asset root must not contain each other"))
	}
	applicationKey := applicationKey(resolved.App.ID)
	return Plan{
		manifest:       resolved,
		assets:         assets,
		hostPath:       hostPath,
		hostSHA256:     hostDigest,
		hostSize:       hostInfo.Size(),
		target:         options.Target,
		outputRoot:     outputRoot,
		appDirectory:   filepath.Join(outputRoot, applicationKey),
		archivePath:    filepath.Join(outputRoot, applicationKey+".zip"),
		applicationKey: applicationKey,
	}, nil
}

func (plan Plan) AssetPaths() []string {
	paths := make([]string, len(plan.assets.Files))
	for index, file := range plan.assets.Files {
		paths[index] = file.RelativePath
	}
	sort.Strings(paths)
	return paths
}

func (plan Plan) Snapshot() Snapshot {
	resolved := plan.manifest
	resolved.Security.Permissions = append([]string(nil), plan.manifest.Security.Permissions...)
	assets := plan.assets
	assets.Files = append([]assettree.File(nil), plan.assets.Files...)
	return Snapshot{
		Manifest: resolved, Assets: assets,
		HostPath: plan.hostPath, HostSHA256: plan.hostSHA256, HostSize: plan.hostSize,
		Target: plan.target, OutputRoot: plan.outputRoot,
		AppDirectory: plan.appDirectory, ArchivePath: plan.archivePath,
		ApplicationKey: plan.applicationKey,
	}
}

func applicationKey(appID string) string {
	parts := strings.Split(appID, ".")
	return parts[len(parts)-1]
}

func containsPath(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func rejectRedirectedPath(path string) error {
	existing := filepath.Clean(path)
	for {
		_, err := os.Lstat(existing)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return err
		}
		existing = parent
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return err
	}
	left := filepath.Clean(existing)
	right := filepath.Clean(resolved)
	if left != right && !strings.EqualFold(left, right) {
		return fmt.Errorf("path traverses a symbolic link or reparse point at %s", filepath.Base(existing))
	}
	return nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
