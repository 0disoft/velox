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

type registrationString struct {
	name, text string
	path       bool
}

func windowsRegistrationStrings(record Record) []registrationString {
	executable := filepath.Join(record.Directory, "app", record.App.ID+".exe")
	uninstaller := filepath.Join(record.Directory, "uninstall.exe")
	return []registrationString{
		{"VeloxInstaller", stateSchema, false},
		{"InstallLocation", record.Directory, true},
		{"DisplayIcon", executable, true},
		{"UninstallString", fmt.Sprintf(`"%s" --uninstall %s`, uninstaller, record.App.ID), true},
		{"DisplayVersion", record.App.Version, false},
		{"DisplayName", record.App.Name, false},
	}
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
	values := windowsRegistrationStrings(record)
	for _, value := range values[:len(values)-1] {
		if err := key.SetStringValue(value.name, value.text); err != nil {
			return err
		}
	}
	for _, name := range []string{"NoModify", "NoRepair"} {
		if err := key.SetDWordValue(name, 1); err != nil {
			return err
		}
	}
	last := values[len(values)-1]
	return key.SetStringValue(last.name, last.text)
}

func validateWindowsRegistration(key registry.Key, record Record) error {
	info, err := key.Stat()
	if err != nil {
		return err
	}
	if info.SubKeyCount != 0 {
		return errors.New("uninstall registration contains unowned subkeys")
	}
	if info.ValueCount > 9 || info.MaxValueNameLen > 64 || info.MaxValueLen > 128<<10 {
		return errors.New("uninstall registration contains unowned or oversized values")
	}
	// A stop between CreateKey and the first value leaves no registry data to
	// preserve. The caller already authenticated the installation's file record.
	if info.ValueCount == 0 {
		return nil
	}
	raw, kind, err := key.GetStringValue(registrationIdentityValue)
	if errors.Is(err, registry.ErrNotExist) {
		owner, _, e1 := key.GetStringValue("VeloxInstaller")
		location, _, e2 := key.GetStringValue("InstallLocation")
		if e1 != nil || e2 != nil || owner != stateSchema || !strings.EqualFold(location, record.Directory) {
			return errors.New("uninstall registration belongs to another installation")
		}
		return validateWindowsRegistrationValues(key, record)
	}
	if err != nil {
		return err
	}
	if kind != registry.SZ {
		return errors.New("uninstall registration identity type was changed")
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
	return validateWindowsRegistrationValues(key, record)
}

func validateWindowsRegistrationValues(key registry.Key, record Record) error {
	names, err := key.ReadValueNames(10)
	if err != nil && err != io.EOF {
		return err
	}
	expected := make(map[string]registrationString)
	for _, field := range windowsRegistrationStrings(record) {
		expected[strings.ToLower(field.name)] = field
	}
	for _, name := range names {
		switch strings.ToLower(name) {
		case strings.ToLower(registrationIdentityValue):
			continue
		case "nomodify", "norepair":
			value, kind, err := key.GetIntegerValue(name)
			if err != nil || kind != registry.DWORD || value != 1 {
				return errors.New("uninstall registration flags were changed")
			}
		default:
			field, ok := expected[strings.ToLower(name)]
			if !ok {
				return errors.New("uninstall registration contains unowned values")
			}
			value, kind, err := key.GetStringValue(name)
			matches := value == field.text || (field.path && strings.EqualFold(value, field.text))
			if err != nil || kind != registry.SZ || !matches {
				return errors.New("uninstall registration metadata was changed")
			}
		}
	}
	return nil
}
