package signingrecord

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPrepareSigningInputCreatesDeterministicExactFileSet(t *testing.T) {
	root := t.TempDir()
	unsigned := filepath.Join(root, "unsigned")
	writeTestFile(t, filepath.Join(unsigned, "velox.exe"), "unsigned cli")
	writeTestFile(t, filepath.Join(unsigned, "velox-host.exe"), "unsigned host")
	writeTestFile(t, filepath.Join(unsigned, "ignored.txt"), "must not be packaged")
	one := filepath.Join(root, "one", SigningInputName)
	two := filepath.Join(root, "two", SigningInputName)

	first, err := PrepareSigningInput(unsigned, one)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PrepareSigningInput(unsigned, two)
	if err != nil {
		t.Fatal(err)
	}
	if first.Artifact != second.Artifact || first.Artifact.File != SigningInputName {
		t.Fatalf("signing input results differ: %#v != %#v", first, second)
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
		t.Fatal("equivalent unsigned inputs produced different signing ZIP bytes")
	}

	reader, err := zip.OpenReader(one)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 2 || reader.File[0].Name != "velox-host.exe" || reader.File[1].Name != "velox.exe" {
		t.Fatalf("signing input entries = %#v", signingInputEntryNames(reader.File))
	}
	for _, entry := range reader.File {
		if entry.Modified.UTC() != time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC) || entry.Mode().Perm() != 0o644 || entry.Method != zip.Deflate {
			t.Fatalf("unexpected ZIP metadata for %s", entry.Name)
		}
	}
}

func TestPrepareSigningInputRejectsInvalidInputsWithoutOutput(t *testing.T) {
	root := t.TempDir()
	unsigned := filepath.Join(root, "unsigned")
	writeTestFile(t, filepath.Join(unsigned, "velox.exe"), "unsigned cli")
	out := filepath.Join(root, SigningInputName)
	if _, err := PrepareSigningInput(unsigned, out); err == nil {
		t.Fatal("PrepareSigningInput accepted a missing host")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("failed signing input remains: %v", err)
	}
	if _, err := PrepareSigningInput(unsigned, filepath.Join(root, "wrong.zip")); err == nil {
		t.Fatal("PrepareSigningInput accepted the wrong output name")
	}
}

func TestPrepareSigningInputRefusesExistingOutput(t *testing.T) {
	root := t.TempDir()
	unsigned := filepath.Join(root, "unsigned")
	writeTestFile(t, filepath.Join(unsigned, "velox.exe"), "unsigned cli")
	writeTestFile(t, filepath.Join(unsigned, "velox-host.exe"), "unsigned host")
	out := filepath.Join(root, SigningInputName)
	writeTestFile(t, out, "keep")
	if _, err := PrepareSigningInput(unsigned, out); err == nil {
		t.Fatal("PrepareSigningInput overwrote an existing output")
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("existing output = %q", data)
	}
}

func TestPrepareSigningInputRejectsLinkedExecutable(t *testing.T) {
	root := t.TempDir()
	unsigned := filepath.Join(root, "unsigned")
	realCLI := filepath.Join(root, "real-velox.exe")
	writeTestFile(t, realCLI, "unsigned cli")
	if err := os.MkdirAll(unsigned, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realCLI, filepath.Join(unsigned, "velox.exe")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	writeTestFile(t, filepath.Join(unsigned, "velox-host.exe"), "unsigned host")
	out := filepath.Join(root, SigningInputName)
	if _, err := PrepareSigningInput(unsigned, out); err == nil {
		t.Fatal("PrepareSigningInput accepted a linked executable")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("failed signing input remains: %v", err)
	}
}

func signingInputEntryNames(files []*zip.File) []string {
	names := make([]string, len(files))
	for index, file := range files {
		names[index] = file.Name
	}
	return names
}
