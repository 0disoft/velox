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
	"sort"
	"strconv"
	"strings"
	"time"
)

const historySchemaVersion = "velox.startup-history/v1"

type metricStats struct {
	P50Ms float64 `json:"p50Ms"`
	P95Ms float64 `json:"p95Ms"`
}

type lifecycleSummary struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Outcome       string                 `json:"outcome"`
	Environment   environment            `json:"environment"`
	Metrics       map[string]metricStats `json:"metrics"`
	Correlation   correlation            `json:"correlation"`
	Ordering      ordering               `json:"ordering"`
}

type environment struct {
	RunnerImage        *string `json:"runnerImage"`
	RunnerImageVersion *string `json:"runnerImageVersion"`
	WebView2Version    string  `json:"webView2Version"`
	GitHubRunID        *string `json:"githubRunId"`
	GitHubRunAttempt   *string `json:"githubRunAttempt"`
	GitCommit          *string `json:"gitCommit"`
}

type correlation struct {
	PearsonCoefficient *float64 `json:"pearsonCoefficient"`
}

type ordering struct {
	ReadyWaitedForFirstBrowserExitCount int `json:"readyWaitedForFirstBrowserExitCount"`
}

type history struct {
	SchemaVersion     string             `json:"schemaVersion"`
	GeneratedAtUTC    string             `json:"generatedAtUtc"`
	Repository        string             `json:"repository"`
	Workflow          string             `json:"workflow"`
	RequestedLimit    int                `json:"requestedLimit"`
	SourceCount       int                `json:"sourceCount"`
	Series            []historyPoint     `json:"series"`
	EnvironmentGroups []environmentGroup `json:"environmentGroups"`
	CollectionIssues  []collectionIssue  `json:"collectionIssues"`
}

type historyPoint struct {
	RunID                               int64    `json:"runId"`
	RunAttempt                          int      `json:"runAttempt"`
	HeadSHA                             string   `json:"headSha"`
	CreatedAtUTC                        string   `json:"createdAtUtc"`
	RunnerImage                         string   `json:"runnerImage"`
	RunnerImageVersion                  string   `json:"runnerImageVersion"`
	WebView2Version                     string   `json:"webView2Version"`
	ImmediateReadyP50Ms                 float64  `json:"immediateReadyP50Ms"`
	ImmediateReadyP95Ms                 float64  `json:"immediateReadyP95Ms"`
	FirstBrowserExitAfterImmediateP50Ms float64  `json:"firstBrowserExitAfterImmediateP50Ms"`
	FirstBrowserExitAfterImmediateP95Ms float64  `json:"firstBrowserExitAfterImmediateP95Ms"`
	ReadyAfterFirstBrowserExitP50Ms     float64  `json:"readyAfterFirstBrowserExitP50Ms"`
	ReadyAfterFirstBrowserExitP95Ms     float64  `json:"readyAfterFirstBrowserExitP95Ms"`
	ReadyWaitedForFirstBrowserExitCount int      `json:"readyWaitedForFirstBrowserExitCount"`
	PearsonCoefficient                  *float64 `json:"pearsonCoefficient"`
}

type environmentGroup struct {
	RunnerImage        string `json:"runnerImage"`
	RunnerImageVersion string `json:"runnerImageVersion"`
	WebView2Version    string `json:"webView2Version"`
	SampleCount        int    `json:"sampleCount"`
}

type collectionIssue struct {
	RunID int64  `json:"runId"`
	Code  string `json:"code"`
}

type workflowRunsResponse struct {
	WorkflowRuns []workflowRun `json:"workflow_runs"`
}

type workflowRun struct {
	ID         int64  `json:"id"`
	RunAttempt int    `json:"run_attempt"`
	HeadSHA    string `json:"head_sha"`
	CreatedAt  string `json:"created_at"`
}

type artifactsResponse struct {
	Artifacts []artifact `json:"artifacts"`
}

type artifact struct {
	Name               string `json:"name"`
	Expired            bool   `json:"expired"`
	ArchiveDownloadURL string `json:"archive_download_url"`
}

type collector struct {
	client  *http.Client
	baseURL string
	token   string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "velox-startup-history:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("velox-startup-history", flag.ContinueOnError)
	repository := flags.String("repository", "", "GitHub owner/repository")
	workflow := flags.String("workflow", "consumer-evidence.yml", "workflow file name")
	currentPath := flags.String("current", "", "current lifecycle summary JSON")
	output := flags.String("output", "", "history JSON output")
	limit := flags.Int("limit", 12, "maximum number of history points")
	apiBase := flags.String("api-base", "https://api.github.com", "GitHub API base URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *repository == "" || *currentPath == "" || *output == "" || flags.NArg() != 0 {
		return errors.New("--repository, --current, and --output are required")
	}
	if *limit < 1 || *limit > 52 {
		return errors.New("--limit must be between 1 and 52")
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return errors.New("GITHUB_TOKEN is required")
	}
	currentBody, err := os.ReadFile(*currentPath)
	if err != nil {
		return err
	}
	current, err := decodeSummary(currentBody)
	if err != nil {
		return fmt.Errorf("decode current summary: %w", err)
	}
	c := collector{client: http.DefaultClient, baseURL: strings.TrimRight(*apiBase, "/"), token: token}
	result, err := c.collect(context.Background(), *repository, *workflow, current, *limit, time.Now().UTC())
	if err != nil {
		return err
	}
	return writeJSON(*output, result)
}

func (c collector) collect(ctx context.Context, repository, workflow string, current lifecycleSummary, limit int, now time.Time) (history, error) {
	currentRunID, err := requiredRunID(current.Environment.GitHubRunID)
	if err != nil {
		return history{}, err
	}
	currentRun, err := c.getRun(ctx, repository, currentRunID)
	if err != nil {
		return history{}, fmt.Errorf("read current run: %w", err)
	}
	currentPoint, err := makePoint(currentRun, current)
	if err != nil {
		return history{}, fmt.Errorf("current summary: %w", err)
	}
	result := history{
		SchemaVersion: historySchemaVersion, GeneratedAtUTC: now.Format(time.RFC3339), Repository: repository,
		Workflow: workflow, RequestedLimit: limit, Series: []historyPoint{currentPoint},
		EnvironmentGroups: []environmentGroup{}, CollectionIssues: []collectionIssue{},
	}
	if limit > 1 {
		runs, err := c.listRuns(ctx, repository, workflow, limit*3)
		if err != nil {
			return history{}, fmt.Errorf("list scheduled runs: %w", err)
		}
		for _, item := range runs {
			if len(result.Series) >= limit {
				break
			}
			if item.ID == currentRunID {
				continue
			}
			body, code, err := c.downloadSummary(ctx, repository, item.ID)
			if err != nil {
				result.CollectionIssues = append(result.CollectionIssues, collectionIssue{RunID: item.ID, Code: code})
				continue
			}
			summary, err := decodeSummary(body)
			if err != nil {
				result.CollectionIssues = append(result.CollectionIssues, collectionIssue{RunID: item.ID, Code: "INVALID_SUMMARY"})
				continue
			}
			point, err := makePoint(item, summary)
			if err != nil {
				result.CollectionIssues = append(result.CollectionIssues, collectionIssue{RunID: item.ID, Code: "INCOMPLETE_SUMMARY"})
				continue
			}
			result.Series = append(result.Series, point)
		}
	}
	sort.Slice(result.Series, func(i, j int) bool { return result.Series[i].CreatedAtUTC < result.Series[j].CreatedAtUTC })
	result.SourceCount = len(result.Series)
	result.EnvironmentGroups = groupEnvironments(result.Series)
	return result, nil
}

func (c collector) getRun(ctx context.Context, repository string, runID int64) (workflowRun, error) {
	var result workflowRun
	err := c.getJSON(ctx, fmt.Sprintf("/repos/%s/actions/runs/%d", repository, runID), &result)
	return result, err
}

func (c collector) listRuns(ctx context.Context, repository, workflow string, perPage int) ([]workflowRun, error) {
	if perPage > 100 {
		perPage = 100
	}
	var result workflowRunsResponse
	path := fmt.Sprintf("/repos/%s/actions/workflows/%s/runs?event=schedule&status=success&per_page=%d", repository, workflow, perPage)
	if err := c.getJSON(ctx, path, &result); err != nil {
		return nil, err
	}
	return result.WorkflowRuns, nil
}

func (c collector) downloadSummary(ctx context.Context, repository string, runID int64) ([]byte, string, error) {
	var listed artifactsResponse
	if err := c.getJSON(ctx, fmt.Sprintf("/repos/%s/actions/runs/%d/artifacts?per_page=100", repository, runID), &listed); err != nil {
		return nil, "ARTIFACT_LIST_FAILED", err
	}
	var selected *artifact
	for index := range listed.Artifacts {
		item := &listed.Artifacts[index]
		if strings.HasPrefix(item.Name, "startup-lifecycle-") && !item.Expired {
			selected = item
			break
		}
	}
	if selected == nil {
		return nil, "ARTIFACT_UNAVAILABLE", errors.New("startup lifecycle artifact is unavailable")
	}
	body, err := c.getBytes(ctx, selected.ArchiveDownloadURL)
	if err != nil {
		return nil, "ARTIFACT_DOWNLOAD_FAILED", err
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, "INVALID_ARTIFACT_ARCHIVE", err
	}
	for _, file := range reader.File {
		if filepath.Base(file.Name) != "startup-lifecycle-summary.json" {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			return nil, "SUMMARY_READ_FAILED", err
		}
		result, readErr := readBounded(stream, 2<<20)
		closeErr := stream.Close()
		if readErr != nil {
			return nil, "SUMMARY_READ_FAILED", readErr
		}
		if closeErr != nil {
			return nil, "SUMMARY_READ_FAILED", closeErr
		}
		return result, "", nil
	}
	return nil, "SUMMARY_MISSING", errors.New("startup lifecycle summary is missing from artifact")
}

func (c collector) getJSON(ctx context.Context, path string, target any) error {
	body, err := c.getBytes(ctx, c.baseURL+path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func (c collector) getBytes(ctx context.Context, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "velox-startup-history")
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := readBounded(response.Body, 16<<20)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub API returned %s", response.Status)
	}
	return body, nil
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

func decodeSummary(body []byte) (lifecycleSummary, error) {
	var result lifecycleSummary
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&result); err != nil {
		return result, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return result, errors.New("trailing JSON value")
	}
	if result.SchemaVersion != "velox.startup-lifecycle-summary/v1" || result.Outcome != "success" {
		return result, errors.New("summary is not successful startup lifecycle v1 evidence")
	}
	return result, nil
}

func makePoint(run workflowRun, raw lifecycleSummary) (historyPoint, error) {
	metric := func(name string) (metricStats, error) {
		value, ok := raw.Metrics[name]
		if !ok {
			return metricStats{}, fmt.Errorf("metric %s is missing", name)
		}
		return value, nil
	}
	immediate, err := metric("immediateReadyMs")
	if err != nil {
		return historyPoint{}, err
	}
	exit, err := metric("firstBrowserExitAfterImmediateStartMs")
	if err != nil {
		return historyPoint{}, err
	}
	afterExit, err := metric("immediateReadyAfterFirstBrowserExitMs")
	if err != nil {
		return historyPoint{}, err
	}
	return historyPoint{
		RunID: run.ID, RunAttempt: run.RunAttempt, HeadSHA: run.HeadSHA, CreatedAtUTC: run.CreatedAt,
		RunnerImage: value(raw.Environment.RunnerImage), RunnerImageVersion: value(raw.Environment.RunnerImageVersion),
		WebView2Version: raw.Environment.WebView2Version, ImmediateReadyP50Ms: immediate.P50Ms, ImmediateReadyP95Ms: immediate.P95Ms,
		FirstBrowserExitAfterImmediateP50Ms: exit.P50Ms, FirstBrowserExitAfterImmediateP95Ms: exit.P95Ms,
		ReadyAfterFirstBrowserExitP50Ms: afterExit.P50Ms, ReadyAfterFirstBrowserExitP95Ms: afterExit.P95Ms,
		ReadyWaitedForFirstBrowserExitCount: raw.Ordering.ReadyWaitedForFirstBrowserExitCount,
		PearsonCoefficient:                  raw.Correlation.PearsonCoefficient,
	}, nil
}

func groupEnvironments(series []historyPoint) []environmentGroup {
	groups := map[string]environmentGroup{}
	for _, point := range series {
		key := point.RunnerImage + "\x00" + point.RunnerImageVersion + "\x00" + point.WebView2Version
		group := groups[key]
		group.RunnerImage, group.RunnerImageVersion, group.WebView2Version = point.RunnerImage, point.RunnerImageVersion, point.WebView2Version
		group.SampleCount++
		groups[key] = group
	}
	result := make([]environmentGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, group)
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i].RunnerImage + result[i].RunnerImageVersion + result[i].WebView2Version
		right := result[j].RunnerImage + result[j].RunnerImageVersion + result[j].WebView2Version
		return left < right
	})
	return result
}

func requiredRunID(raw *string) (int64, error) {
	if raw == nil || *raw == "" {
		return 0, errors.New("current summary has no GitHub run ID")
	}
	result, err := strconv.ParseInt(*raw, 10, 64)
	if err != nil || result < 1 {
		return 0, errors.New("current summary has an invalid GitHub run ID")
	}
	return result, nil
}

func value(raw *string) string {
	if raw == nil {
		return ""
	}
	return *raw
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
