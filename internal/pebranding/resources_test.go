package pebranding

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestIconResourcesValidateBoundsAndGroups(t *testing.T) {
	data, err := os.ReadFile("../../assets/branding/velox.ico")
	if err != nil {
		t.Fatal(err)
	}
	resources, err := iconResources(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) < 3 || resources[len(resources)-1].id != 11 || resources[len(resources)-2].id != 1 {
		t.Fatal("missing window icon groups")
	}
	for _, bad := range [][]byte{data[:5], append([]byte{0, 0, 1, 0, 255, 255}, data[6:]...), append([]byte(nil), data...)} {
		if len(bad) == len(data) {
			binary.LittleEndian.PutUint32(bad[18:], uint32(len(data)))
		}
		if _, err := iconResources(bad); err == nil {
			t.Fatal("accepted invalid ICO")
		}
	}
}

func TestVersionResourcesAndInvalidText(t *testing.T) {
	options := Options{Enabled: true, Name: "Velox 한글", Version: "1.2.3-beta.1", Company: "Rodisoft"}
	if err := Validate(options); err != nil {
		t.Fatal(err)
	}
	resource := versionResource(options, "app.exe")
	if int(binary.LittleEndian.Uint16(resource)) != len(resource) || !bytes.Contains(resource, utf16Bytes(options.Name)) || !bytes.Contains(resource, utf16Bytes("ProductVersion")) {
		t.Fatal("invalid version resource")
	}
	for _, version := range []string{"abc", "65536.1", "-1", "1.2.3.4.5"} {
		options.Version = version
		if err := Validate(options); err == nil {
			t.Fatalf("accepted %s", version)
		}
	}
	options.Version = "1.2.3"
	options.Company = "nul\x00text"
	if err := Validate(options); err == nil {
		t.Fatal("accepted embedded NUL")
	}
}
