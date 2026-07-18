package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultAPIBase = "https://api.github.com"

var actionUsePattern = regexp.MustCompile(`^\s*(?:-\s*)?uses:\s+(actions/[A-Za-z0-9_.-]+)@([0-9a-f]{40})\s+#\s*(v[0-9][A-Za-z0-9_.-]*)\s*$`)

type actionPin struct {
	Repository string
	SHA        string
	Version    string
	Locations  []string
}

type githubClient struct {
	baseURL *url.URL
	client  *http.Client
	token   string
}

type gitObject struct {
	SHA  string `json:"sha"`
	Type string `json:"type"`
}

type verification struct {
	Pin actionPin
	Err error
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "actutum-action-pins:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("actutum-action-pins", flag.ContinueOnError)
	root := flags.String("root", ".github/workflows", "workflow directory")
	apiBase := flags.String("api-base", defaultAPIBase, "GitHub API base URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	pins, err := discoverPins(*root)
	if err != nil {
		return err
	}
	baseURL, err := url.Parse(*apiBase)
	if err != nil {
		return fmt.Errorf("parse API base: %w", err)
	}
	checker := githubClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
		token:   os.Getenv("GITHUB_TOKEN"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, result := range verifyAll(ctx, checker, pins) {
		if result.Err != nil {
			return result.Err
		}
		pin := result.Pin
		fmt.Printf("verified %s %s %s (%d use sites)\n", pin.Repository, pin.Version, pin.SHA, len(pin.Locations))
	}
	return nil
}

func verifyAll(ctx context.Context, client githubClient, pins []actionPin) []verification {
	results := make([]verification, len(pins))
	var wait sync.WaitGroup
	wait.Add(len(pins))
	for index, pin := range pins {
		go func() {
			defer wait.Done()
			results[index] = verification{Pin: pin, Err: client.verify(ctx, pin)}
		}()
	}
	wait.Wait()
	return results
}

func discoverPins(root string) ([]actionPin, error) {
	byRepository := make(map[string]actionPin)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (filepath.Ext(path) != ".yml" && filepath.Ext(path) != ".yaml") {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()
			if !strings.Contains(line, "uses: actions/") {
				continue
			}
			match := actionUsePattern.FindStringSubmatch(line)
			if match == nil {
				return fmt.Errorf("%s:%d must pin actions/* to a 40-character SHA with a version comment", path, lineNumber)
			}
			location := fmt.Sprintf("%s:%d", filepath.ToSlash(path), lineNumber)
			current, exists := byRepository[match[1]]
			if exists && (current.SHA != match[2] || current.Version != match[3]) {
				return fmt.Errorf("%s uses inconsistent pins", match[1])
			}
			current.Repository, current.SHA, current.Version = match[1], match[2], match[3]
			current.Locations = append(current.Locations, location)
			byRepository[match[1]] = current
		}
		return scanner.Err()
	})
	if err != nil {
		return nil, err
	}
	if len(byRepository) == 0 {
		return nil, errors.New("no actions/* workflow uses found")
	}
	result := make([]actionPin, 0, len(byRepository))
	for _, pin := range byRepository {
		result = append(result, pin)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Repository < result[j].Repository })
	return result, nil
}

func (client githubClient) verify(ctx context.Context, pin actionPin) error {
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := client.get(ctx, "/repos/"+pin.Repository+"/releases/latest", &release); err != nil {
		return fmt.Errorf("%s latest release: %w", pin.Repository, err)
	}
	if release.TagName != pin.Version {
		return fmt.Errorf("%s is pinned to %s, latest stable release is %s", pin.Repository, pin.Version, release.TagName)
	}
	var reference struct {
		Object gitObject `json:"object"`
	}
	if err := client.get(ctx, "/repos/"+pin.Repository+"/git/ref/tags/"+url.PathEscape(pin.Version), &reference); err != nil {
		return fmt.Errorf("%s tag %s: %w", pin.Repository, pin.Version, err)
	}
	object := reference.Object
	if object.Type == "tag" {
		var annotated struct {
			Object gitObject `json:"object"`
		}
		if err := client.get(ctx, "/repos/"+pin.Repository+"/git/tags/"+object.SHA, &annotated); err != nil {
			return fmt.Errorf("%s annotated tag %s: %w", pin.Repository, pin.Version, err)
		}
		object = annotated.Object
	}
	if object.Type != "commit" {
		return fmt.Errorf("%s tag %s resolves to %s instead of commit", pin.Repository, pin.Version, object.Type)
	}
	if !strings.EqualFold(object.SHA, pin.SHA) {
		return fmt.Errorf("%s %s resolves to %s, workflow pins %s", pin.Repository, pin.Version, object.SHA, pin.SHA)
	}
	return nil
}

func (client githubClient) get(ctx context.Context, path string, output any) error {
	reference, err := client.baseURL.Parse(path)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, reference.String(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "actutum-action-pins")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if client.token != "" {
		request.Header.Set("Authorization", "Bearer "+client.token)
	}
	response, err := client.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %s", response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(output); err != nil {
		return fmt.Errorf("decode GitHub response: %w", err)
	}
	return nil
}
