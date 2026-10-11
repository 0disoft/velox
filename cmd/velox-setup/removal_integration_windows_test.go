package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/0disoft/velox/internal/assettree"
	"github.com/0disoft/velox/internal/buildreport"
	"github.com/0disoft/velox/internal/installer"
	"github.com/0disoft/velox/internal/runtimeconfig"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func TestBuiltSetupLateRemovalRetry(t *testing.T) {
	setup := os.Getenv("VELOX_BUILT_SETUP")
	if setup == "" {
		t.Skip("VELOX_BUILT_SETUP enables this optional real Setup helper test")
	}
	setup, err := filepath.Abs(setup)
	if err != nil {
		t.Fatal(err)
	}
	image, err := pe.Open(setup)
	if err != nil {
		t.Fatal(err)
	}
	header, ok := image.OptionalHeader.(*pe.OptionalHeader64)
	image.Close()
	if !ok || header.Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI {
		t.Fatal("a source-built Windows x64 GUI Setup is required")
	}
	bytes, err := os.ReadFile(setup)
	if err != nil {
		t.Fatal(err)
	}
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal(err)
	}
	id := "dev.velox.retry-" + hex.EncodeToString(token[:])
	local, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
	if err != nil {
		t.Fatal(err)
	}
	programs, err := windows.KnownFolderPath(windows.FOLDERID_Programs, 0)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(local, "Programs", "Velox", id)
	profile := filepath.Join(local, "Velox", "profiles", id)
	shortcut := filepath.Join(programs, "Velox", id+".lnk")
	keyPath := `Software\Microsoft\Windows\CurrentVersion\Uninstall\Velox.` + id
	for _, path := range []string{target, profile, shortcut} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("test identity already exists: %q %v", path, err)
		}
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err == nil {
		key.Close()
		t.Fatal("test registration already exists")
	}
	if !errors.Is(err, registry.ErrNotExist) {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.Mkdir(filepath.Join(source, "web"), 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(source, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON := func(name string, value any) {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		write(name, data)
	}
	write("web/index.html", []byte("<!doctype html><title>Installer fixture</title>"))
	// This is an installer/removal fixture, not a launched consumer host.
	write(id+".exe", bytes)
	tree, err := assettree.Scan(filepath.Join(source, "web"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(bytes)
	app := buildreport.App{ID: id, Name: "Velox Removal Retry Fixture", Version: "0.1.0"}
	writeJSON("build-result.json", buildreport.Report{
		SchemaVersion: buildreport.SchemaVersion, ReleaseVersion: "test", Target: "windows-x64", App: app,
		Contracts:   buildreport.Contracts{Manifest: 1, Runtime: 1, Host: 1, IPC: 1},
		Host:        buildreport.File{File: id + ".exe", Bytes: int64(len(bytes)), SHA256: hex.EncodeToString(digest[:])},
		Assets:      buildreport.Assets{Files: 1, Bytes: tree.TotalBytes, SHA256: tree.Digest},
		Permissions: []string{}, Outputs: buildreport.OutputCounts{PortableFiles: 4},
	})
	writeJSON("velox.runtime.json", runtimeconfig.Config{
		RuntimeVersion: 1, App: runtimeconfig.App{ID: app.ID, Name: app.Name, Version: app.Version},
		Assets: runtimeconfig.Assets{Root: "web", Entry: "index.html"},
		Window: runtimeconfig.Window{Width: 640, Height: 480}, Security: runtimeconfig.Security{Permissions: []string{}},
	})
	if _, err := installer.Install(source, setup); err != nil {
		t.Fatal(err)
	}
	lock := windows.InvalidHandle
	t.Cleanup(func() {
		if lock != windows.InvalidHandle {
			_ = windows.CloseHandle(lock)
		}
		if _, err := installer.Uninstall(id); err != nil && !os.IsNotExist(err) {
			t.Errorf("test installation cleanup failed: %v", err)
		}
	})
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(profile, "owned-test-draft.txt")
	if err := os.WriteFile(marker, []byte("retained draft\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(marker)
		if err := os.Remove(profile); err != nil {
			t.Errorf("test profile retained: %v", err)
		}
	})
	name, err := windows.UTF16PtrFromString(target)
	if err != nil {
		t.Fatal(err)
	}
	lock, err = windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	failedHelper := exec.CommandContext(ctx, setup, "--uninstall", id, "--remove-helper", "--wait-pid", "4294967295", "--silent")
	output, err := failedHelper.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 6 || !strings.Contains(string(output), "& '"+strings.ReplaceAll(setup, "'", "''")+"' --uninstall "+id) {
		t.Fatalf("built helper did not expose its retry command: %v; %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(target, "uninstall.exe")); !os.IsNotExist(err) {
		t.Fatalf("fixture did not reach late failure: %v", err)
	}
	if err := windows.CloseHandle(lock); err != nil {
		t.Fatal(err)
	}
	lock = windows.InvalidHandle
	retry := exec.CommandContext(ctx, setup, "--uninstall", id, "--silent")
	output, err = retry.CombinedOutput()
	if err != nil {
		t.Fatalf("retry launcher failed: %v; %s", err, output)
	}
	helperDirectory := strings.TrimSpace(string(output))
	if filepath.Dir(helperDirectory) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(helperDirectory), "velox-uninstall-") {
		t.Fatalf("unexpected private helper location: %q", helperDirectory)
	}
	awaitOwnedRemovalHelper(t, filepath.Join(helperDirectory, "remove.exe"))
	t.Cleanup(func() {
		if err := os.RemoveAll(helperDirectory); err != nil {
			t.Errorf("owned helper cleanup failed: %v", err)
		}
	})
	for _, path := range []string{target, shortcut, filepath.Join(filepath.Dir(target), ".velox-removal-"+id+".json")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("retry retained %q: %v", path, err)
		}
	}
	key, err = registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err == nil {
		key.Close()
		t.Fatal("retry retained uninstall registration")
	}
	if !errors.Is(err, registry.ErrNotExist) {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(marker); err != nil || string(data) != "retained draft\r\n" {
		t.Fatalf("real helper changed the draft marker: %v", err)
	}
	t.Logf("built Setup sha256=%x app=%s: late failure=6; public retry launcher=0; installation, shortcut, registry and journal removed; profile marker preserved", digest, id)
}

func awaitOwnedRemovalHelper(t *testing.T, executable string) {
	t.Helper()
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	var unresolved error
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		if !strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), filepath.Base(executable)) {
			continue
		}
		process, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
		if err != nil {
			if !errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
				unresolved = err
			}
			continue
		}
		buffer := make([]uint16, 32768)
		size := uint32(len(buffer))
		queryErr := windows.QueryFullProcessImageName(process, 0, &buffer[0], &size)
		if queryErr != nil {
			unresolved = queryErr
			windows.CloseHandle(process)
			continue
		}
		if !strings.EqualFold(windows.UTF16ToString(buffer[:size]), executable) {
			windows.CloseHandle(process)
			continue
		}
		status, err := windows.WaitForSingleObject(process, 10000)
		var code uint32
		exitErr := windows.GetExitCodeProcess(process, &code)
		windows.CloseHandle(process)
		if err != nil || status != windows.WAIT_OBJECT_0 || exitErr != nil || code != 0 {
			t.Fatalf("owned helper completion: wait=%d err=%v exit=%d err=%v", status, err, code, exitErr)
		}
		return
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		t.Fatal(err)
	}
	if unresolved != nil {
		t.Fatalf("cannot confirm absence of owned helper: %v", unresolved)
	}
	// The launched helper can finish before enumeration. The absence is an
	// authoritative process snapshot; callers still check every removal result.
	t.Log("owned helper already absent from process snapshot; exit code unobserved")
}
