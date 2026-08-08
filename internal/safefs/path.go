package safefs

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ValidateRelativePath applies the portable Windows-safe path policy used by
// source assets and packaged artifacts.
func ValidateRelativePath(value string) error {
	normalized := filepath.ToSlash(value)
	if normalized == "" || path.IsAbs(normalized) || filepath.IsAbs(value) || filepath.VolumeName(value) != "" {
		return errors.New("path must be relative")
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == "" || part == "." || part == ".." {
			return errors.New("empty or traversal segment")
		}
		if strings.Contains(part, ":") {
			return errors.New("alternate data streams are not allowed")
		}
		trimmed := strings.TrimRight(part, ". ")
		if trimmed != part {
			return errors.New("trailing dots or spaces are not allowed")
		}
		base := strings.ToUpper(strings.SplitN(trimmed, ".", 2)[0])
		if isReservedName(base) {
			return fmt.Errorf("reserved Windows name %q", part)
		}
	}
	return nil
}

func ValidateArchiveEntry(name string) error {
	if strings.Contains(name, "\\") || path.Clean(name) != name {
		return errors.New("archive entry is not canonical")
	}
	return ValidateRelativePath(name)
}

// RejectLinkedComponents inspects every existing component without following
// its final component. Missing suffixes are allowed for create operations.
func RejectLinkedComponents(value string) error {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	volume := filepath.VolumeName(absolute)
	root := volume + string(filepath.Separator)
	remainder := strings.TrimPrefix(absolute, root)
	current := root
	for _, part := range strings.Split(remainder, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect path component %s: %w", current, err)
		}
		linked, err := IsLinkOrReparse(current, info)
		if err != nil {
			return err
		}
		if linked {
			return fmt.Errorf("path component is a link or reparse point: %s", current)
		}
	}
	return nil
}

func EnsureDirectory(value string, mode os.FileMode) error {
	if err := RejectLinkedComponents(value); err != nil {
		return err
	}
	if err := os.MkdirAll(value, mode); err != nil {
		return err
	}
	return RejectLinkedComponents(value)
}

func OpenVerifiedRegular(value string) (*os.File, os.FileInfo, error) {
	if err := RejectLinkedComponents(value); err != nil {
		return nil, nil, err
	}
	linkInfo, err := os.Lstat(value)
	if err != nil {
		return nil, nil, err
	}
	linked, err := IsLinkOrReparse(value, linkInfo)
	if err != nil {
		return nil, nil, err
	}
	if linked || !linkInfo.Mode().IsRegular() {
		return nil, nil, errors.New("file must be a regular file, not a link or reparse point")
	}
	file, err := os.Open(value)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(linkInfo, info) {
		file.Close()
		return nil, nil, errors.New("file changed while opening")
	}
	return file, info, nil
}

func isReservedName(name string) bool {
	if name == "CON" || name == "PRN" || name == "AUX" || name == "NUL" {
		return true
	}
	for index := 1; index <= 9; index++ {
		if name == fmt.Sprintf("COM%d", index) || name == fmt.Sprintf("LPT%d", index) {
			return true
		}
	}
	return false
}
