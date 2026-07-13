package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDiscoverPinsRejectsMutableReference(t *testing.T) {
	root := t.TempDir()
	workflow := filepath.Join(root, "test.yml")
	if err := os.WriteFile(workflow, []byte("steps:\n  - uses: actions/checkout@v7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := discoverPins(root)
	if err == nil || !strings.Contains(err.Error(), "40-character SHA") {
		t.Fatalf("discoverPins error = %v", err)
	}
}

func TestDiscoverPinsCoalescesMatchingUses(t *testing.T) {
	root := t.TempDir()
	body := "steps:\n  - uses: actions/checkout@1111111111111111111111111111111111111111 # v7.0.0\n  - uses: actions/checkout@1111111111111111111111111111111111111111 # v7.0.0\n"
	if err := os.WriteFile(filepath.Join(root, "test.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	pins, err := discoverPins(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 1 || len(pins[0].Locations) != 2 {
		t.Fatalf("pins = %#v", pins)
	}
}

func TestRepositoryWorkflowPinsAreWellFormed(t *testing.T) {
	pins, err := discoverPins(filepath.Join("..", "..", ".github", "workflows"))
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 4 {
		t.Fatalf("action repositories = %d, want 4", len(pins))
	}
}

func TestVerifyChecksLatestReleaseAndTagCommit(t *testing.T) {
	const sha = "1111111111111111111111111111111111111111"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/actions/checkout/releases/latest":
			fmt.Fprint(response, `{"tag_name":"v7.0.0"}`)
		case "/repos/actions/checkout/git/ref/tags/v7.0.0":
			fmt.Fprintf(response, `{"object":{"sha":%q,"type":"commit"}}`, sha)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	client := githubClient{baseURL: mustURL(t, server.URL), client: server.Client()}
	if err := client.verify(context.Background(), actionPin{Repository: "actions/checkout", Version: "v7.0.0", SHA: sha}); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRejectsStaleRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(response, `{"tag_name":"v8.0.0"}`)
	}))
	defer server.Close()

	client := githubClient{baseURL: mustURL(t, server.URL), client: server.Client()}
	err := client.verify(context.Background(), actionPin{Repository: "actions/checkout", Version: "v7.0.0", SHA: strings.Repeat("1", 40)})
	if err == nil || !strings.Contains(err.Error(), "latest stable release") {
		t.Fatalf("verify error = %v", err)
	}
}

func TestVerifyAllRunsRepositoriesConcurrentlyAndKeepsOrder(t *testing.T) {
	const sha = "1111111111111111111111111111111111111111"
	var active int32
	var maximum int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		current := atomic.AddInt32(&active, 1)
		defer atomic.AddInt32(&active, -1)
		for {
			observed := atomic.LoadInt32(&maximum)
			if current <= observed || atomic.CompareAndSwapInt32(&maximum, observed, current) {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
		if strings.HasSuffix(request.URL.Path, "/releases/latest") {
			fmt.Fprint(response, `{"tag_name":"v1.0.0"}`)
			return
		}
		fmt.Fprintf(response, `{"object":{"sha":%q,"type":"commit"}}`, sha)
	}))
	defer server.Close()

	client := githubClient{baseURL: mustURL(t, server.URL), client: server.Client()}
	pins := []actionPin{
		{Repository: "actions/a", Version: "v1.0.0", SHA: sha},
		{Repository: "actions/b", Version: "v1.0.0", SHA: sha},
		{Repository: "actions/c", Version: "v1.0.0", SHA: sha},
		{Repository: "actions/d", Version: "v1.0.0", SHA: sha},
	}
	results := verifyAll(context.Background(), client, pins)
	if atomic.LoadInt32(&maximum) < 2 {
		t.Fatalf("maximum concurrent requests = %d, want at least 2", maximum)
	}
	for index, result := range results {
		if result.Err != nil {
			t.Fatal(result.Err)
		}
		if result.Pin.Repository != pins[index].Repository {
			t.Fatalf("result %d repository = %s, want %s", index, result.Pin.Repository, pins[index].Repository)
		}
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	value, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
