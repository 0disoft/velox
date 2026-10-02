package hygiene_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/runtimeconfig"
)

func TestExternalLinksFixtureKeepsPermissionAndManualBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	launcher, err := os.ReadFile(filepath.Join(root, "scripts", "external-links-manual.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(launcher), "windowsHide: false,") {
		t.Fatal("manual fixture window must remain visible")
	}
	fixture := filepath.Join(root, "tests", "fixtures", "external-links")
	for _, mode := range []string{"allowed", "denied"} {
		cfg, err := runtimeconfig.Load(filepath.Join(fixture, mode+".runtime.json"))
		if err != nil {
			t.Fatal(err)
		}
		want := ""
		if mode == "allowed" {
			want = "external.open"
		}
		if strings.Join(cfg.Security.Permissions, ",") != want {
			t.Fatalf("%s permissions=%v", mode, cfg.Security.Permissions)
		}
	}
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(fixture, "web", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	html, app := read("index.html"), read("app.js")
	for _, value := range []string{"https://github.com/0disoft/velox", "http://example.com/", "file:///C:/velox-external-check.txt", "connect-src 'none'", "data-field=", "aria-label="} {
		if !strings.Contains(html, value) {
			t.Errorf("missing fixture boundary %q", value)
		}
	}
	if strings.Count(app, `bridge.invoke("external.open"`) != 1 || !strings.Contains(app, `addEventListener("click"`) || !strings.Contains(app, "Queued for confirmation") {
		t.Fatal("fixture must invoke only on a click and report queued, not launched")
	}
	for _, forbidden := range []string{"window.open(", "fetch(", ".innerHTML", "setInterval(", "setTimeout("} {
		if strings.Contains(app, forbidden) {
			t.Errorf("unexpected automatic or untrusted side effect %q", forbidden)
		}
	}
}
