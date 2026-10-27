// Package configio bounds on-disk JSON configuration reads before decoding.
package configio

import (
	"fmt"
	"io"
	"os"

	"github.com/0disoft/velox/internal/artifactlimits"
)

var ErrTooLarge = fmt.Errorf("metadata exceeds %d-byte limit", artifactlimits.MaxMetadataBytes)

// ReadFile preserves ordinary file-opening behavior while limiting content
// reads even if the file grows after Stat. The caller owns JSON validation.
func ReadFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > artifactlimits.MaxMetadataBytes {
		return nil, ErrTooLarge
	}
	return readBounded(file)
}

func readBounded(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, artifactlimits.MaxMetadataBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > artifactlimits.MaxMetadataBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}
