package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const monitorSchemaVersion = "actutum.actions-warning-monitor/v1"

var downloadArtifactBufferWarning = warningSignature{
	Action: "actions/download-artifact",
	Code:   "DEP0005",
	Needles: []string{
		"[DEP0005] DeprecationWarning: Buffer() is deprecated",
		"Buffer() is deprecated due to security and usability issues",
	},
}

type warningSignature struct {
	Action  string
	Code    string
	Needles []string
}

type report struct {
	SchemaVersion string    `json:"schemaVersion"`
	CheckedAtUTC  string    `json:"checkedAtUtc"`
	Repository    string    `json:"repository"`
	RunID         int64     `json:"runId"`
	Status        string    `json:"status"`
	Findings      []finding `json:"findings"`
}

type finding struct {
	Action          string `json:"action"`
	Code            string `json:"code"`
	OccurrenceCount int    `json:"occurrenceCount"`
}

type monitor struct {
	client  *http.Client
	baseURL string
	token   string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "actutum-actions-warning-monitor:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("actutum-actions-warning-monitor", flag.ContinueOnError)
	repository := flags.String("repository", "", "GitHub owner/repository")
	runIDText := flags.String("run-id", "", "completed GitHub Actions run ID")
	output := flags.String("output", "", "monitor JSON output")
	apiBase := flags.String("api-base", "https://api.github.com", "GitHub API base URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *repository == "" || *runIDText == "" || *output == "" || flags.NArg() != 0 {
		return errors.New("--repository, --run-id, and --output are required")
	}
	runID, err := strconv.ParseInt(*runIDText, 10, 64)
	if err != nil || runID < 1 {
		return errors.New("--run-id must be a positive integer")
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return errors.New("GITHUB_TOKEN is required")
	}
	m := monitor{client: http.DefaultClient, baseURL: strings.TrimRight(*apiBase, "/"), token: token}
	archive, err := m.downloadLogs(context.Background(), *repository, runID)
	if err != nil {
		return err
	}
	result, err := inspectLogs(*repository, runID, archive, time.Now().UTC())
	if err != nil {
		return err
	}
	return writeJSON(*output, result)
}

func (m monitor) downloadLogs(ctx context.Context, repository string, runID int64) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/actions/runs/%d/logs", m.baseURL, repository, runID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+m.token)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "actutum-actions-warning-monitor")
	response, err := m.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := readBounded(response.Body, 64<<20)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub API returned %s", response.Status)
	}
	return body, nil
}

func inspectLogs(repository string, runID int64, archive []byte, now time.Time) (report, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return report{}, fmt.Errorf("open workflow log archive: %w", err)
	}
	needleCounts := make([]int, len(downloadArtifactBufferWarning.Needles))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			return report{}, err
		}
		body, readErr := readBounded(stream, 16<<20)
		closeErr := stream.Close()
		if readErr != nil {
			return report{}, readErr
		}
		if closeErr != nil {
			return report{}, closeErr
		}
		text := string(body)
		for index, needle := range downloadArtifactBufferWarning.Needles {
			needleCounts[index] += strings.Count(text, needle)
		}
	}
	occurrences := 0
	for _, count := range needleCounts {
		if count > occurrences {
			occurrences = count
		}
	}
	result := report{
		SchemaVersion: monitorSchemaVersion, CheckedAtUTC: now.Format(time.RFC3339), Repository: repository,
		RunID: runID, Status: "absent", Findings: []finding{},
	}
	if occurrences > 0 {
		result.Status = "present"
		result.Findings = append(result.Findings, finding{
			Action: downloadArtifactBufferWarning.Action, Code: downloadArtifactBufferWarning.Code, OccurrenceCount: occurrences,
		})
	}
	return result, nil
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("evidence exceeds %d-byte limit", limit)
	}
	return body, nil
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
