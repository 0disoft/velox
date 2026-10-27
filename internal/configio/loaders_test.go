package configio_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/artifactlimits"
	"github.com/0disoft/velox/internal/hostmeta"
	"github.com/0disoft/velox/internal/manifest"
	"github.com/0disoft/velox/internal/runtimeconfig"
)

func TestConfigurationLoadSizeBoundaries(t *testing.T) {
	for _, loader := range []struct {
		name string
		body string
		load func(string) error
	}{
		{"manifest", `{"schemaVersion":1,"app":{"id":"dev.test.limits","name":"Limits","version":"1"}}`, func(path string) error { _, err := manifest.Load(path); return err }},
		{"runtime", `{"runtimeVersion":1,"app":{"id":"dev.test.limits","name":"Limits","version":"1"},"assets":{"root":"web","entry":"index.html"},"window":{"width":640,"height":480},"security":{"permissions":[]}}`, func(path string) error { _, err := runtimeconfig.Load(path); return err }},
		{"host", `{"schemaVersion":"velox.host/v1","releaseVersion":"test","target":"windows-x64","contracts":{"host":1,"runtime":1,"ipc":1},"host":{"file":"host.exe","bytes":1,"sha256":"` + strings.Repeat("a", 64) + `"}}`, func(path string) error { _, err := hostmeta.Load(path); return err }},
	} {
		t.Run(loader.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "web"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "web", "index.html"), []byte("ok"), 0o644); err != nil {
				t.Fatal(err)
			}
			for _, size := range []int{artifactlimits.MaxMetadataBytes - 1, artifactlimits.MaxMetadataBytes, artifactlimits.MaxMetadataBytes + 1} {
				t.Run(strconv.Itoa(size), func(t *testing.T) {
					path := filepath.Join(root, "config.json")
					body := loader.body + strings.Repeat(" ", size-len(loader.body))
					if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
						t.Fatal(err)
					}
					err := loader.load(path)
					if size <= artifactlimits.MaxMetadataBytes {
						if err != nil {
							t.Fatalf("valid %d-byte configuration rejected: %v", size, err)
						}
					} else if err == nil || !strings.Contains(err.Error(), "byte limit") {
						t.Fatalf("oversized configuration: got %v, want size rejection", err)
					}
				})
			}
		})
	}
}
