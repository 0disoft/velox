package outputpair

import (
	"errors"
	"fmt"
	"os"
)

// Recover reconciles process-interrupted directory/archive promotion states.
// It intentionally makes no power-loss durability claim.
func Recover(finalDirectory, finalArchive string) error {
	backupDirectory := finalDirectory + ".previous"
	backupArchive := finalArchive + ".previous"
	finalDirectoryExists, err := validate(finalDirectory, true)
	if err != nil {
		return fmt.Errorf("validate output directory: %w", err)
	}
	finalArchiveExists, err := validate(finalArchive, false)
	if err != nil {
		return fmt.Errorf("validate output archive: %w", err)
	}
	backupDirectoryExists, err := validate(backupDirectory, true)
	if err != nil {
		return fmt.Errorf("validate recovery directory: %w", err)
	}
	backupArchiveExists, err := validate(backupArchive, false)
	if err != nil {
		return fmt.Errorf("validate recovery archive: %w", err)
	}

	if finalDirectoryExists && finalArchiveExists {
		return cleanupBackups(backupDirectory, backupArchive, backupDirectoryExists, backupArchiveExists)
	}
	if !backupDirectoryExists && !backupArchiveExists {
		if finalDirectoryExists != finalArchiveExists {
			return errors.New("incomplete output pair has no recovery backup")
		}
		return nil
	}
	if backupDirectoryExists && !backupArchiveExists && !finalDirectoryExists && finalArchiveExists {
		if err := os.Rename(backupDirectory, finalDirectory); err != nil {
			return fmt.Errorf("restore interrupted directory backup: %w", err)
		}
		return nil
	}
	if backupDirectoryExists && backupArchiveExists {
		if err := removeFinals(finalDirectory, finalArchive, finalDirectoryExists, finalArchiveExists); err != nil {
			return err
		}
		if err := os.Rename(backupDirectory, finalDirectory); err != nil {
			return fmt.Errorf("restore recovery directory: %w", err)
		}
		if err := os.Rename(backupArchive, finalArchive); err != nil {
			return fmt.Errorf("restore recovery archive: %w", err)
		}
		return nil
	}
	return errors.New("ambiguous output recovery state; recovery data was retained")
}

func Promote(finalDirectory, finalArchive, stageDirectory, stageArchive string) error {
	if err := Recover(finalDirectory, finalArchive); err != nil {
		return err
	}
	finalDirectoryExists, _ := validate(finalDirectory, true)
	finalArchiveExists, _ := validate(finalArchive, false)
	if finalDirectoryExists != finalArchiveExists {
		return errors.New("output pair is incomplete after recovery")
	}
	backupDirectory := finalDirectory + ".previous"
	backupArchive := finalArchive + ".previous"

	if finalDirectoryExists {
		if err := os.Rename(finalDirectory, backupDirectory); err != nil {
			return fmt.Errorf("backup output directory: %w", err)
		}
		if err := os.Rename(finalArchive, backupArchive); err != nil {
			restoreErr := os.Rename(backupDirectory, finalDirectory)
			return errors.Join(fmt.Errorf("backup output archive: %w", err), wrap("restore output directory", restoreErr))
		}
	}
	rollback := func() error {
		var result error
		result = errors.Join(result, wrap("remove partially promoted directory", os.RemoveAll(finalDirectory)))
		if err := os.Remove(finalArchive); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, fmt.Errorf("remove partially promoted archive: %w", err))
		}
		if finalDirectoryExists {
			result = errors.Join(result, wrap("restore output directory", os.Rename(backupDirectory, finalDirectory)))
			result = errors.Join(result, wrap("restore output archive", os.Rename(backupArchive, finalArchive)))
		}
		return result
	}
	if err := os.Rename(stageDirectory, finalDirectory); err != nil {
		return errors.Join(fmt.Errorf("promote output directory: %w", err), rollback())
	}
	if err := os.Rename(stageArchive, finalArchive); err != nil {
		return errors.Join(fmt.Errorf("promote output archive: %w", err), rollback())
	}

	// Publication is complete. Leftover backups are recoverable garbage and must
	// not turn a successful build into a false failure.
	_ = cleanupBackups(backupDirectory, backupArchive, finalDirectoryExists, finalArchiveExists)
	return nil
}

func validate(path string, wantDirectory bool) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, errors.New("path must not be a symbolic link")
	}
	if wantDirectory && !info.IsDir() {
		return false, errors.New("path is not a directory")
	}
	if !wantDirectory && !info.Mode().IsRegular() {
		return false, errors.New("path is not a regular file")
	}
	return true, nil
}

func cleanupBackups(directory, archive string, hasDirectory, hasArchive bool) error {
	var result error
	if hasDirectory {
		result = errors.Join(result, wrap("remove recovery directory", os.RemoveAll(directory)))
	}
	if hasArchive {
		if err := os.Remove(archive); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, fmt.Errorf("remove recovery archive: %w", err))
		}
	}
	return result
}

func removeFinals(directory, archive string, hasDirectory, hasArchive bool) error {
	var result error
	if hasDirectory {
		result = errors.Join(result, wrap("remove partial output directory", os.RemoveAll(directory)))
	}
	if hasArchive {
		result = errors.Join(result, wrap("remove partial output archive", os.Remove(archive)))
	}
	return result
}

func wrap(operation string, err error) error {
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
