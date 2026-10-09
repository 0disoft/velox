package installer

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
	"strings"

	"github.com/0disoft/velox/internal/appidentity"
	"github.com/0disoft/velox/internal/assettree"
	"github.com/0disoft/velox/internal/buildreport"
	"github.com/0disoft/velox/internal/inspector"
	"github.com/0disoft/velox/internal/safefs"
)

const stateFile = "velox-install.json"
const stateSchema = "velox.install/v1"

type Record struct {
	SchemaVersion string          `json:"schemaVersion"`
	App           buildreport.App `json:"app"`
	Directory     string          `json:"directory"`
	Files         []File          `json:"files"`
	ShortcutHash  string          `json:"shortcutSha256"`
}

type File struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type Result struct {
	AppID     string `json:"appId"`
	Directory string `json:"directory"`
}

type environment struct {
	root            string
	check           func(Record) error
	prepare         func(Record) (registration, error)
	unregister      func(Record) error
	validateRemoval func(Record) error
	removable       func(string) error
}

type registration struct {
	shortcutHash string
	publish      func() error
	cleanup      func()
}

// Install creates a fresh per-user installation from an inspected portable
// directory and a prebuilt uninstaller. Existing installations are never replaced.
func Install(source, uninstaller string) (Result, error) {
	env, err := userEnvironment()
	if err != nil {
		return Result{}, err
	}
	return install(source, uninstaller, env)
}

func install(source, uninstaller string, env environment) (Result, error) {
	if err := safefs.RejectLinkedComponents(source); err != nil {
		return Result{}, err
	}
	inspected, err := inspector.Inspect(source)
	if err != nil {
		return Result{}, err
	}
	if inspected.Kind != "directory" {
		return Result{}, errors.New("a verified portable directory is required")
	}
	if err := appidentity.Validate(inspected.App.ID); err != nil {
		return Result{}, err
	}
	target := filepath.Join(env.root, inspected.App.ID)
	record := Record{SchemaVersion: stateSchema, App: inspected.App, Directory: target}
	if err := safefs.RejectLinkedComponents(target); err != nil {
		return Result{}, err
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return Result{}, errors.New("installation already exists; uninstall it before installing again")
	}
	if err := env.check(record); err != nil {
		return Result{}, err
	}
	if err := safefs.EnsureDirectory(env.root, 0o755); err != nil {
		return Result{}, err
	}
	stage, err := os.MkdirTemp(env.root, ".velox-install-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(stage)
	app := filepath.Join(stage, "app")
	tree, err := assettree.Scan(source)
	if err != nil {
		return Result{}, err
	}
	for _, file := range tree.Files {
		relative := "app/" + file.RelativePath
		copied, err := copyFile(file.SourcePath, filepath.Join(stage, filepath.FromSlash(relative)))
		if err != nil {
			return Result{}, err
		}
		if copied.Bytes != file.Size || copied.SHA256 != file.SHA256 {
			return Result{}, errors.New("portable source changed while installing")
		}
		copied.Path = relative
		record.Files = append(record.Files, copied)
	}
	if _, err := inspector.Inspect(app); err != nil {
		return Result{}, fmt.Errorf("verify staged installation: %w", err)
	}
	file, err := copyFile(uninstaller, filepath.Join(stage, "uninstall.exe"))
	if err != nil {
		return Result{}, err
	}
	file.Path = "uninstall.exe"
	record.Files = append(record.Files, file)
	prepared, err := env.prepare(record)
	if err != nil {
		return Result{}, err
	}
	defer prepared.cleanup()
	record.ShortcutHash = prepared.shortcutHash
	// Commit complete ownership before either the installation or shortcut is
	// published; an interruption after publication must not need a final update.
	if err := writeState(stage, record); err != nil {
		return Result{}, err
	}
	if err := os.Rename(stage, target); err != nil {
		return Result{}, err
	}
	if err := prepared.publish(); err != nil {
		cleanupErr := os.RemoveAll(target)
		return Result{}, errors.Join(err, cleanupErr)
	}
	return Result{AppID: record.App.ID, Directory: target}, nil
}

// Uninstall removes only files recorded by this installer. User data is outside
// this tree and is never touched. Changed or additional files cause refusal.
func Uninstall(appID string) (Result, error) {
	env, err := userEnvironment()
	if err != nil {
		return Result{}, err
	}
	return uninstall(appID, env)
}

func uninstall(appID string, env environment) (Result, error) {
	if err := appidentity.Validate(appID); err != nil {
		return Result{}, err
	}
	target := filepath.Join(env.root, appID)
	record, err := readState(target, appID)
	if err != nil {
		return Result{}, err
	}
	if err := env.validateRemoval(record); err != nil {
		return Result{}, err
	}
	tree, err := assettree.Scan(target)
	if err != nil {
		return Result{}, err
	}
	expected := make(map[string]File, len(record.Files))
	for _, file := range record.Files {
		expected[file.Path] = file
	}
	for _, file := range tree.Files {
		if file.RelativePath == stateFile {
			continue
		}
		owned, ok := expected[file.RelativePath]
		if !ok || owned.Bytes != file.Size || owned.SHA256 != file.SHA256 {
			return Result{}, fmt.Errorf("preserving changed or unowned file %q; move it before uninstalling", file.RelativePath)
		}
		if err := env.removable(file.SourcePath); err != nil {
			return Result{}, fmt.Errorf("close the app before uninstalling: %w", err)
		}
	}
	// Missing recorded files are allowed so a failed partial removal can be retried.
	for _, file := range record.Files {
		if err := os.Remove(filepath.Join(target, filepath.FromSlash(file.Path))); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Result{}, err
		}
	}
	if err := env.unregister(record); err != nil {
		return Result{}, err
	}
	if err := os.Remove(filepath.Join(target, stateFile)); err != nil {
		return Result{}, err
	}
	var directories []string
	if err := filepath.WalkDir(target, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			directories = append(directories, path)
		}
		return nil
	}); err != nil {
		return Result{}, err
	}
	sort.Slice(directories, func(i, j int) bool { return len(directories[i]) > len(directories[j]) })
	for _, directory := range directories {
		if err := os.Remove(directory); err != nil {
			return Result{}, err
		}
	}
	return Result{AppID: appID, Directory: target}, nil
}

func readState(root, appID string) (Record, error) {
	file, _, err := safefs.OpenVerifiedRegular(filepath.Join(root, stateFile))
	if err != nil {
		return Record{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 16<<20))
	decoder.DisallowUnknownFields()
	var record Record
	if err := decoder.Decode(&record); err != nil {
		return Record{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Record{}, errors.New("invalid installation state trailing data")
	}
	if record.SchemaVersion != stateSchema || record.App.ID != appID || !strings.EqualFold(record.Directory, root) || len(record.Files) < 4 || len(record.Files) > 100_001 {
		return Record{}, errors.New("installation ownership does not match")
	}
	seen := make(map[string]bool)
	for _, file := range record.Files {
		key := strings.ToLower(file.Path)
		if safefs.ValidateArchiveEntry(file.Path) != nil || (file.Path != "uninstall.exe" && !strings.HasPrefix(file.Path, "app/")) || seen[key] || file.Bytes < 0 || len(file.SHA256) != 64 {
			return Record{}, errors.New("invalid installed file record")
		}
		if _, err := hex.DecodeString(file.SHA256); err != nil {
			return Record{}, err
		}
		seen[key] = true
	}
	if !seen["uninstall.exe"] || !seen["app/"+appID+".exe"] || !seen["app/build-result.json"] || !seen["app/velox.runtime.json"] {
		return Record{}, errors.New("installed file record is incomplete")
	}
	return record, nil
}

func copyFile(source, target string) (File, error) {
	input, info, err := safefs.OpenVerifiedRegular(source)
	if err != nil {
		return File{}, err
	}
	defer input.Close()
	if info.Size() > 512<<20 {
		return File{}, errors.New("installer input is too large")
	}
	if err := safefs.EnsureDirectory(filepath.Dir(target), 0o755); err != nil {
		return File{}, err
	}
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return File{}, err
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(output, hash), io.LimitReader(input, info.Size()+1))
	closeErr := output.Close()
	if size != info.Size() {
		copyErr = errors.Join(copyErr, errors.New("installer input changed size"))
	}
	return File{Bytes: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, errors.Join(copyErr, closeErr)
}
