package installer

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/0disoft/velox/internal/safefs"
)

const maxShortcutBytes = 1 << 20

func snapshotWindowsShortcut(path, expectedHash string) (File, error) {
	file, info, err := safefs.OpenVerifiedRegular(path)
	if err != nil {
		return File{}, err
	}
	defer file.Close()
	if expectedHash == "" || info.Size() > maxShortcutBytes {
		return File{}, errors.New("Start Menu shortcut changed or oversized; move it before uninstalling")
	}
	snapshot := File{Path: filepath.Base(path), Bytes: info.Size(), SHA256: expectedHash}
	if err := verifyRemovalFile(file, snapshot); err != nil {
		return File{}, fmt.Errorf("Start Menu shortcut changed; move it before uninstalling: %w", err)
	}
	return snapshot, nil
}

func removeWindowsShortcut(path, expectedHash string) error {
	snapshot, err := snapshotWindowsShortcut(path, expectedHash)
	if err != nil {
		return err
	}
	return removeRecordedFile(path, snapshot)
}
