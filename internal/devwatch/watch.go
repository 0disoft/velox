// Package devwatch detects stable source edits for opt-in development runs.
package devwatch

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/0disoft/velox/internal/assettree"
	"github.com/0disoft/velox/internal/safefs"
)

const Interval = 500 * time.Millisecond
const QuietPeriod = 500 * time.Millisecond
const maxTextBytes = 64 << 20
const maxFiles = 10000

type Watcher struct {
	root, entry       string
	applied, observed [32]byte
	stableSince       time.Time
}

func New(root, entry string) (*Watcher, error) {
	w := &Watcher{root: root, entry: entry}
	digest, err := w.snapshot()
	if err != nil {
		return nil, err
	}
	w.applied, w.observed = digest, digest
	return w, nil
}

// Poll emits once after a quiet period. Reverting an edit cancels the pending
// notification; unreadable or unsafe intermediate trees never trigger reload.
func (w *Watcher) Poll(now time.Time) (bool, error) {
	digest, err := w.snapshot()
	if err != nil {
		w.stableSince = time.Time{}
		return false, err
	}
	if digest != w.observed || w.stableSince.IsZero() {
		w.observed, w.stableSince = digest, now
		return false, nil
	}
	if digest == w.applied || now.Sub(w.stableSince) < QuietPeriod {
		return false, nil
	}
	w.applied = digest
	return true, nil
}

// Run owns no background work after it returns. The caller chooses whether to
// start it and must cancel its context when the development window closes.
func (w *Watcher) Run(ctx context.Context, changed func(), failed func(error)) {
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()
	var lastError string
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			reload, err := w.Poll(now)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				if err.Error() != lastError && failed != nil {
					failed(err)
				}
				lastError = err.Error()
				continue
			}
			lastError = ""
			if reload {
				changed()
			}
		}
	}
}

func (w *Watcher) snapshot() ([32]byte, error) {
	var result [32]byte
	if err := assettree.ValidateResolvedEntry(w.root, w.entry); err != nil {
		return result, err
	}
	tree, err := assettree.ScanMetadata(w.root)
	if err != nil {
		return result, err
	}
	if len(tree.Files) > maxFiles {
		return result, fmt.Errorf("watch asset count exceeds %d", maxFiles)
	}
	hash := sha256.New()
	var total int64
	for _, asset := range tree.Files {
		ext := strings.ToLower(filepath.Ext(asset.RelativePath))
		if asset.SourcePath != w.entry && ext != ".html" && ext != ".htm" && ext != ".css" && ext != ".js" && ext != ".mjs" {
			continue
		}
		file, before, err := safefs.OpenVerifiedRegular(asset.SourcePath)
		if err != nil {
			return result, err
		}
		fmt.Fprintf(hash, "%s\x00%d\x00", asset.RelativePath, before.Size())
		n, readErr := io.Copy(hash, io.LimitReader(file, maxTextBytes-total+1))
		after, statErr := file.Stat()
		closeErr := file.Close()
		total += n
		if readErr != nil || statErr != nil || closeErr != nil {
			return result, fmt.Errorf("read watch asset %s: %v / %v / %v", asset.RelativePath, readErr, statErr, closeErr)
		}
		if total > maxTextBytes {
			return result, fmt.Errorf("watched text exceeds %d bytes", maxTextBytes)
		}
		if n != before.Size() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
			return result, fmt.Errorf("watch asset changed during read: %s", asset.RelativePath)
		}
	}
	copy(result[:], hash.Sum(nil))
	return result, nil
}
