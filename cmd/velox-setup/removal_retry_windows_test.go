package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestSilentRemovalHelperFailureOffersRetry(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(self, "-test.run=^TestSilentRemovalFailureHelper$")
	child.Env = append(os.Environ(), "VELOX_TEST_REMOVAL_FAILURE=1")
	output, err := child.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 6 {
		t.Fatalf("helper exit=%v output=%s", err, output)
	}
	if !strings.Contains(string(output), "To retry removal") || !strings.Contains(string(output), "& '"+strings.ReplaceAll(self, "'", "''")+"' --uninstall ") {
		t.Fatalf("no usable helper retry command: %s", output)
	}
}

func TestSilentRemovalFailureHelper(t *testing.T) {
	if os.Getenv("VELOX_TEST_REMOVAL_FAILURE") != "1" {
		t.Skip("isolated child entry-point check")
	}
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal(err)
	}
	id := "dev.velox.retry-missing-" + hex.EncodeToString(token[:])
	local, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(local, "Programs", "Velox", id)); !os.IsNotExist(err) {
		t.Fatalf("missing-installation fixture is not absent: %v", err)
	}
	os.Exit(run([]string{"--uninstall", id, "--remove-helper", "--wait-pid", "4294967295", "--silent"}))
}

func TestRetryRemovalDecisions(t *testing.T) {
	for _, test := range []struct {
		name                     string
		silent, retry            bool
		failures, calls, prompts int
		wantError                bool
	}{
		{"success", false, false, 0, 1, 0, false},
		{"silent-failure", true, true, 1, 1, 0, true},
		{"cancel", false, false, 1, 1, 1, true},
		{"retry-once", false, true, 1, 2, 1, false},
		{"retry-twice", false, true, 2, 3, 2, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cause := errors.New("locked directory")
			calls, prompts := 0, 0
			err := retryRemoval(`C:\private helper\remove.exe`, "dev.velox.test", test.silent, func() error {
				calls++
				if calls <= test.failures {
					return cause
				}
				return nil
			}, func(text string) bool {
				prompts++
				if !strings.Contains(text, "locked directory") || !strings.Contains(text, "Retry now?") ||
					!strings.Contains(text, `& 'C:\private helper\remove.exe' --uninstall dev.velox.test`) {
					t.Fatalf("unusable retry dialog: %q", text)
				}
				return test.retry
			})
			if calls != test.calls || prompts != test.prompts || (err != nil) != test.wantError {
				t.Fatalf("calls=%d prompts=%d err=%v", calls, prompts, err)
			}
			if err != nil && !errors.Is(err, cause) {
				t.Fatal("original cause was lost")
			}
		})
	}
}

func TestRetryCommandQuotesPowerShellPath(t *testing.T) {
	self := "C:\\O'Brien\\$cache`name\\remove.exe"
	err := retryRemoval(self, "dev.velox.test", true, func() error { return errors.New("locked") }, nil)
	if err == nil || !strings.Contains(err.Error(), "& 'C:\\O''Brien\\$cache`name\\remove.exe' --uninstall dev.velox.test") {
		t.Fatalf("retry command did not use a literal PowerShell path: %v", err)
	}
}
