package outputpair

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPromoteRecoversAfterRollbackCleanupFailure(t *testing.T) {
	for _, failCleanup := range []bool{false, true} {
		name := "successful cleanup"
		if failCleanup {
			name = "failed cleanup"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			directory, archive := filepath.Join(root, "app"), filepath.Join(root, "app.zip")
			stageDirectory, stageArchive := filepath.Join(root, "stage"), filepath.Join(root, "stage.zip")
			write(t, filepath.Join(directory, "old.txt"))
			write(t, archive)
			write(t, filepath.Join(stageDirectory, "new.txt"))
			// Omit stageArchive to fail the second promotion after the directory moved.
			cleanupFailure := errors.New("injected directory cleanup failure")
			cleanupCalls := 0
			removeDirectory := func(path string) error {
				cleanupCalls++
				if path != directory {
					t.Fatalf("unexpected cleanup path %q", path)
				}
				if failCleanup {
					return cleanupFailure
				}
				return os.RemoveAll(path)
			}
			err := promote(directory, archive, stageDirectory, stageArchive, removeDirectory)
			if err == nil || !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing archive promotion error: %v", err)
			}
			if cleanupCalls != 1 {
				t.Fatalf("cleanup calls = %d, want 1", cleanupCalls)
			}
			if failCleanup {
				if !errors.Is(err, cleanupFailure) {
					t.Fatalf("missing rollback error: %v", err)
				}
				assertExists(t, filepath.Join(directory+".previous", "old.txt"))
				assertExists(t, archive+".previous")
				assertExists(t, filepath.Join(directory, "new.txt"))
				assertMissing(t, archive)
			}
			// A later invocation must recover the original pair, not accept a mixed pair.
			if err := Recover(directory, archive); err != nil {
				t.Fatal(err)
			}
			assertExists(t, filepath.Join(directory, "old.txt"))
			assertMissing(t, filepath.Join(directory, "new.txt"))
			data, err := os.ReadFile(archive)
			if err != nil || string(data) != "fixture" {
				t.Fatalf("previous archive changed: %q, %v", data, err)
			}
			assertMissing(t, directory+".previous")
			assertMissing(t, archive+".previous")
		})
	}
}
