package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func removalJournalPath(directory, appID string) string {
	return filepath.Join(filepath.Dir(directory), ".velox-removal-"+appID+".json")
}

func readRemovalJournal(directory, appID string) (Record, error) {
	return readRecordAt(removalJournalPath(directory, appID), directory, appID)
}

func validateRemovalJournal(record Record) error {
	stored, err := readRemovalJournal(record.Directory, record.App.ID)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("preserving unreadable removal recovery record: %w", err)
	}
	if !sameOwnership(stored, record) {
		return errors.New("preserving conflicting removal recovery record")
	}
	return nil
}

func prepareRemovalJournal(record Record) error {
	path := removalJournalPath(record.Directory, record.App.ID)
	if err := publishStateAt(path, record, os.Link); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		return validateRemovalJournal(record)
	}
	return nil
}

func removeRemovalJournal(record Record) error {
	if err := removeOwnedRecord(record, removalJournalPath(record.Directory, record.App.ID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
