package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCollectBuildsOrderedEnvironmentGroupedHistory(t *testing.T) {
	oldSummary := summaryFixture("100", "runner-a", "1", "120.1", 500, 450, 50)
	archive := zipSummary(t, oldSummary)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("missing bearer token")
		}
		switch request.URL.Path {
		case "/repos/0disoft/velox/actions/runs/200":
			writeFixtureJSON(t, response, workflowRun{ID: 200, RunAttempt: 2, HeadSHA: "current", CreatedAt: "2026-07-13T03:23:00Z"})
		case "/repos/0disoft/velox/actions/workflows/consumer-evidence.yml/runs":
			writeFixtureJSON(t, response, workflowRunsResponse{WorkflowRuns: []workflowRun{
				{ID: 200, RunAttempt: 2, HeadSHA: "current", CreatedAt: "2026-07-13T03:23:00Z"},
				{ID: 150, RunAttempt: 1, HeadSHA: "missing", CreatedAt: "2026-07-06T03:23:00Z"},
				{ID: 100, RunAttempt: 1, HeadSHA: "old", CreatedAt: "2026-06-29T03:23:00Z"},
			}})
		case "/repos/0disoft/velox/actions/runs/150/artifacts":
			writeFixtureJSON(t, response, artifactsResponse{})
		case "/repos/0disoft/velox/actions/runs/100/artifacts":
			writeFixtureJSON(t, response, artifactsResponse{Artifacts: []artifact{{Name: "startup-lifecycle-100-1", ArchiveDownloadURL: serverURL(request) + "/artifact/100"}}})
		case "/artifact/100":
			response.Header().Set("Content-Type", "application/zip")
			_, _ = response.Write(archive)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	current := summaryFixture("200", "runner-a", "1", "120.1", 600, 550, 50)
	c := collector{client: server.Client(), baseURL: server.URL, token: "secret"}
	result, err := c.collect(context.Background(), "0disoft/velox", "consumer-evidence.yml", current, 12, time.Date(2026, 7, 13, 4, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceCount != 2 || len(result.Series) != 2 {
		t.Fatalf("unexpected history size: %#v", result)
	}
	if result.Series[0].RunID != 100 || result.Series[1].RunID != 200 {
		t.Fatalf("history is not chronological: %#v", result.Series)
	}
	if len(result.CollectionIssues) != 1 || result.CollectionIssues[0].Code != "ARTIFACT_UNAVAILABLE" {
		t.Fatalf("missing artifact issue was not preserved: %#v", result.CollectionIssues)
	}
	if len(result.EnvironmentGroups) != 1 || result.EnvironmentGroups[0].SampleCount != 2 {
		t.Fatalf("environment grouping is wrong: %#v", result.EnvironmentGroups)
	}
}

func TestCollectRejectsCurrentSummaryWithoutRunID(t *testing.T) {
	current := summaryFixture("", "runner-a", "1", "120.1", 600, 550, 50)
	c := collector{client: http.DefaultClient, baseURL: "https://example.invalid", token: "secret"}
	if _, err := c.collect(context.Background(), "0disoft/velox", "consumer-evidence.yml", current, 12, time.Now()); err == nil {
		t.Fatal("expected missing run ID to fail")
	}
}

func TestDecodeSummaryRejectsTrailingJSON(t *testing.T) {
	body, err := json.Marshal(summaryFixture("200", "runner-a", "1", "120.1", 600, 550, 50))
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, []byte("{}")...)
	if _, err := decodeSummary(body); err == nil {
		t.Fatal("expected trailing JSON to fail")
	}
}

func TestReadBoundedRejectsTruncatedEvidence(t *testing.T) {
	if _, err := readBounded(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("expected oversized evidence to fail")
	}
}

func summaryFixture(runID, image, imageVersion, webView string, ready, browserExit, afterExit float64) lifecycleSummary {
	return lifecycleSummary{
		SchemaVersion: "velox.startup-lifecycle-summary/v1", Outcome: "success",
		Environment: environment{RunnerImage: &image, RunnerImageVersion: &imageVersion, WebView2Version: webView, GitHubRunID: &runID},
		Metrics: map[string]metricStats{
			"immediateReadyMs":                      {P50Ms: ready, P95Ms: ready + 10},
			"firstBrowserExitAfterImmediateStartMs": {P50Ms: browserExit, P95Ms: browserExit + 10},
			"immediateReadyAfterFirstBrowserExitMs": {P50Ms: afterExit, P95Ms: afterExit + 10},
		},
		Ordering: ordering{ReadyWaitedForFirstBrowserExitCount: 10},
	}
}

func zipSummary(t *testing.T, summary lifecycleSummary) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	file, err := archive.Create("startup-lifecycle-summary.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(file).Encode(summary); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func writeFixtureJSON(t *testing.T, response http.ResponseWriter, value any) {
	t.Helper()
	response.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(response).Encode(value); err != nil {
		t.Fatal(err)
	}
}

func serverURL(request *http.Request) string {
	return "http://" + request.Host
}
