package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"

	"github.com/0disoft/velox/internal/safefs"
)

func removeOwnedRecord(record Record, path string) error {
	snapshot, err := ownedRecordSnapshot(record, path)
	if err != nil {
		return err
	}
	return removeRecordedFile(path, snapshot)
}

func ownedRecordSnapshot(expected Record, path string) (File, error) {
	file, info, err := safefs.OpenVerifiedRegular(path)
	if err != nil {
		return File{}, err
	}
	defer file.Close()
	if info.Size() > 16<<20 {
		return File{}, errors.New("preserving oversized installation ownership record")
	}
	hash := sha256.New()
	stored, err := decodeRecord(io.TeeReader(io.LimitReader(file, info.Size()+1), hash), expected.Directory, expected.App.ID)
	if err != nil {
		return File{}, err
	}
	if !sameOwnership(stored, expected) {
		return File{}, errors.New("preserving installation ownership changed before deletion")
	}
	return File{Path: filepath.Base(path), Bytes: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}
