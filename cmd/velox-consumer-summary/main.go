package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const scope = "checkout-complete-to-portable-zip"

type observation struct {
	EvidenceLevel string  `json:"evidenceLevel"`
	SampleID      string  `json:"sampleId"`
	Outcome       string  `json:"outcome"`
	DurationMS    float64 `json:"durationMs"`
	Acquisition   *struct {
		ArchiveSHA256 string `json:"archiveSha256"`
	} `json:"acquisition"`
	Build *struct {
		ProcessTrace struct {
			Status string `json:"status"`
		} `json:"processTrace"`
	} `json:"build"`
	Error *struct {
		Phase string `json:"phase"`
	} `json:"error"`
}

type sample struct {
	SampleID              string  `json:"sampleId"`
	Outcome               string  `json:"outcome"`
	DurationMS            float64 `json:"durationMs"`
	FailurePhase          *string `json:"failurePhase"`
	ProcessEvidenceStatus *string `json:"processEvidenceStatus"`
}

type statistics struct {
	MinMS float64 `json:"minMs"`
	P50MS float64 `json:"p50Ms"`
	P95MS float64 `json:"p95Ms"`
	MaxMS float64 `json:"maxMs"`
}

type summary struct {
	SchemaVersion                  string      `json:"schemaVersion"`
	Scope                          string      `json:"scope"`
	EvidenceLevel                  string      `json:"evidenceLevel"`
	ExpectedSamples                int         `json:"expectedSamples"`
	ObservedSamples                int         `json:"observedSamples"`
	SuccessCount                   int         `json:"successCount"`
	FailureCount                   int         `json:"failureCount"`
	MissingCount                   int         `json:"missingCount"`
	ProcessEvidencePassCount       int         `json:"processEvidencePassCount"`
	ProcessEvidenceFailCount       int         `json:"processEvidenceFailCount"`
	ProcessEvidenceUnverifiedCount int         `json:"processEvidenceUnverifiedCount"`
	ReleaseArchiveSHA256           *string     `json:"releaseArchiveSha256"`
	Statistics                     *statistics `json:"statistics"`
	Samples                        []sample    `json:"samples"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "velox-consumer-summary:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("velox-consumer-summary", flag.ContinueOnError)
	root := flags.String("results-root", "", "raw JSON file or directory (recursive)")
	output := flags.String("output", "", "summary JSON output, outside the raw results directory")
	expected := flags.Int("expected-samples", 0, "expected sample count (1-100)")
	rawPath := flags.String("raw-schema", "schema/consumer-e2e-v1.schema.json", "raw result JSON schema")
	summaryPath := flags.String("summary-schema", "schema/consumer-e2e-summary-v1.schema.json", "summary JSON schema")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *root == "" || *output == "" || *expected < 1 || *expected > 100 || flags.NArg() != 0 {
		return errors.New("--results-root, --output and --expected-samples (1-100) are required")
	}
	rawSchema, err := compileSchema(*rawPath)
	if err != nil {
		return err
	}
	summarySchema, err := compileSchema(*summaryPath)
	if err != nil {
		return err
	}
	paths, err := resultPaths(*root)
	if err != nil {
		return err
	}
	outputPath, err := filepath.Abs(*output)
	if err != nil {
		return err
	}
	outputInfo, err := os.Stat(outputPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	observations := make([]observation, 0, len(paths))
	for _, path := range paths {
		inputPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		inputInfo, err := os.Stat(inputPath)
		if err != nil {
			return err
		}
		if inputPath == outputPath || (outputInfo != nil && os.SameFile(inputInfo, outputInfo)) {
			return errors.New("output must not overwrite a raw result")
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := validateJSON(body, rawSchema); err != nil {
			return fmt.Errorf("invalid raw result %s: %w", filepath.Base(path), err)
		}
		var item observation
		if err := json.Unmarshal(body, &item); err != nil {
			return err
		}
		observations = append(observations, item)
	}
	result, evidenceErr := summarize(observations, *expected)
	if result.SchemaVersion == "" {
		return evidenceErr
	}
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := validateJSON(body, summarySchema); err != nil {
		return fmt.Errorf("invalid summary: %w", err)
	}
	// Incomplete evidence is still written so CI can upload the failure report.
	if err := writeResult(outputPath, body); err != nil {
		return err
	}
	if _, err := stdout.Write(body); err != nil {
		return err
	}
	return evidenceErr
}

type offlineLoader struct{}

func (offlineLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external schema loading is disabled: %s", url)
}

func compileSchema(path string) (*jsonschema.Schema, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := decodeJSON(body)
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(offlineLoader{})
	compiler.AssertFormat()
	const location = "urn:velox:consumer-schema"
	if err := compiler.AddResource(location, doc); err != nil {
		return nil, err
	}
	return compiler.Compile(location)
}

func decodeJSON(body []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing JSON value or invalid content")
	}
	return value, nil
}

func validateJSON(body []byte, schema *jsonschema.Schema) error {
	doc, err := decodeJSON(body)
	if err != nil {
		return err
	}
	return schema.Validate(doc)
}

func resultPaths(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{root}, nil
	}
	paths := []string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() && strings.EqualFold(filepath.Ext(path), ".json") {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func summarize(items []observation, expected int) (summary, error) {
	result := summary{
		SchemaVersion: "velox.consumer-e2e-summary/v1", Scope: scope,
		EvidenceLevel: "local-contract-summary", ExpectedSamples: expected,
		ObservedSamples: len(items), MissingCount: max(0, expected-len(items)),
		Samples: []sample{},
	}
	seen, digests := map[string]bool{}, map[string]bool{}
	hostedOnly := len(items) > 0
	durations := []float64{}
	for _, item := range items {
		id := strings.ToLower(item.SampleID)
		if seen[id] {
			return summary{}, errors.New("raw results contain duplicate sample IDs")
		}
		seen[id] = true
		hostedOnly = hostedOnly && item.EvidenceLevel == "hosted-runner-evidence"
		row := sample{SampleID: item.SampleID, Outcome: item.Outcome, DurationMS: item.DurationMS}
		if item.Error != nil {
			row.FailurePhase = &item.Error.Phase
		}
		if item.Build != nil {
			row.ProcessEvidenceStatus = &item.Build.ProcessTrace.Status
		}
		result.Samples = append(result.Samples, row)
		if item.Outcome == "failure" {
			result.FailureCount++
			continue
		}
		result.SuccessCount++
		durations = append(durations, item.DurationMS)
		digests[item.Acquisition.ArchiveSHA256] = true
		switch item.Build.ProcessTrace.Status {
		case "pass":
			result.ProcessEvidencePassCount++
		case "fail":
			result.ProcessEvidenceFailCount++
		case "unverified":
			result.ProcessEvidenceUnverifiedCount++
		}
	}
	if hostedOnly {
		result.EvidenceLevel = "hosted-runner-summary"
	}
	if len(digests) == 1 {
		for digest := range digests {
			result.ReleaseArchiveSHA256 = &digest
		}
	}
	sort.Slice(result.Samples, func(i, j int) bool {
		return strings.ToLower(result.Samples[i].SampleID) < strings.ToLower(result.Samples[j].SampleID)
	})
	if len(durations) > 0 {
		sort.Float64s(durations)
		round := func(value float64) float64 { return math.RoundToEven(value*1000) / 1000 }
		rank := func(p float64) float64 { return round(durations[int(math.Ceil(p*float64(len(durations))))-1]) }
		result.Statistics = &statistics{round(durations[0]), rank(0.50), rank(0.95), round(durations[len(durations)-1])}
	}
	if len(items) != expected || result.FailureCount > 0 || len(digests) != 1 ||
		(hostedOnly && (result.ProcessEvidenceFailCount > 0 || result.ProcessEvidenceUnverifiedCount > 0)) {
		return result, errors.New("consumer end-to-end evidence is incomplete or inconsistent")
	}
	return result, nil
}

func writeResult(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".consumer-summary-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(body); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
