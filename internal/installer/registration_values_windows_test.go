package installer

import (
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/buildreport"
	"golang.org/x/sys/windows/registry"
)

func TestWindowsRegistrationPreservesChangedTypesAndFlags(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(registry.Key, Record) error
	}{
		{"qword-flag", func(k registry.Key, _ Record) error { return k.SetQWordValue("NoModify", 1) }},
		{"changed-flag", func(k registry.Key, _ Record) error { return k.SetDWordValue("NoRepair", 0) }},
		{"expandable-owner", func(k registry.Key, _ Record) error { return k.SetExpandStringValue("VeloxInstaller", stateSchema) }},
		{"expandable-identity", func(k registry.Key, r Record) error {
			return k.SetExpandStringValue(registrationIdentityValue, testRegistrationIdentity(r.Directory))
		}},
		{"foreign-binary", func(k registry.Key, _ Record) error { return k.SetBinaryValue("UserNotes", []byte("keep me")) }},
		{"oversized-display", func(k registry.Key, _ Record) error {
			return k.SetBinaryValue("DisplayName", make([]byte, (128<<10)+1))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := testRegistrationNamespace(root)
			key, existed, err := registry.CreateKey(registry.CURRENT_USER, path, registry.ALL_ACCESS)
			if err != nil || existed {
				t.Fatalf("fresh isolated key: existed=%v err=%v", existed, err)
			}
			t.Cleanup(func() {
				key.Close()
				_ = registry.DeleteKey(registry.CURRENT_USER, path)
				_ = registry.DeleteKey(registry.CURRENT_USER, `Software\VeloxInstallerTests`)
			})
			record := Record{SchemaVersion: stateSchema, Directory: filepath.Join(root, "app"),
				App: buildreport.App{ID: "dev.velox.test", Name: "Test", Version: "1.0.0"}}
			if err := writeWindowsRegistration(key, record); err != nil {
				t.Fatal(err)
			}
			if err := validateWindowsRegistration(key, record); err != nil {
				t.Fatalf("valid written registration was rejected: %v", err)
			}
			if err := test.change(key, record); err != nil {
				t.Fatal(err)
			}
			if err := validateWindowsRegistration(key, record); err == nil {
				t.Fatal("changed value type, flag or foreign data was accepted")
			}
		})
	}
}
