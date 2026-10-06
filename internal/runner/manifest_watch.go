package runner

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/0disoft/velox/internal/devwatch"
	"github.com/0disoft/velox/internal/manifest"
	"github.com/0disoft/velox/internal/safefs"
)

const maxManifestBytes = 1 << 20

type manifestWatch struct {
	path              string
	applied, observed [32]byte
	stableSince       time.Time
}

func newManifestWatch(path string) (*manifestWatch, error) {
	w := &manifestWatch{path: path}
	data, err := w.read()
	if err == nil {
		w.applied = sha256.Sum256(data)
		w.observed = w.applied
		_, err = manifest.Parse(path, data)
	}
	return w, err
}

func (w *manifestWatch) read() ([]byte, error) {
	file, before, err := safefs.OpenVerifiedRegular(w.path)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	after, statErr := file.Stat()
	closeErr := file.Close()
	if err := errors.Join(readErr, statErr, closeErr); err != nil {
		return nil, err
	}
	if len(data) > maxManifestBytes {
		return nil, fmt.Errorf("manifest exceeds watch limit of %d bytes", maxManifestBytes)
	}
	if int64(len(data)) != before.Size() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, errors.New("manifest changed during read")
	}
	return data, nil
}

func (w *manifestWatch) poll(now time.Time) (bool, error) {
	data, err := w.read()
	if err != nil {
		w.stableSince = time.Time{}
		return false, err
	}
	digest := sha256.Sum256(data)
	if digest != w.observed || w.stableSince.IsZero() {
		w.observed, w.stableSince = digest, now
		return false, nil
	}
	if digest == w.applied || now.Sub(w.stableSince) < devwatch.QuietPeriod {
		return false, nil
	}
	w.applied = digest
	if _, err := manifest.Parse(w.path, data); err != nil {
		return false, err
	}
	return true, nil
}

func (w *manifestWatch) run(ctx context.Context, stderr io.Writer, initialErr error) {
	ticker := time.NewTicker(devwatch.Interval)
	defer ticker.Stop()
	var lastError string
	report := func(err error) {
		if err.Error() != lastError {
			fmt.Fprintf(stderr, "velox: watch: manifest error: %v; running app unchanged, retrying\n", err)
		}
		lastError = err.Error()
	}
	if initialErr != nil && ctx.Err() == nil {
		report(initialErr)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			changed, err := w.poll(now)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				report(err)
				continue
			}
			lastError = ""
			if changed {
				fmt.Fprintln(stderr, "velox: watch: manifest changed; restart required to apply settings (running app unchanged)")
			}
		}
	}
}

// Host diagnostics and manifest notices share stderr during a watched run.
type synchronizedWriter struct {
	mu sync.Mutex
	io.Writer
}

func (w *synchronizedWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Writer.Write(data)
}
