package assettree

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/artifactlimits"
)

func TestScanArtifactPreservesContentSummary(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, filepath.Join(root, "z.txt"), "z")
	writeAsset(t, filepath.Join(root, "nested", "empty.txt"), "")
	want, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ScanArtifact(root)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ScanArtifact() = %+v, %v; want %+v", got, err, want)
	}
}

func TestScanAppliesRemainingArtifactBudget(t *testing.T) {
	root := t.TempDir()
	writeAsset(t, filepath.Join(root, "one.txt"), "x")
	for _, test := range []struct {
		name    string
		budget  artifactlimits.Budget
		allowed bool
	}{
		{"last file", artifactlimits.Budget{Files: artifactlimits.MaxFiles - 1}, true},
		{"too many files", artifactlimits.Budget{Files: artifactlimits.MaxFiles}, false},
		{"last byte", artifactlimits.Budget{Bytes: artifactlimits.MaxTotalBytes - 1}, true},
		{"too many bytes", artifactlimits.Budget{Bytes: artifactlimits.MaxTotalBytes}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := scan(root, true, &test.budget)
			if (err == nil) != test.allowed {
				t.Fatalf("scan() = %v, allowed=%t", err, test.allowed)
			}
		})
	}
}

func TestHashFileRejectsSizeChangesAfterBudgetCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.txt")
	writeAsset(t, path, "data")
	for _, expectedSize := range []int64{3, 5} {
		if _, err := hashFile(path, expectedSize); err == nil || !strings.Contains(err.Error(), "changed size") {
			t.Fatalf("hashFile(expectedSize=%d) = %v, want size change rejection", expectedSize, err)
		}
	}
}
