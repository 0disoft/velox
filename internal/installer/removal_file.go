package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

func verifyRemovalFile(file *os.File, owned File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != owned.Bytes || owned.Bytes > 512<<20 {
		return fmt.Errorf("preserving changed installed file %q; restore it or move it before retrying removal", owned.Path)
	}
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(file, owned.Bytes+1))
	if err != nil {
		return err
	}
	if size != owned.Bytes || hex.EncodeToString(hash.Sum(nil)) != owned.SHA256 {
		return fmt.Errorf("preserving changed installed file %q; restore it or move it before retrying removal", owned.Path)
	}
	return nil
}

func removeRecordedFile(path string, owned File) error {
	file, err := openRemovalFile(path, owned)
	if err != nil {
		return fmt.Errorf("preserving installed file %q: %w", owned.Path, err)
	}
	return errors.Join(file.remove(), file.close())
}
