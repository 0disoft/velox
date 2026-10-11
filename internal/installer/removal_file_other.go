//go:build !windows

package installer

import (
	"errors"
	"os"

	"github.com/0disoft/velox/internal/safefs"
)

// Non-Windows hosts only exercise the portable installer engine in tests;
// public Install/Uninstall are unsupported. Path removal is not atomic here.
type removalFile struct {
	file *os.File
	info os.FileInfo
	path string
}

func openRemovalFile(path string, owned File) (*removalFile, error) {
	file, info, err := safefs.OpenVerifiedRegular(path)
	if err != nil {
		return nil, err
	}
	if err := verifyRemovalFile(file, owned); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return &removalFile{file: file, info: info, path: path}, nil
}

func (file *removalFile) remove() error {
	current, err := os.Lstat(file.path)
	if err != nil {
		return err
	}
	if !os.SameFile(file.info, current) {
		return errors.New("preserving installed file replaced during removal")
	}
	return os.Remove(file.path)
}

func (file *removalFile) close() error { return file.file.Close() }
