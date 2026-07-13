package main

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDownloadLogsAndDetectKnownWarning(t *testing.T) {
	archive := logArchive(t, "2026-07-13T00:00:00Z [DEP0005] DeprecationWarning: Buffer() is deprecated due to security and usability issues\n")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/repos/0disoft/velox/actions/runs/42/logs" {
			http.NotFound(response, request)
			return
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("missing bearer token")
		}
		response.Header().Set("Content-Type", "application/zip")
		_, _ = response.Write(archive)
	}))
	defer server.Close()

	m := monitor{client: server.Client(), baseURL: server.URL, token: "secret"}
	body, err := m.downloadLogs(context.Background(), "0disoft/velox", 42)
	if err != nil {
		t.Fatal(err)
	}
	result, err := inspectLogs("0disoft/velox", 42, body, time.Date(2026, 7, 13, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "present" || len(result.Findings) != 1 || result.Findings[0].OccurrenceCount != 1 {
		t.Fatalf("known warning was not detected: %#v", result)
	}
}

func TestInspectLogsReportsAbsentWithoutKnownWarning(t *testing.T) {
	result, err := inspectLogs("0disoft/velox", 42, logArchive(t, "ordinary action output\n"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "absent" || len(result.Findings) != 0 {
		t.Fatalf("unexpected finding: %#v", result)
	}
}

func TestInspectLogsRejectsInvalidArchive(t *testing.T) {
	if _, err := inspectLogs("0disoft/velox", 42, []byte("not a zip"), time.Now()); err == nil {
		t.Fatal("expected invalid archive to fail")
	}
}

func TestReadBoundedRejectsTruncatedEvidence(t *testing.T) {
	if _, err := readBounded(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("expected oversized evidence to fail")
	}
}

func logArchive(t *testing.T, content string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	file, err := archive.Create("Consumer sample 0.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
