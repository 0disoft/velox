package archive

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateFilesIsDeterministicAndRootless(t *testing.T) {
	root := t.TempDir()
	first := writeInput(t, root, "first.bin", "first")
	second := writeInput(t, root, "second.bin", "second")
	one := filepath.Join(root, "one.zip")
	two := filepath.Join(root, "two.zip")

	inputs := []Input{{Source: second, Name: "velox-host.exe"}, {Source: first, Name: "velox.exe"}}
	firstResult, err := CreateFiles(one, inputs)
	if err != nil {
		t.Fatal(err)
	}
	secondResult, err := CreateFiles(two, []Input{inputs[1], inputs[0]})
	if err != nil {
		t.Fatal(err)
	}
	if firstResult != secondResult {
		t.Fatalf("archive results differ: %#v != %#v", firstResult, secondResult)
	}
	oneData, err := os.ReadFile(one)
	if err != nil {
		t.Fatal(err)
	}
	twoData, err := os.ReadFile(two)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(oneData, twoData) {
		t.Fatal("equivalent file lists produced different ZIP bytes")
	}

	reader, err := zip.OpenReader(one)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 2 || reader.File[0].Name != "velox-host.exe" || reader.File[1].Name != "velox.exe" {
		t.Fatalf("archive entries = %#v", entryNames(reader.File))
	}
	for _, entry := range reader.File {
		if entry.Modified.UTC() != normalizedTime || entry.Mode().Perm() != 0o644 || entry.Method != zip.Deflate {
			t.Fatalf("entry metadata for %s = time %s, mode %o, method %d", entry.Name, entry.Modified.UTC().Format(time.RFC3339), entry.Mode().Perm(), entry.Method)
		}
	}
}

func TestCreateFilesRejectsUnsafeAndDuplicateEntries(t *testing.T) {
	root := t.TempDir()
	input := writeInput(t, root, "input.bin", "input")
	tests := []struct {
		name   string
		inputs []Input
	}{
		{name: "empty", inputs: nil},
		{name: "parent traversal", inputs: []Input{{Source: input, Name: "../input.bin"}}},
		{name: "backslash", inputs: []Input{{Source: input, Name: `dir\\input.bin`}}},
		{name: "drive", inputs: []Input{{Source: input, Name: "C:/input.bin"}}},
		{name: "duplicate", inputs: []Input{{Source: input, Name: "input.bin"}, {Source: input, Name: "input.bin"}}},
		{name: "case collision", inputs: []Input{{Source: input, Name: "INPUT.bin"}, {Source: input, Name: "input.bin"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := filepath.Join(root, test.name+".zip")
			if _, err := CreateFiles(output, test.inputs); err == nil {
				t.Fatal("CreateFiles accepted invalid inputs")
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("invalid archive output remains: %v", err)
			}
		})
	}
}

func TestCreateFilesRefusesExistingOutputAndRemovesPartialOutput(t *testing.T) {
	root := t.TempDir()
	input := writeInput(t, root, "input.bin", "input")
	output := filepath.Join(root, "archive.zip")
	if err := os.WriteFile(output, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFiles(output, []Input{{Source: input, Name: "input.bin"}}); err == nil {
		t.Fatal("CreateFiles overwrote existing output")
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("existing output = %q", data)
	}

	partial := filepath.Join(root, "partial.zip")
	missing := filepath.Join(root, "missing.bin")
	if _, err := CreateFiles(partial, []Input{{Source: missing, Name: "missing.bin"}}); err == nil {
		t.Fatal("CreateFiles accepted missing input")
	}
	if _, err := os.Stat(partial); !os.IsNotExist(err) {
		t.Fatalf("partial archive remains: %v", err)
	}
}

func writeInput(t *testing.T, root, name, content string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func entryNames(files []*zip.File) []string {
	names := make([]string, len(files))
	for index, file := range files {
		names[index] = file.Name
	}
	return names
}
