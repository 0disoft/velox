package safefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRelativePathRejectsUnsafePortableNames(t *testing.T) {
	for _, value := range []string{"..", "../file", "CON.foo.bar", "LPT1.tar.gz", "web/NUL.txt", "web/file. ", "web/file:stream"} {
		if err := ValidateRelativePath(value); err == nil {
			t.Errorf("ValidateRelativePath(%q) accepted an unsafe path", value)
		}
	}
	if err := ValidateRelativePath("web/content.min.js"); err != nil {
		t.Fatalf("ValidateRelativePath() rejected a safe path: %v", err)
	}
}

func TestRejectLinkedComponentsRejectsExistingLink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, target); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if err := RejectLinkedComponents(filepath.Join(target, "file.txt")); err == nil {
		t.Fatal("RejectLinkedComponents() accepted a linked component")
	}
}
