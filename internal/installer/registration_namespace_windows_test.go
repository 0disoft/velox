package installer

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"
)

// TempDir ends in a counter such as 001; only its full path identifies a run.
// Derivation stays deterministic so interruption helpers use their parent's key.
func testRegistrationNamespace(root string) string {
	digest := sha256.Sum256([]byte(filepath.Clean(root)))
	return fmt.Sprintf(`Software\VeloxInstallerTests\%x`, digest[:16])
}

func TestRegistryNamespaceIsolatesMatchingTempBasenames(t *testing.T) {
	parent := t.TempDir()
	first := filepath.Join(parent, "first-run", "001")
	second := filepath.Join(parent, "second-run", "001")
	if testRegistrationNamespace(first) == testRegistrationNamespace(second) {
		t.Fatal("distinct temporary roots share a registry namespace")
	}
	if testRegistrationNamespace(first) != testRegistrationNamespace(first+string(filepath.Separator)+".") {
		t.Fatal("helper and parent must derive the same namespace for the same root")
	}
}
