//go:build windows

package pebranding

import (
	"bytes"
	"context"
	"debug/pe"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestApplyPreservesCodeAndIsDeterministic(t *testing.T) {
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	icon, err := os.ReadFile("../../assets/branding/velox.ico")
	if err != nil {
		t.Fatal(err)
	}
	options := Options{Enabled: true, Name: "Example 한글", Version: "1.2.3", Company: "Rodisoft", Icon: icon}
	var first []byte
	for i := 0; i < 2; i++ {
		path := filepath.Join(t.TempDir(), "app.exe")
		if err := os.WriteFile(path, original, 0600); err != nil {
			t.Fatal(err)
		}
		if err := Apply(path, "app.exe", options); err != nil {
			t.Fatal(err)
		}
		result, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = result
		} else if !bytes.Equal(first, result) {
			t.Fatal("resource edits are not deterministic")
		}
		before, _ := pe.NewFile(bytes.NewReader(original))
		after, err := pe.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		left, _ := before.Section(".text").Data()
		right, _ := after.Section(".text").Data()
		before.Close()
		after.Close()
		if !bytes.Equal(left, right) {
			t.Fatal("branding changed executable code")
		}
		module, err := windows.LoadLibraryEx(path, 0, windows.LOAD_LIBRARY_AS_DATAFILE)
		if err != nil {
			t.Fatal(err)
		}
		find := kernel.NewProc("FindResourceW")
		for _, item := range [][2]uintptr{{1, 16}, {1, 14}, {11, 14}, {101, 3}} {
			handle, _, _ := find.Call(uintptr(module), item[0], item[1])
			if handle == 0 {
				t.Errorf("missing resource %v", item)
			}
		}
		versionHandle, _, _ := find.Call(uintptr(module), 1, 16)
		version, err := windows.LoadResourceData(module, windows.Handle(versionHandle))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(version, versionResource(options, "app.exe")) {
			t.Error("version resource differs from expected metadata")
		}
		windows.FreeLibrary(module)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		output, err := exec.CommandContext(ctx, path, "-test.run=^$").CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("branded executable cannot start: %v: %.1024s", err, output)
		}
	}
	unchanged, _ := os.ReadFile(source)
	if !bytes.Equal(original, unchanged) {
		t.Fatal("template modified")
	}
}

func TestApplyRejectsSignedTemplateWithoutWriting(t *testing.T) {
	source, _ := os.Executable()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	peOffset := int(binary.LittleEndian.Uint32(data[60:]))
	security := peOffset + 24 + 112 + 4*8
	binary.LittleEndian.PutUint32(data[security:], uint32(len(data)))
	binary.LittleEndian.PutUint32(data[security+4:], 8)
	data = append(data, make([]byte, 8)...)
	path := filepath.Join(t.TempDir(), "signed.exe")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Apply(path, "app.exe", Options{Enabled: true, Name: "App", Version: "1.0"}); err == nil {
		t.Fatal("accepted signed template")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(data, after) {
		t.Fatal("rejected template was modified")
	}
}
