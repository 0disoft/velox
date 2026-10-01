//go:build windows

package builder

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/buildplan"
	"github.com/0disoft/velox/internal/inspector"
)

func TestBrandedBuildReportsFinalHostAndInspects(t *testing.T) {
	root, config, host := fixture(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, host, original)
	writeFixture(t, filepath.Join(filepath.Dir(host), "velox-host.json"), hostMetadata(original))
	icon, err := os.ReadFile("../../assets/branding/velox.ico")
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(root, "app.ico"), icon)
	data, _ := os.ReadFile(config)
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	value["branding"] = map[string]any{"icon": "app.ico", "company": "Rodisoft"}
	data, _ = json.Marshal(value)
	writeFixture(t, config, data)
	plan, err := buildplan.CreateBuild(buildplan.Options{ManifestPath: config, HostPath: host})
	if err != nil {
		t.Fatal(err)
	}
	first, err := Build(plan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(plan)
	if err != nil {
		t.Fatal(err)
	}
	if first.ArchiveSHA256 != second.ArchiveSHA256 || first.Report.Host.SHA256 == plan.Snapshot().HostSHA256 {
		t.Fatal("branded digest contract violated")
	}
	for _, path := range []string{first.ArchivePath, first.DirectoryPath} {
		if _, err := inspector.Inspect(path); err != nil {
			t.Fatal(err)
		}
	}
	unchanged, _ := os.ReadFile(host)
	if !bytes.Equal(unchanged, original) {
		t.Fatal("source template changed")
	}
}
