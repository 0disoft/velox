package outputpair

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecoverInterruptedPromotionStates(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
	}{
		{name: "after directory backup", paths: []string{"app.previous/old.txt", "app.zip"}},
		{name: "after both backups", paths: []string{"app.previous/old.txt", "app.zip.previous"}},
		{name: "after directory promotion", paths: []string{"app/new.txt", "app.previous/old.txt", "app.zip.previous"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for _, path := range test.paths {
				write(t, filepath.Join(root, path))
			}
			directory := filepath.Join(root, "app")
			archive := filepath.Join(root, "app.zip")
			if err := Recover(directory, archive); err != nil {
				t.Fatal(err)
			}
			assertExists(t, filepath.Join(directory, "old.txt"))
			assertExists(t, archive)
			assertMissing(t, directory+".previous")
			assertMissing(t, archive+".previous")
		})
	}
}

func TestRecoverKeepsPublishedPairAndCleansBackups(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"app/new.txt", "app.zip", "app.previous/old.txt", "app.zip.previous"} {
		write(t, filepath.Join(root, path))
	}
	if err := Recover(filepath.Join(root, "app"), filepath.Join(root, "app.zip")); err != nil {
		t.Fatal(err)
	}
	assertExists(t, filepath.Join(root, "app", "new.txt"))
	assertMissing(t, filepath.Join(root, "app.previous"))
	assertMissing(t, filepath.Join(root, "app.zip.previous"))
}

func TestRecoverRejectsIncompletePairWithoutBackup(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "app", "partial.txt"))
	if err := Recover(filepath.Join(root, "app"), filepath.Join(root, "app.zip")); err == nil {
		t.Fatal("Recover() accepted an incomplete pair without recovery data")
	}
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s: %v", path, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent: %v", path, err)
	}
}
