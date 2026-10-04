package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const validObservation = `{
  "schemaVersion":"velox.consumer-e2e/v1", "scope":"checkout-complete-to-portable-zip",
  "evidenceLevel":"local-contract-smoke", "sampleId":"sample-1", "outcome":"success",
  "startedAtUtc":"2026-10-05T00:00:00Z", "finishedAtUtc":"2026-10-05T00:00:01Z", "durationMs":123.4567,
  "acquisition":{"mode":"local-file","archiveBytes":10,"archiveSha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","expectedSha256Verified":true},
  "release":{"version":"0.5.10-alpha.63","target":"windows-x64"},
  "fixture":{"kind":"dependency-free-init-template","appId":"dev.velox.test","assetFiles":1,"assetBytes":1,"assetSha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "build":{"durationMs":12,"archiveBytes":10,"archiveSha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","portableFiles":1,"portableBytes":1,"survivingIntermediateFiles":0,
    "processTrace":{"status":"pass","rootProcessCount":1,"descendants":[],"forbiddenDescendants":[]}},
  "environment":{"os":"Windows","architecture":"X64","runnerImage":"windows-2025","runnerImageVersion":"1","githubRunId":"1","githubRunAttempt":"1","gitCommit":"dddddddddddddddddddddddddddddddddddddddd"},
  "measurement":{"tool":"scripts/measure-consumer-e2e.ps1","toolVersion":1,"unit":"milliseconds","endToEndClock":"System.DateTime.UtcNow-across-workflow-steps","buildClock":"System.Diagnostics.Stopwatch","startsAt":"after-checkout-before-release-artifact-acquisition","endsAt":"after-portable-zip-inspection","warmupSamples":0,"concurrency":1,"osFileCacheState":"uncontrolled-local-state"},
  "error":null
}`

func invoke(t *testing.T, bodies []string, expected int) (summary, error, bool) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "raw", "nested")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, body := range bodies {
		if err := os.WriteFile(filepath.Join(root, strconv.Itoa(i)+".json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(dir, "out", "summary.json")
	var stdout bytes.Buffer
	err := run([]string{"--results-root", filepath.Dir(root), "--output", output,
		"--expected-samples", strconv.Itoa(expected), "--raw-schema", "../../schema/consumer-e2e-v1.schema.json",
		"--summary-schema", "../../schema/consumer-e2e-summary-v1.schema.json"}, &stdout)
	body, readErr := os.ReadFile(output)
	if os.IsNotExist(readErr) {
		return summary{}, err, false
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(body, stdout.Bytes()) {
		t.Fatal("stdout does not match saved summary")
	}
	var result summary
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	return result, err, true
}

func hosted(body string) string {
	body = strings.ReplaceAll(body, "local-contract-smoke", "hosted-runner-evidence")
	body = strings.ReplaceAll(body, "local-file", "github-actions-artifact")
	return strings.ReplaceAll(body, "uncontrolled-local-state", "fresh-hosted-runner-when-executed-in-an-isolated-job")
}

func TestSummarizeSuccessfulSamples(t *testing.T) {
	second := strings.ReplaceAll(validObservation, "sample-1", "sample-2")
	second = strings.ReplaceAll(second, "123.4567", "987.6543")
	result, err, written := invoke(t, []string{second, validObservation}, 2)
	if err != nil || !written {
		t.Fatalf("run: written=%v err=%v", written, err)
	}
	if result.SuccessCount != 2 || result.ProcessEvidencePassCount != 2 || result.EvidenceLevel != "local-contract-summary" || result.MissingCount != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if result.Samples[0].SampleID != "sample-1" || result.ReleaseArchiveSHA256 == nil {
		t.Fatalf("unexpected sample ordering or digest: %+v", result)
	}
	want := statistics{123.457, 123.457, 987.654, 987.654}
	if result.Statistics == nil || *result.Statistics != want {
		t.Fatalf("statistics: got %+v want %+v", result.Statistics, want)
	}
}

func TestIncompleteEvidenceStillWritesSummary(t *testing.T) {
	var failure map[string]any
	if err := json.Unmarshal([]byte(validObservation), &failure); err != nil {
		t.Fatal(err)
	}
	failure["sampleId"], failure["outcome"] = "failed-1", "failure"
	for _, key := range []string{"acquisition", "release", "fixture", "build"} {
		failure[key] = nil
	}
	failure["error"] = map[string]any{"phase": "build", "code": "PHASE_FAILED"}
	body, _ := json.Marshal(failure)
	tests := []struct {
		name     string
		bodies   []string
		expected int
	}{
		{"missing", []string{validObservation}, 2},
		{"excess", []string{validObservation, strings.ReplaceAll(validObservation, "sample-1", "sample-2")}, 1},
		{"empty", nil, 1},
		{"failure", []string{string(body)}, 1},
		{"different-release", []string{validObservation, strings.ReplaceAll(strings.ReplaceAll(validObservation, "sample-1", "sample-2"), strings.Repeat("a", 64), strings.Repeat("e", 64))}, 2},
		{"hosted-unverified", []string{strings.ReplaceAll(hosted(validObservation), `"status":"pass"`, `"status":"unverified"`)}, 1},
		{"hosted-fail", []string{strings.ReplaceAll(hosted(validObservation), `"status":"pass"`, `"status":"fail"`)}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err, written := invoke(t, tc.bodies, tc.expected)
			if err == nil || !written {
				t.Fatalf("expected failure with saved summary: written=%v err=%v", written, err)
			}
			if tc.name == "failure" && (result.FailureCount != 1 || result.Statistics != nil || result.Samples[0].FailurePhase == nil || *result.Samples[0].FailurePhase != "build") {
				t.Fatalf("failure details lost: %+v", result)
			}
		})
	}
}

func TestInvalidEvidenceDoesNotWriteSummary(t *testing.T) {
	cases := map[string][]string{
		"duplicate":               {validObservation, validObservation},
		"duplicate-case":          {validObservation, strings.ReplaceAll(validObservation, "sample-1", "SAMPLE-1")},
		"unknown-field":           {strings.Replace(validObservation, `"sampleId":`, `"extra":true,"sampleId":`, 1)},
		"missing-field":           {strings.Replace(validObservation, `"sampleId":"sample-1",`, "", 1)},
		"trailing-value":          {validObservation + "{}"},
		"negative-duration":       {strings.ReplaceAll(validObservation, "123.4567", "-1")},
		"bad-date":                {strings.ReplaceAll(validObservation, "2026-10-05T00:00:00Z", "yesterday")},
		"missing-success-build":   {strings.ReplaceAll(validObservation, `"outcome":"success"`, `"outcome":"failure"`)},
		"hosted-with-local-clock": {strings.ReplaceAll(validObservation, "local-contract-smoke", "hosted-runner-evidence")},
	}
	for name, bodies := range cases {
		t.Run(name, func(t *testing.T) {
			_, err, written := invoke(t, bodies, 1)
			if err == nil || written {
				t.Fatalf("expected invalid input without output: written=%v err=%v", written, err)
			}
		})
	}
}

func TestHostedAndLocalProcessPolicy(t *testing.T) {
	result, err, _ := invoke(t, []string{hosted(validObservation)}, 1)
	if err != nil || result.EvidenceLevel != "hosted-runner-summary" {
		t.Fatalf("hosted: %+v %v", result, err)
	}
	result, err, _ = invoke(t, []string{strings.ReplaceAll(validObservation, `"status":"pass"`, `"status":"unverified"`)}, 1)
	if err != nil || result.ProcessEvidenceUnverifiedCount != 1 {
		t.Fatalf("local unverified evidence must remain diagnostic: %+v %v", result, err)
	}
}

func TestUsageAndSingleFileInput(t *testing.T) {
	for _, args := range [][]string{nil, {"--expected-samples", "101"}, {"--unknown"}, {"--results-root", "raw", "--output", "summary", "--expected-samples", "1", "extra"}} {
		if err := run(args, io.Discard); err == nil {
			t.Fatalf("accepted invalid usage: %v", args)
		}
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	if err := os.WriteFile(input, []byte(validObservation), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{"--results-root", input, "--output", filepath.Join(dir, "summary.json"), "--expected-samples", "1", "--raw-schema", "../../schema/consumer-e2e-v1.schema.json", "--summary-schema", "../../schema/consumer-e2e-summary-v1.schema.json"}
	if err := run(args, io.Discard); err != nil {
		t.Fatal(err)
	}
	args[3] = input
	if err := run(args, io.Discard); err == nil {
		t.Fatal("output overwrote raw input")
	}
	body, _ := os.ReadFile(input)
	if string(body) != validObservation {
		t.Fatal("raw input changed")
	}
}
