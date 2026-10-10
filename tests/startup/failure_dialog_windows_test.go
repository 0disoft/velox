package startup_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestBuiltHostStartupFailureDialogs(t *testing.T) {
	host := goHost(t, repositoryRoot(t))
	for _, scenario := range []string{"configuration", "profile"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			profile := filepath.Join(root, "private-profile")
			args := host.arguments(profile)
			wantCode, wantText := 2, "configuration or packaged assets"
			if scenario == "configuration" {
				args = []string{"--config", filepath.Join(root, "private-missing.json")}
			} else {
				wantCode, wantText = 5, "application data folder"
				if err := os.WriteFile(profile, []byte("private existing data"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command(host.executable, args...)
			cmd.Env = append(os.Environ(), "VELOX_BENCH_MODE=", "VELOX_DATA_DIR="+profile)
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			waited := false
			defer func() {
				if !waited {
					_ = cmd.Process.Kill()
					<-done
				}
			}()
			user32 := windows.NewLazySystemDLL("user32.dll")
			var dialog uintptr
			deadline := time.Now().Add(5 * time.Second)
			for dialog == 0 && time.Now().Before(deadline) {
				hwnd, _, _ := user32.NewProc("GetTopWindow").Call(0)
				for hwnd != 0 {
					var pid uint32
					user32.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&pid)))
					if pid == uint32(cmd.Process.Pid) && startupDialogText(user32, hwnd) == "Velox - Unable to start" {
						dialog = hwnd
						break
					}
					hwnd, _, _ = user32.NewProc("GetWindow").Call(hwnd, 2)
				}
				if dialog == 0 {
					time.Sleep(10 * time.Millisecond)
				}
			}
			if dialog == 0 {
				t.Fatal("owned startup failure dialog not found")
			}
			var text string
			// The dialog can become enumerable before its static controls are ready.
			for time.Now().Before(deadline) {
				child, _, _ := user32.NewProc("GetWindow").Call(dialog, 5)
				var body strings.Builder
				for child != 0 {
					body.WriteString(startupDialogText(user32, child))
					child, _, _ = user32.NewProc("GetWindow").Call(child, 2)
				}
				text = body.String()
				if strings.Contains(text, wantText) {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !strings.Contains(text, wantText) || strings.Contains(text, root) || strings.Contains(text, "private") {
				t.Fatalf("unexpected or sensitive dialog: %q", text)
			}
			ok, _, err := user32.NewProc("PostMessageW").Call(dialog, 0x0010, 0, 0)
			if ok == 0 {
				t.Fatalf("dismiss owned dialog: %v", err)
			}
			select {
			case err := <-done:
				waited = true
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != wantCode || !strings.Contains(output.String(), "velox-host:") {
					t.Fatalf("exit/stderr changed: %v %q", err, output.String())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("host did not exit after notice dismissal")
			}
			if scenario == "profile" {
				body, err := os.ReadFile(profile)
				if err != nil || string(body) != "private existing data" {
					t.Fatalf("existing profile changed: %q %v", body, err)
				}
			}
		})
	}
}

func startupDialogText(user32 *windows.LazyDLL, hwnd uintptr) string {
	var text [512]uint16
	user32.NewProc("GetWindowTextW").Call(hwnd, uintptr(unsafe.Pointer(&text[0])), uintptr(len(text)))
	return windows.UTF16ToString(text[:])
}
