package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const registrationIdentityValue = "VeloxInstallIdentity"

type registrationIdentity struct {
	SchemaVersion string `json:"schemaVersion"`
	Directory     string `json:"directory"`
}

type registrationValueWriter interface {
	SetStringValue(string, string) error
	SetDWordValue(string, uint32) error
}

func writeWindowsRegistration(key registrationValueWriter, record Record) error {
	identity, err := json.Marshal(registrationIdentity{stateSchema, record.Directory})
	if err != nil {
		return err
	}
	// Establish both ownership fields in one value before any display metadata.
	if err := key.SetStringValue(registrationIdentityValue, string(identity)); err != nil {
		return err
	}
	executable := filepath.Join(record.Directory, "app", record.App.ID+".exe")
	uninstaller := filepath.Join(record.Directory, "uninstall.exe")
	for _, value := range []struct{ name, text string }{
		{"VeloxInstaller", stateSchema},
		{"InstallLocation", record.Directory},
		{"DisplayIcon", executable},
		{"UninstallString", fmt.Sprintf(`"%s" --uninstall %s`, uninstaller, record.App.ID)},
		{"DisplayVersion", record.App.Version},
	} {
		if err := key.SetStringValue(value.name, value.text); err != nil {
			return err
		}
	}
	for _, name := range []string{"NoModify", "NoRepair"} {
		if err := key.SetDWordValue(name, 1); err != nil {
			return err
		}
	}
	return key.SetStringValue("DisplayName", record.App.Name)
}

func validateWindowsRegistration(key registry.Key, record Record) error {
	info, err := key.Stat()
	if err != nil {
		return err
	}
	if info.SubKeyCount != 0 {
		return errors.New("uninstall registration contains unowned subkeys")
	}
	// A stop between CreateKey and the first value leaves no registry data to
	// preserve. The caller already authenticated the installation's file record.
	if info.ValueCount == 0 {
		return nil
	}
	raw, _, err := key.GetStringValue(registrationIdentityValue)
	if errors.Is(err, registry.ErrNotExist) {
		owner, _, e1 := key.GetStringValue("VeloxInstaller")
		location, _, e2 := key.GetStringValue("InstallLocation")
		if e1 != nil || e2 != nil || owner != stateSchema || !strings.EqualFold(location, record.Directory) {
			return errors.New("uninstall registration belongs to another installation")
		}
		return nil
	}
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var identity registrationIdentity
	if err := decoder.Decode(&identity); err != nil {
		return errors.New("uninstall registration identity is invalid")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF || identity.SchemaVersion != stateSchema || !strings.EqualFold(identity.Directory, record.Directory) {
		return errors.New("uninstall registration belongs to another installation")
	}
	for _, field := range []struct{ name, expected string }{
		{"VeloxInstaller", stateSchema}, {"InstallLocation", record.Directory},
	} {
		value, _, err := key.GetStringValue(field.name)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		matches := value == field.expected
		if field.name == "InstallLocation" {
			matches = strings.EqualFold(value, field.expected)
		}
		if err != nil || !matches {
			return errors.New("uninstall registration ownership was changed")
		}
	}
	return nil
}
