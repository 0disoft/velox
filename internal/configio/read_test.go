package configio

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/artifactlimits"
)

type growingReader struct{ bytes int }

func (reader *growingReader) Read(data []byte) (int, error) {
	clear(data)
	reader.bytes += len(data)
	return len(data), nil
}

func TestReadBoundedStopsGrowingInputAtLimitPlusOne(t *testing.T) {
	reader := &growingReader{}
	data, err := readBounded(reader)
	if !errors.Is(err, ErrTooLarge) || data != nil || reader.bytes != artifactlimits.MaxMetadataBytes+1 {
		t.Fatalf("readBounded() = %d bytes, %v; consumed %d", len(data), err, reader.bytes)
	}
}

type failingReader struct{ err error }

func (reader failingReader) Read([]byte) (int, error) { return 0, reader.err }

func TestReadBoundedPreservesIOError(t *testing.T) {
	cause := io.ErrUnexpectedEOF
	if _, err := readBounded(failingReader{cause}); !errors.Is(err, cause) {
		t.Fatalf("readBounded() = %v, want original cause", err)
	}
}

func TestReadFilePreservesMissingPathError(t *testing.T) {
	if _, err := ReadFile(filepath.Join(t.TempDir(), "missing.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadFile() = %v, want missing-file cause", err)
	}
}
