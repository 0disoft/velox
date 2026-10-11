package installer

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
)

func removeInstallationDirectories(record Record) error {
	root := record.Directory
	var directories []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && path != root {
			directories = append(directories, path)
		}
		return nil
	}); err != nil {
		return err
	}
	sort.Slice(directories, func(i, j int) bool { return len(directories[i]) > len(directories[j]) })
	for _, directory := range directories {
		if err := os.Remove(directory); err != nil {
			return err
		}
	}
	stored, err := readState(root, record.App.ID)
	if err != nil {
		return err
	}
	if !sameOwnership(stored, record) {
		return errors.New("preserving installation ownership changed during removal")
	}
	if err := os.Remove(filepath.Join(root, stateFile)); err != nil {
		return err
	}
	if err := os.Remove(root); err != nil {
		return errors.Join(err, restoreState(root, record))
	}
	return nil
}
