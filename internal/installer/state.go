package installer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/0disoft/velox/internal/safefs"
)

func writeState(root string, record Record) error {
	return publishState(root, record, replaceStateFile)
}

func restoreState(root string, record Record) error {
	// Publishing a completed temporary file by link cannot overwrite a file
	// created by somebody else after removal of the original ownership record.
	return publishState(root, record, os.Link)
}

func sameOwnership(a, b Record) bool {
	return a.SchemaVersion == b.SchemaVersion && a.App == b.App &&
		a.Directory == b.Directory && a.ShortcutHash == b.ShortcutHash && slices.Equal(a.Files, b.Files)
}

func publishState(root string, record Record, publish func(string, string) error) error {
	return publishStateAt(filepath.Join(root, stateFile), record, publish)
}

func publishStateAt(path string, record Record, publish func(string, string) error) error {
	if err := safefs.RejectLinkedComponents(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".velox-state-")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(append(data, '\n')); err != nil {
		return errors.Join(err, temp.Close())
	}
	if err := temp.Sync(); err != nil {
		return errors.Join(err, temp.Close())
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return publish(temp.Name(), path)
}
