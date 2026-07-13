package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
)

const summarySchemaVersion = "velox.startup-lifecycle-summary/v1"

type evidence struct {
	SchemaVersion string          `json:"schemaVersion"`
	Scope         string          `json:"scope"`
	EvidenceLevel string          `json:"evidenceLevel"`
	Outcome       string          `json:"outcome"`
	Repetitions   int             `json:"repetitions"`
	StartedAtUTC  string          `json:"startedAtUtc"`
	FinishedAtUTC string          `json:"finishedAtUtc"`
	Environment   environment     `json:"environment"`
	Measurement   json.RawMessage `json:"measurement"`
	Samples       []sample        `json:"samples"`
}

type environment struct {
	OS                 string  `json:"os"`
	Architecture       string  `json:"architecture"`
	WebView2Version    string  `json:"webView2Version"`
	RunnerImage        *string `json:"runnerImage"`
	RunnerImageVersion *string `json:"runnerImageVersion"`
	GitHubRunID        *string `json:"githubRunId"`
	GitHubRunAttempt   *string `json:"githubRunAttempt"`
	GitCommit          *string `json:"gitCommit"`
}

type sample struct {
	Index            int       `json:"index"`
	Outcome          string    `json:"outcome"`
	First            *launch   `json:"first"`
	Immediate        *launch   `json:"immediate"`
	ProfileReleaseMs *float64  `json:"profileReleaseMs"`
	Timeline         *timeline `json:"timeline"`
	Error            *runError `json:"error"`
}

type launch struct {
	ReadyMs                float64 `json:"readyMs"`
	HostExitMs             float64 `json:"hostExitMs"`
	BrowserProcessID       uint32  `json:"browserProcessId"`
	BrowserExitAfterHostMs float64 `json:"browserExitAfterHostMs"`
}

type timeline struct {
	ImmediateProcessStartAfterFirstHostExitMs float64 `json:"immediateProcessStartAfterFirstHostExitMs"`
	FirstBrowserExitAfterImmediateStartMs     float64 `json:"firstBrowserExitAfterImmediateStartMs"`
	ImmediateReadyAfterFirstBrowserExitMs     float64 `json:"immediateReadyAfterFirstBrowserExitMs"`
	ImmediateReadyWaitedForFirstBrowserExit   bool    `json:"immediateReadyWaitedForFirstBrowserExit"`
}

type runError struct {
	Phase string `json:"phase"`
	Code  string `json:"code"`
}

type summary struct {
	SchemaVersion       string                 `json:"schemaVersion"`
	SourceSchemaVersion string                 `json:"sourceSchemaVersion"`
	SourceSHA256        string                 `json:"sourceSha256"`
	EvidenceLevel       string                 `json:"evidenceLevel"`
	Outcome             string                 `json:"outcome"`
	Repetitions         int                    `json:"repetitions"`
	ObservedSamples     int                    `json:"observedSamples"`
	SuccessCount        int                    `json:"successCount"`
	FailureCount        int                    `json:"failureCount"`
	Environment         environment            `json:"environment"`
	Metrics             map[string]metricStats `json:"metrics"`
	Correlation         correlation            `json:"correlation"`
	Ordering            ordering               `json:"ordering"`
	FailedSamples       []failedSample         `json:"failedSamples"`
}

type metricStats struct {
	MinimumMs float64 `json:"minimumMs"`
	P50Ms     float64 `json:"p50Ms"`
	P95Ms     float64 `json:"p95Ms"`
	MaximumMs float64 `json:"maximumMs"`
}

type correlation struct {
	MetricX            string   `json:"metricX"`
	MetricY            string   `json:"metricY"`
	SampleCount        int      `json:"sampleCount"`
	PearsonCoefficient *float64 `json:"pearsonCoefficient"`
}

type ordering struct {
	ReadyWaitedForFirstBrowserExitCount int   `json:"readyWaitedForFirstBrowserExitCount"`
	ReadyBeforeFirstBrowserExitCount    int   `json:"readyBeforeFirstBrowserExitCount"`
	ViolationSampleIndexes              []int `json:"violationSampleIndexes"`
}

type failedSample struct {
	Index int    `json:"index"`
	Phase string `json:"phase"`
	Code  string `json:"code"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "velox-startup-summary:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("velox-startup-summary", flag.ContinueOnError)
	input := flags.String("input", "", "startup lifecycle evidence JSON")
	output := flags.String("output", "", "summary JSON output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *input == "" || *output == "" || flags.NArg() != 0 {
		return errors.New("--input and --output are required")
	}
	body, err := os.ReadFile(*input)
	if err != nil {
		return err
	}
	var raw evidence
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return fmt.Errorf("decode evidence: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("decode evidence: trailing JSON value")
	}
	result, err := summarize(raw, body)
	if err != nil {
		return err
	}
	return writeJSON(*output, result)
}

func summarize(raw evidence, source []byte) (summary, error) {
	if raw.SchemaVersion != "velox.startup-lifecycle/v2" {
		return summary{}, fmt.Errorf("unsupported source schema %q", raw.SchemaVersion)
	}
	if raw.Scope != "fresh-and-immediate-same-profile-startup" {
		return summary{}, fmt.Errorf("unsupported source scope %q", raw.Scope)
	}
	if raw.Repetitions != len(raw.Samples) {
		return summary{}, errors.New("repetitions does not match observed sample count")
	}
	digest := sha256.Sum256(source)
	result := summary{
		SchemaVersion: summarySchemaVersion, SourceSchemaVersion: raw.SchemaVersion,
		SourceSHA256: hex.EncodeToString(digest[:]), EvidenceLevel: raw.EvidenceLevel,
		Outcome: raw.Outcome, Repetitions: raw.Repetitions, ObservedSamples: len(raw.Samples),
		Environment: raw.Environment, Metrics: make(map[string]metricStats), FailedSamples: []failedSample{},
		Correlation: correlation{MetricX: "firstBrowserExitAfterImmediateStartMs", MetricY: "immediateReadyMs"},
		Ordering:    ordering{ViolationSampleIndexes: []int{}},
	}
	values := map[string][]float64{
		"firstReadyMs": {}, "immediateReadyMs": {}, "firstBrowserExitAfterHostMs": {},
		"immediateBrowserExitAfterHostMs": {}, "profileReleaseMs": {},
		"firstBrowserExitAfterImmediateStartMs": {}, "immediateReadyAfterFirstBrowserExitMs": {},
	}
	for _, item := range raw.Samples {
		if item.Outcome != "success" {
			result.FailureCount++
			failure := failedSample{Index: item.Index}
			if item.Error != nil {
				failure.Phase, failure.Code = item.Error.Phase, item.Error.Code
			}
			result.FailedSamples = append(result.FailedSamples, failure)
			continue
		}
		if item.First == nil || item.Immediate == nil || item.ProfileReleaseMs == nil || item.Timeline == nil {
			return summary{}, fmt.Errorf("successful sample %d is incomplete", item.Index)
		}
		result.SuccessCount++
		values["firstReadyMs"] = append(values["firstReadyMs"], item.First.ReadyMs)
		values["immediateReadyMs"] = append(values["immediateReadyMs"], item.Immediate.ReadyMs)
		values["firstBrowserExitAfterHostMs"] = append(values["firstBrowserExitAfterHostMs"], item.First.BrowserExitAfterHostMs)
		values["immediateBrowserExitAfterHostMs"] = append(values["immediateBrowserExitAfterHostMs"], item.Immediate.BrowserExitAfterHostMs)
		values["profileReleaseMs"] = append(values["profileReleaseMs"], *item.ProfileReleaseMs)
		values["firstBrowserExitAfterImmediateStartMs"] = append(values["firstBrowserExitAfterImmediateStartMs"], item.Timeline.FirstBrowserExitAfterImmediateStartMs)
		values["immediateReadyAfterFirstBrowserExitMs"] = append(values["immediateReadyAfterFirstBrowserExitMs"], item.Timeline.ImmediateReadyAfterFirstBrowserExitMs)
		if item.Timeline.ImmediateReadyWaitedForFirstBrowserExit {
			result.Ordering.ReadyWaitedForFirstBrowserExitCount++
		} else {
			result.Ordering.ReadyBeforeFirstBrowserExitCount++
			result.Ordering.ViolationSampleIndexes = append(result.Ordering.ViolationSampleIndexes, item.Index)
		}
	}
	for name, metricValues := range values {
		if len(metricValues) > 0 {
			result.Metrics[name] = describe(metricValues)
		}
	}
	result.Correlation.SampleCount = result.SuccessCount
	result.Correlation.PearsonCoefficient = pearson(values["firstBrowserExitAfterImmediateStartMs"], values["immediateReadyMs"])
	expectedOutcome := "success"
	if result.FailureCount > 0 {
		expectedOutcome = "failure"
	}
	if raw.Outcome != expectedOutcome {
		return summary{}, fmt.Errorf("outcome %q does not match sample results", raw.Outcome)
	}
	return result, nil
}

func describe(values []float64) metricStats {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return metricStats{MinimumMs: sorted[0], P50Ms: nearestRank(sorted, 0.50), P95Ms: nearestRank(sorted, 0.95), MaximumMs: sorted[len(sorted)-1]}
}

func nearestRank(sorted []float64, percentile float64) float64 {
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

func pearson(x, y []float64) *float64 {
	if len(x) != len(y) || len(x) < 2 {
		return nil
	}
	var sumX, sumY float64
	for index := range x {
		sumX, sumY = sumX+x[index], sumY+y[index]
	}
	meanX, meanY := sumX/float64(len(x)), sumY/float64(len(y))
	var numerator, squareX, squareY float64
	for index := range x {
		dx, dy := x[index]-meanX, y[index]-meanY
		numerator += dx * dy
		squareX += dx * dx
		squareY += dy * dy
	}
	denominator := math.Sqrt(squareX * squareY)
	if denominator == 0 {
		return nil
	}
	value := math.Max(-1, math.Min(1, numerator/denominator))
	return &value
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, body, 0o644); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}
