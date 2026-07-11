package assettree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRelativePathRejectsWindowsHazards(t *testing.T) {
	invalid := []string{
		"CON", "con.txt", "nested/PRN.json", "COM1", "LPT9.log",
		"name:stream", "trailing.", "trailing ",
	}
	for _, path := range invalid {
		t.Run(path, func(t *testing.T) {
			if err := validateRelativePath(path); err == nil {
				t.Fatalf("validateRelativePath(%q) succeeded", path)
			}
		})
	}
}

func TestScanIsSortedAndContentAddressed(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, filepath.Join(root, "z.txt"), "z")
	writeAsset(t, filepath.Join(root, "nested", "a.txt"), "a")

	first, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Files) != 2 || first.Files[0].RelativePath != "nested/a.txt" || first.Files[1].RelativePath != "z.txt" {
		t.Fatalf("unexpected order: %+v", first.Files)
	}
	if first.Digest != second.Digest || first.TotalBytes != 2 {
		t.Fatalf("unstable tree: first=%+v second=%+v", first, second)
	}
}

func TestScanRejectsSymbolicLinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "outside.txt")
	writeAsset(t, target, "outside")
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	_, err := Scan(root)
	if err == nil || !strings.Contains(err.Error(), "link or reparse point") {
		t.Fatalf("Scan() error = %v", err)
	}
}

func writeAsset(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
