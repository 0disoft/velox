package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/velox/internal/buildreport"
	"github.com/0disoft/velox/internal/runtimeconfig"
	"golang.org/x/sys/windows/registry"
)

func TestWindowsRegistryProcessInterruptionAndReinstall(t *testing.T) {
	for cut := 0; cut <= 9; cut++ {
		t.Run(fmt.Sprintf("after-%d-writes", cut), func(t *testing.T) {
			source, setup, _ := fixture(t)
			root := t.TempDir()
			registryRoot := testRegistrationNamespace(root) + `\`
			keyPath := registryRoot + "dev.velox.test"
			t.Cleanup(func() {
				_ = registry.DeleteKey(registry.CURRENT_USER, keyPath)
				_ = registry.DeleteKey(registry.CURRENT_USER, strings.TrimSuffix(registryRoot, `\`))
				_ = registry.DeleteKey(registry.CURRENT_USER, `Software\VeloxInstallerTests`)
			})
			env := windowsEnvironment(filepath.Join(root, "apps"), filepath.Join(root, "menu"), registryRoot)
			prepare := env.prepare
			env.prepare = func(record Record) (registration, error) {
				prepared, err := prepare(record)
				// The child performs the real registry writes after the owned tree
				// is promoted, without publishing a final shortcut in this fixture.
				prepared.publish = func() error { return nil }
				return prepared, err
			}
			installed, err := install(source, setup, env)
			if err != nil {
				t.Fatal(err)
			}
			draft := filepath.Join(root, "user-draft.md")
			if err := os.WriteFile(draft, []byte("retained draft\r\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			self, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, self, "-test.run=^TestWindowsRegistrationInterruptionHelper$")
			child.Env = append(os.Environ(), "VELOX_REGISTRATION_CUT_DIRECTORY="+installed.Directory, "VELOX_REGISTRATION_CUT="+strconv.Itoa(cut))
			output, err := child.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 97 {
				t.Fatalf("child did not exit at boundary %d: %v; %s", cut, err, output)
			}
			key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
			if err != nil {
				t.Fatal(err)
			}
			info, err := key.Stat()
			if err != nil || int(info.ValueCount) != cut {
				key.Close()
				t.Fatalf("unexpected interrupted key: %+v: %v", info, err)
			}
			if cut > 0 {
				if _, _, err := key.GetStringValue(registrationIdentityValue); err != nil {
					key.Close()
					t.Fatalf("ownership was not first: %v", err)
				}
			}
			key.Close()
			if _, err := uninstall(installed.AppID, env); err != nil {
				t.Fatalf("cannot remove interrupted registration: %v", err)
			}
			setInstallerFixtureVersion(t, source, "2.0.0")
			env.prepare = prepare
			reinstalled, err := install(source, setup, env)
			if err != nil {
				t.Fatalf("cannot reinstall after interruption recovery: %v", err)
			}
			record, err := readState(reinstalled.Directory, reinstalled.AppID)
			if err != nil || record.App.Version != "2.0.0" {
				t.Fatalf("new version not installed: %+v %v", record.App, err)
			}
			key, err = registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
			if err != nil {
				t.Fatal(err)
			}
			version, _, err := key.GetStringValue("DisplayVersion")
			key.Close()
			if err != nil || version != "2.0.0" {
				t.Fatalf("Apps registration version=%q: %v", version, err)
			}
			if _, err := uninstall(reinstalled.AppID, env); err != nil {
				t.Fatal(err)
			}
			if data, err := os.ReadFile(draft); err != nil || string(data) != "retained draft\r\n" {
				t.Fatalf("user draft changed: %v", err)
			}
		})
	}
}

type interruptedRegistrationWriter struct {
	key         registry.Key
	writes, cut int
}

func (w *interruptedRegistrationWriter) wrote(err error) error {
	if err == nil {
		w.writes++
		if w.writes == w.cut {
			os.Exit(97)
		}
	}
	return err
}

func (w *interruptedRegistrationWriter) SetStringValue(name, value string) error {
	return w.wrote(w.key.SetStringValue(name, value))
}

func (w *interruptedRegistrationWriter) SetDWordValue(name string, value uint32) error {
	return w.wrote(w.key.SetDWordValue(name, value))
}

func TestWindowsRegistrationInterruptionHelper(t *testing.T) {
	directory := os.Getenv("VELOX_REGISTRATION_CUT_DIRECTORY")
	if directory == "" {
		t.Skip("isolated child-process fixture only")
	}
	cut, err := strconv.Atoi(os.Getenv("VELOX_REGISTRATION_CUT"))
	if err != nil || cut < 0 || cut > 9 {
		t.Fatal("invalid cut point")
	}
	record, err := readState(directory, "dev.velox.test")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(directory))
	keyPath := testRegistrationNamespace(root) + `\dev.velox.test`
	key, existing, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil || existing {
		t.Fatalf("isolated registration creation: existing=%v error=%v", existing, err)
	}
	defer key.Close()
	if cut == 0 {
		os.Exit(97)
	}
	if err := writeWindowsRegistration(&interruptedRegistrationWriter{key: key, cut: cut}, record); err != nil {
		t.Fatal(err)
	}
	// os.Exit deliberately bypasses Go defers at each real registry boundary.
	t.Fatal("did not reach requested process exit boundary")
}

func setInstallerFixtureVersion(t *testing.T, source, version string) {
	t.Helper()
	for name, value := range map[string]any{
		"build-result.json":  &buildreport.Report{},
		"velox.runtime.json": &runtimeconfig.Config{},
	} {
		path := filepath.Join(source, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, value); err != nil {
			t.Fatal(err)
		}
		switch value := value.(type) {
		case *buildreport.Report:
			value.App.Version = version
		case *runtimeconfig.Config:
			value.App.Version = version
		}
		data, err = json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
