package installer

import (
	"errors"
	"fmt"
	"slices"
)

func rollbackInstallation(expected Record, env environment) error {
	rollback := env
	rollback.validateRemoval = func(stored Record) error {
		if stored.SchemaVersion != expected.SchemaVersion || stored.App != expected.App ||
			stored.Directory != expected.Directory || stored.ShortcutHash != expected.ShortcutHash ||
			!slices.Equal(stored.Files, expected.Files) {
			return errors.New("installation ownership changed during publication")
		}
		return nil
	}
	// Publication owns cleanup of its registry operation. Do not unregister a
	// foreign key or shortcut that appeared while that operation was failing.
	rollback.unregister = func(Record) error { return nil }
	if _, err := uninstall(expected.App.ID, rollback); err != nil {
		return fmt.Errorf("rollback preserved installation at %q; resolve this issue before uninstalling and retrying: %w", expected.Directory, err)
	}
	return nil
}
