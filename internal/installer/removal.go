package installer

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
)

func removeInstallationDirectories(record Record) error {
	return finishDirectoryRemoval(record, func(path string) error { return removeInstallationEntry(record, path) })
}

func removeInstallationEntry(record Record, path string) error {
	if path == filepath.Join(record.Directory, stateFile) {
		return removeOwnedRecord(record, path)
	}
	return os.Remove(path)
}

func finishDirectoryRemoval(record Record, remove func(string) error) error {
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
		if err := remove(directory); err != nil {
			return err
		}
	}
	stored, err := readState(root, record.App.ID)
	if errors.Is(err, os.ErrNotExist) {
		stored, err = readRemovalJournal(root, record.App.ID)
	}
	if err != nil {
		return err
	}
	if !sameOwnership(stored, record) {
		return errors.New("preserving installation ownership changed during removal")
	}
	if err := prepareRemovalJournal(record); err != nil {
		return err
	}
	if err := remove(filepath.Join(root, stateFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := remove(root); err != nil {
		return errors.Join(err, restoreState(root, record))
	}
	return removeRemovalJournal(record)
}
