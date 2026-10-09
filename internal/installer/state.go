package installer

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/0disoft/velox/internal/safefs"
)

func writeState(root string, record Record) error {
	path := filepath.Join(root, stateFile)
	if err := safefs.RejectLinkedComponents(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(root, ".velox-state-")
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
	return replaceStateFile(temp.Name(), path)
}
