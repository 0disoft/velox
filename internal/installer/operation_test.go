package installer

import (
	"errors"
	"os"
	"testing"
)

func TestUninstallPreservesInstallationDuringOverlappingRemovalAndReinstall(t *testing.T) {
	source, setup, env := fixture(t)
	installed, err := install(source, setup, env)
	if err != nil {
		t.Fatal(err)
	}
	var nestedErr error
	overlapping := env
	// Pause A after reading ownership. Run B's complete removal/reinstall,
	// then let A resume with its stale, byte-identical ownership record.
	overlapping.validateRemoval = func(record Record) error {
		_, nestedErr = uninstall(record.App.ID, env)
		if nestedErr != nil {
			return nestedErr
		}
		_, nestedErr = install(source, setup, env)
		return nestedErr
	}
	_, err = uninstall(installed.AppID, overlapping)
	if !errors.Is(nestedErr, errOperationInProgress) || !errors.Is(err, errOperationInProgress) {
		t.Errorf("overlapping removal/reinstall admitted: nested=%v, outer=%v", nestedErr, err)
	}
	if _, err := os.Stat(installed.Directory); err != nil {
		t.Fatalf("resumed stale uninstall removed the installation: %v", err)
	}
}

func TestInstallHoldsOperationThroughChecksAndReleasesOnError(t *testing.T) {
	source, setup, env := fixture(t)
	other := env
	cause := errors.New("injected environment failure")
	env.check = func(record Record) error {
		if _, err := uninstall(record.App.ID, other); !errors.Is(err, errOperationInProgress) {
			t.Errorf("removal entered during installation: %v", err)
		}
		return cause
	}
	if _, err := install(source, setup, env); !errors.Is(err, cause) {
		t.Fatalf("install() = %v, want original environment failure", err)
	}
	release, err := acquireOperation(env.root, "dev.velox.test")
	if err != nil {
		t.Fatalf("failed installation leaked its operation lease: %v", err)
	}
	release()
}
