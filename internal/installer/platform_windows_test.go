package installer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestWindowsInstallShortcutRegistrationAndRemoval(t *testing.T) {
	source, setup, _ := fixture(t)
	root := t.TempDir()
	registryRoot := `Software\VeloxInstallerTests\` + filepath.Base(root) + `\`
	env := windowsEnvironment(filepath.Join(root, "apps"), filepath.Join(root, "menu"), registryRoot)
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, registryRoot+"dev.velox.test")
		_ = registry.DeleteKey(registry.CURRENT_USER, strings.TrimSuffix(registryRoot, `\`))
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\VeloxInstallerTests`)
	})
	result, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, registryRoot+result.AppID, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := key.GetStringValue("UninstallString")
	key.Close()
	if err != nil || !strings.Contains(command, `uninstall.exe" --uninstall dev.velox.test`) {
		t.Fatalf("uninstall command = %q: %v", command, err)
	}
	link := filepath.Join(root, "menu", result.AppID+".lnk")
	if info, err := os.Stat(link); err != nil || info.Size() < 76 {
		t.Fatalf("shortcut = %v %v", info, err)
	}
	if _, err := uninstall(result.AppID, env); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(link); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("shortcut remains")
	}
	key, err = registry.OpenKey(registry.CURRENT_USER, registryRoot+result.AppID, registry.QUERY_VALUE)
	if err == nil {
		key.Close()
		t.Fatal("registration remains")
	}
	if !errors.Is(err, registry.ErrNotExist) {
		t.Fatal(err)
	}
}
