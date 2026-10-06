package runner

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validWatchManifest = `{"schemaVersion":1,"app":{"id":"com.example.run","name":"Run","version":"1.0.0"},"assets":{"root":"site","entry":"index.html"}}`

func writeWatchManifest(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func pollManifest(t *testing.T, w *manifestWatch, now time.Time, changed, failed bool) {
	t.Helper()
	got, err := w.poll(now)
	if got != changed || (err != nil) != failed {
		t.Fatalf("changed=%t want=%t err=%v wantError=%t", got, changed, err, failed)
	}
}

func TestManifestWatchStableContentChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom-config.json")
	writeWatchManifest(t, path, validWatchManifest)
	w, err := newManifestWatch(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	pollManifest(t, w, now, false, false)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	writeWatchManifest(t, path, strings.Replace(validWatchManifest, `"Run"`, `"New"`, 1))
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	pollManifest(t, w, now.Add(time.Second), false, false)
	pollManifest(t, w, now.Add(1200*time.Millisecond), false, false)
	pollManifest(t, w, now.Add(1500*time.Millisecond), true, false)
	pollManifest(t, w, now.Add(2*time.Second), false, false)
	// Reverting a burst to the last reported content cancels the notice.
	writeWatchManifest(t, path, validWatchManifest)
	pollManifest(t, w, now.Add(3*time.Second), false, false)
	writeWatchManifest(t, path, strings.Replace(validWatchManifest, `"Run"`, `"New"`, 1))
	pollManifest(t, w, now.Add(4*time.Second), false, false)
	pollManifest(t, w, now.Add(5*time.Second), false, false)
}

func TestManifestWatchInvalidThenValid(t *testing.T) {
	for _, invalid := range []string{`{`, strings.Replace(validWatchManifest, `"schemaVersion":1`, `"schemaVersion":9`, 1)} {
		t.Run(invalid, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "velox.json")
			writeWatchManifest(t, path, validWatchManifest)
			w, err := newManifestWatch(path)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now()
			writeWatchManifest(t, path, invalid)
			pollManifest(t, w, now, false, false)
			pollManifest(t, w, now.Add(time.Second), false, true)
			pollManifest(t, w, now.Add(2*time.Second), false, false)
			writeWatchManifest(t, path, validWatchManifest)
			pollManifest(t, w, now.Add(3*time.Second), false, false)
			pollManifest(t, w, now.Add(4*time.Second), true, false)
		})
	}
}

func TestManifestWatchMissingOversizedAndLinkedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "velox.json")
	writeWatchManifest(t, path, validWatchManifest)
	w, err := newManifestWatch(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	pollManifest(t, w, now, false, true)
	writeWatchManifest(t, path, strings.Repeat(" ", maxManifestBytes+1))
	pollManifest(t, w, now.Add(time.Second), false, true)
	writeWatchManifest(t, path, validWatchManifest+"\n")
	pollManifest(t, w, now.Add(2*time.Second), false, false)
	pollManifest(t, w, now.Add(3*time.Second), true, false)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	writeWatchManifest(t, outside, validWatchManifest)
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	pollManifest(t, w, now.Add(4*time.Second), false, true)
}

type watchOutput struct {
	events chan string
}

func (w *watchOutput) Write(data []byte) (int, error) {
	w.events <- string(data)
	return len(data), nil
}

func awaitWatchOutput(t *testing.T, output *watchOutput, want string) {
	t.Helper()
	select {
	case got := <-output.events:
		if !strings.Contains(got, want) {
			t.Fatalf("output=%q want substring=%q", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %q", want)
	}
}

func TestExecuteWatchNoticesDoNotRestartOrChangeRuntimeConfig(t *testing.T) {
	plan := runnerPlan(t)
	output := &watchOutput{events: make(chan string, 16)}
	launches := 0
	result, err := Execute(plan, Options{Watch: true}, func(_, configPath string, _ Options, _, stderr io.Writer) (int, error) {
		launches++
		before, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		writeWatchManifest(t, plan.Snapshot().Manifest.ConfigPath, `{`)
		awaitWatchOutput(t, output, "manifest error:")
		writeWatchManifest(t, plan.Snapshot().Manifest.ConfigPath, strings.Replace(validWatchManifest, `"Run"`, `"Edited"`, 1))
		awaitWatchOutput(t, output, "restart required")
		_, _ = io.WriteString(stderr, "host still running\n")
		awaitWatchOutput(t, output, "host still running")
		after, err := os.ReadFile(configPath)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("runtime config changed: err=%v", err)
		}
		return 5, nil
	}, io.Discard, output)
	if result.ExitCode != 5 || err == nil || launches != 1 {
		t.Fatalf("result=%+v err=%v launches=%d", result, err, launches)
	}
	// Execute joins the watcher before returning, even on a nonzero host exit.
	if len(output.events) != 0 {
		t.Fatal("unexpected additional watch output")
	}
}

func TestCancelledManifestWatchDoesNotReport(t *testing.T) {
	w := &manifestWatch{path: filepath.Join(t.TempDir(), "missing.json")}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var output bytes.Buffer
	w.run(ctx, &output, os.ErrNotExist)
	if output.Len() != 0 {
		t.Fatalf("output after cancellation: %q", output.String())
	}
}
