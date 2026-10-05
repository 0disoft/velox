package devwatch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Watcher, string, string) {
	t.Helper()
	root := t.TempDir()
	entry := filepath.Join(root, "index.html")
	if err := os.WriteFile(entry, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}
	w, err := New(root, entry)
	if err != nil {
		t.Fatal(err)
	}
	return w, root, entry
}

func poll(t *testing.T, w *Watcher, now time.Time, want bool) {
	t.Helper()
	got, err := w.Poll(now)
	if err != nil || got != want {
		t.Fatalf("reload=%t want=%t err=%v", got, want, err)
	}
}

func TestContentEditsDebounceAndDoNotRepeat(t *testing.T) {
	w, root, entry := fixture(t)
	now := time.Now()
	poll(t, w, now, false)
	info, _ := os.Stat(entry)
	if err := os.WriteFile(entry, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	// Same size and timestamp still has different content.
	if err := os.Chtimes(entry, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	poll(t, w, now.Add(time.Second), false)
	if err := os.WriteFile(filepath.Join(root, "app.js"), []byte("next"), 0644); err != nil {
		t.Fatal(err)
	}
	poll(t, w, now.Add(1200*time.Millisecond), false)
	poll(t, w, now.Add(1600*time.Millisecond), false)
	poll(t, w, now.Add(1800*time.Millisecond), true)
	poll(t, w, now.Add(3*time.Second), false)
	if err := os.Remove(filepath.Join(root, "app.js")); err != nil {
		t.Fatal(err)
	}
	poll(t, w, now.Add(4*time.Second), false)
	poll(t, w, now.Add(5*time.Second), true)
}

func TestRevertedTextEditsAndUnwatchedAssetsDoNotReload(t *testing.T) {
	w, root, entry := fixture(t)
	now := time.Now()
	if err := os.WriteFile(entry, []byte("edited"), 0644); err != nil {
		t.Fatal(err)
	}
	poll(t, w, now, false)
	if err := os.WriteFile(entry, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data.bin"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	poll(t, w, now.Add(time.Second), false)
	poll(t, w, now.Add(2*time.Second), false)
}

func TestImageAndFontMetadataEditsDebounce(t *testing.T) {
	for _, ext := range []string{
		".png", ".apng", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".bmp", ".ico", ".svg",
		".woff", ".woff2", ".ttf", ".otf", ".eot", ".PNG", ".WOFF2",
	} {
		t.Run(ext, func(t *testing.T) {
			w, root, _ := fixture(t)
			now := time.Now()
			path := filepath.Join(root, "asset"+ext)
			if err := os.WriteFile(path, []byte("before"), 0644); err != nil {
				t.Fatal(err)
			}
			poll(t, w, now, false)
			poll(t, w, now.Add(time.Second), true)
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("edited"), 0644); err != nil {
				t.Fatal(err)
			}
			modified := info.ModTime().Add(time.Second)
			if err := os.Chtimes(path, modified, modified); err != nil {
				t.Fatal(err)
			}
			poll(t, w, now.Add(2*time.Second), false)
			poll(t, w, now.Add(2200*time.Millisecond), false)
			poll(t, w, now.Add(3*time.Second), true)
			poll(t, w, now.Add(4*time.Second), false)
			if err := os.Rename(path, filepath.Join(root, "renamed"+ext)); err != nil {
				t.Fatal(err)
			}
			poll(t, w, now.Add(5*time.Second), false)
			poll(t, w, now.Add(6*time.Second), true)
			if err := os.Remove(filepath.Join(root, "renamed"+ext)); err != nil {
				t.Fatal(err)
			}
			poll(t, w, now.Add(7*time.Second), false)
			poll(t, w, now.Add(8*time.Second), true)
		})
	}
}

func TestVisualAssetsUseMetadataNotTextReadBudget(t *testing.T) {
	w, root, _ := fixture(t)
	path := filepath.Join(root, "large.woff2")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	truncateErr := file.Truncate(maxTextBytes + 1)
	closeErr := file.Close()
	if truncateErr != nil || closeErr != nil {
		t.Fatalf("truncate=%v close=%v", truncateErr, closeErr)
	}
	now := time.Now()
	poll(t, w, now, false)
	poll(t, w, now.Add(time.Second), true)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err = os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteAt([]byte("edit"), 0)
	closeErr = file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("write=%v close=%v", writeErr, closeErr)
	}
	// Metadata-only detection deliberately cannot see a same-size/time edit.
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	poll(t, w, now.Add(2*time.Second), false)
	if err := os.Truncate(path, maxTextBytes+2); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	poll(t, w, now.Add(3*time.Second), false)
	poll(t, w, now.Add(4*time.Second), true)
}

func TestMissingEntryWaitsForStableRecovery(t *testing.T) {
	w, _, entry := fixture(t)
	now := time.Now()
	if err := os.Remove(entry); err != nil {
		t.Fatal(err)
	}
	if changed, err := w.Poll(now); changed || err == nil {
		t.Fatalf("changed=%t err=%v", changed, err)
	}
	if err := os.WriteFile(entry, []byte("recovered"), 0644); err != nil {
		t.Fatal(err)
	}
	poll(t, w, now.Add(time.Second), false)
	poll(t, w, now.Add(2*time.Second), true)
}

func TestLinkedTreeRejected(t *testing.T) {
	w, root, _ := fixture(t)
	outside := filepath.Join(t.TempDir(), "outside.js")
	if err := os.WriteFile(outside, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.js")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if changed, err := w.Poll(time.Now()); changed || err == nil {
		t.Fatalf("changed=%t err=%v", changed, err)
	}
}

func TestLinkedVisualAssetRejected(t *testing.T) {
	w, root, _ := fixture(t)
	outside := filepath.Join(t.TempDir(), "outside.woff2")
	if err := os.WriteFile(outside, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.woff2")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if changed, err := w.Poll(time.Now()); changed || err == nil {
		t.Fatalf("changed=%t err=%v", changed, err)
	}
}

func TestCancelledRunStopsWithoutCallbacks(t *testing.T) {
	w, _, _ := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.Run(ctx, func() { t.Fatal("changed after cancel") }, func(error) { t.Fatal("error after cancel") })
}
