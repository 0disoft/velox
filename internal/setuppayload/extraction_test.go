package setuppayload

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/0disoft/velox/internal/builder"
	"github.com/0disoft/velox/internal/buildinfo"
	"github.com/0disoft/velox/internal/buildplan"
	"github.com/0disoft/velox/internal/inspector"
)

func TestExtractionReturnsCompletedInspection(t *testing.T) {
	root := t.TempDir()
	write := func(name string, body []byte) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	host := []byte("host")
	write("release/velox-host.exe", host)
	write("release/velox-host.json", []byte(fmt.Sprintf(`{"schemaVersion":"velox.host/v1","releaseVersion":%q,"target":"windows-x64","contracts":{"host":1,"runtime":1,"ipc":1},"host":{"file":"velox-host.exe","bytes":%d,"sha256":"%x"}}`, buildinfo.Version, len(host), sha256.Sum256(host))))
	write("web/index.html", []byte("<title>Extraction</title>"))
	write("velox.json", []byte(`{"schemaVersion":1,"app":{"id":"dev.velox.extraction","name":"Extracted App","version":"2.3.4"}}`))
	plan, err := buildplan.Create(buildplan.Options{ManifestPath: filepath.Join(root, "velox.json"), HostPath: filepath.Join(root, "release", "velox-host.exe"), OutputRoot: filepath.Join(root, "dist")})
	if err != nil {
		t.Fatal(err)
	}
	build, err := builder.Build(plan)
	if err != nil {
		t.Fatal(err)
	}
	want, err := inspector.Inspect(build.DirectoryPath)
	if err != nil {
		t.Fatal(err)
	}
	setup := filepath.Join(root, "setup.exe")
	if _, err := pack(bytes.NewReader([]byte("template")), 8, build.ArchivePath, setup); err != nil {
		t.Fatal(err)
	}
	payload, err := Open(setup)
	if err != nil {
		t.Fatal(err)
	}
	defer payload.Close()
	output := filepath.Join(root, "extracted")
	extracted, err := payload.Extract(output)
	if err != nil {
		t.Fatal(err)
	}
	if extracted.Directory != filepath.Join(output, want.App.ID) || !reflect.DeepEqual(extracted.Inspection, want) {
		t.Fatalf("extraction result = %+v, want inspection %+v", extracted, want)
	}
	if extracted.Inspection.App.Name != "Extracted App" || extracted.Inspection.App.Version != "2.3.4" {
		t.Fatal("Setup confirmation metadata unavailable")
	}
}

func TestExtractionFailureReturnsNoUsableInspection(t *testing.T) {
	payload, err := Open(packedFixture(t, "dev.velox.test/file"))
	if err != nil {
		t.Fatal(err)
	}
	defer payload.Close()
	extracted, err := payload.Extract(filepath.Join(t.TempDir(), "output"))
	if err == nil || !reflect.DeepEqual(extracted, Extraction{}) {
		t.Fatalf("invalid tree returned usable inspection: %+v %v", extracted, err)
	}
}
