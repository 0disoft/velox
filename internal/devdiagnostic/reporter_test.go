package devdiagnostic

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reporterFixture(t *testing.T) (*Reporter, *bytes.Buffer) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.js"), []byte("private contents"), 0644); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	r, err := New(root, &output)
	if err != nil {
		t.Fatal(err)
	}
	return r, &output
}

func TestReporterKeepsOnlyKnownSourceMetadata(t *testing.T) {
	r, output := reporterFixture(t)
	for _, raw := range []string{
		`{"kind":"uncaught-error","source":"app.js","line":2,"column":4}`,
		`{"kind":"unhandled-rejection","source":"app.js","line":9,"column":9}`,
		`{"kind":"uncaught-error","source":"https://private.invalid/app.js?token=secret","line":3,"column":4}`,
	} {
		r.Report(json.RawMessage(raw))
	}
	want := "velox-debug: uncaught-error app.js:2:4\n" +
		"velox-debug: unhandled-rejection <unknown>:0:0\n" +
		"velox-debug: uncaught-error <unknown>:0:0\n"
	if output.String() != want {
		t.Fatalf("output=%q want=%q", output.String(), want)
	}
}

func TestReporterRejectsMalformedOrContentBearingRequests(t *testing.T) {
	for _, raw := range []string{
		`null`, `{}`, `[]`, `{"kind":"uncaught-error"}`,
		`{"kind":"uncaught-error","source":"app.js","line":null,"column":0}`,
		`{"kind":"uncaught-error","source":"app.js","line":-1,"column":0}`,
		`{"kind":"uncaught-error","source":"app.js","line":1.5,"column":0}`,
		`{"kind":"uncaught-error","source":"app.js","line":10000001,"column":0}`,
		`{"kind":"uncaught-error","source":"app.js","line":0,"column":10000001}`,
		`{"kind":"custom secret","source":"app.js","line":0,"column":0}`,
		`{"kind":"uncaught-error","source":"app.js","line":0,"column":0,"message":"secret"}`,
		`{"kind":"uncaught-error","source":"app.js","line":0,"column":0,"stack":"secret"}`,
		`{"kind":"uncaught-error","source":"app.js","line":0,"column":0} {}`,
		strings.Repeat(" ", maxRequestBytes+1),
	} {
		t.Run(raw[:min(len(raw), 50)], func(t *testing.T) {
			r, output := reporterFixture(t)
			r.Report(json.RawMessage(raw))
			if output.Len() != 0 {
				t.Fatalf("invalid request logged: %q", output.String())
			}
		})
	}
}

func TestReporterBoundsAttemptsForWholeRun(t *testing.T) {
	r, output := reporterFixture(t)
	valid := json.RawMessage(`{"kind":"uncaught-error","source":"app.js","line":1,"column":2}`)
	for range MaxReports + 3 {
		r.Report(valid)
	}
	if strings.Count(output.String(), "\n") != MaxReports {
		t.Fatal("unbounded output")
	}
	r, output = reporterFixture(t)
	for range MaxReports {
		r.Report(json.RawMessage(`{"message":"not accepted"}`))
	}
	r.Report(valid)
	if output.Len() != 0 {
		t.Fatal("invalid attempts did not consume budget")
	}
}
