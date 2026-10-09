package builder

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/velox/internal/assettree"
	"github.com/0disoft/velox/internal/buildplan"
	"github.com/0disoft/velox/internal/manifest"
)

func TestBuildPreservesAppNamedLikeLegacyBackup(t *testing.T) {
	for _, suffix := range []string{".previous", ".zip.previous"} {
		t.Run(suffix, func(t *testing.T) {
			rootA, configA, hostA := fixture(t)
			_, configB, hostB := fixture(t)
			value, err := manifest.Load(configB)
			if err != nil {
				t.Fatal(err)
			}
			value.App.ID += suffix
			body, err := json.Marshal(value.Manifest)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, configB, body)
			out := filepath.Join(rootA, "dist")
			build := func(config, host string) Result {
				t.Helper()
				plan, err := buildplan.CreateBuild(buildplan.Options{ManifestPath: config, HostPath: host, OutputRoot: out})
				if err != nil {
					t.Fatal(err)
				}
				result, err := Build(plan)
				if err != nil {
					t.Fatal(err)
				}
				return result
			}
			build(configA, hostA)
			other := build(configB, hostB)
			before, err := assettree.Scan(other.DirectoryPath)
			if err != nil {
				t.Fatal(err)
			}
			archiveBefore, err := os.ReadFile(other.ArchivePath)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, filepath.Join(rootA, "web", "app.js"), []byte("console.log('updated A')\n"))
			build(configA, hostA)
			after, err := assettree.Scan(other.DirectoryPath)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("rebuilding A changed B's directory: %v", err)
			}
			archiveAfter, err := os.ReadFile(other.ArchivePath)
			if err != nil || !bytes.Equal(archiveBefore, archiveAfter) || sha256.Sum256(archiveAfter) != sha256.Sum256(archiveBefore) {
				t.Fatalf("rebuilding A changed B's ZIP: %v", err)
			}
			build(configB, hostB)
		})
	}
}
