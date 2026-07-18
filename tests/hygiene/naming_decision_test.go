package hygiene_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActutumPublicIdentityDecisionIsComplete(t *testing.T) {
	review := readNormalized(t, repositoryPath("docs", "product", "05-naming-review.md"))
	for _, required := range []string{
		"Selected public name: Actutum",
		"Selected CLI spelling: `actutum`",
		"The public command is `actutum`",
		"`0.6.0-alpha.1`",
		"6df8d7ad2a81c9432dc53b2c16cd0c94a227ab718d8d3e8648c269c42fb315b7",
		"not a GitHub Release",
		"not a legal opinion",
		"GitHub repository-name and user-name search returned no exact `actutum`",
	} {
		if !strings.Contains(review, required) {
			t.Errorf("naming review lacks %q", required)
		}
	}

	decision := readNormalized(t, repositoryPath("docs", "adr", "0014-adopt-actutum-public-identity.md"))
	for _, required := range []string{
		"Status: Accepted",
		"product: `Actutum`",
		"CLI and executable: `actutum` and `actutum.exe`",
		"Go module and intended repository: `github.com/0disoft/actutum`",
		"JavaScript bridge: `window.actutum`",
		"environment prefix: `ACTUTUM_`",
		"Actutum candidate is",
		"`0.6.0-alpha.1`",
		"Do not provide aliases",
	} {
		if !strings.Contains(decision, required) {
			t.Errorf("ADR 0014 lacks %q", required)
		}
	}

	index := readNormalized(t, repositoryPath("docs", "adr", "README.md"))
	if !strings.Contains(index, "| 0013 | Superseded by 0014 |") ||
		!strings.Contains(index, "| 0014 | Accepted |") {
		t.Fatal("ADR index does not bind the public-name supersession")
	}
}

func TestActutumTechnicalIdentityHasNoWorkingNameAliases(t *testing.T) {
	root := repositoryPath()
	paths := []string{
		".github",
		".signpath",
		"cmd",
		"diagrams",
		"examples",
		"internal",
		"schema",
		"scripts",
		filepath.Join("tests", "fixtures"),
		filepath.Join("tests", "startup"),
		filepath.Join("third_party", "go-webview2"),
		".gitignore",
		"ARCHITECTURE.md",
		"CONTRIBUTING.md",
		"DEVELOPMENT.md",
		"LICENSE",
		"PRIVACY.md",
		"SECURITY.md",
		"THIRD_PARTY_NOTICES.md",
		"go.mod",
	}
	oldTokens := []string{
		"github.com/0disoft/velox",
		"velox.exe",
		"velox-host",
		"velox.json",
		"velox.runtime.json",
		"window.velox",
		"VELOX_",
		"schemas.velox.invalid",
		"velox.",
		"0.5.10-alpha.1",
	}

	for _, relative := range paths {
		path := filepath.Join(root, relative)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			assertNoWorkingNameTokens(t, root, path, oldTokens)
			continue
		}
		if err := filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			assertNoWorkingNameTokens(t, root, current, oldTokens)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	for _, oldPath := range []string{
		filepath.Join(root, "cmd", "velox"),
		filepath.Join(root, "schema", "velox-v1.schema.json"),
		filepath.Join(root, "examples", "hello", "velox.json"),
		filepath.Join(root, ".signpath", "policies", "velox"),
	} {
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Errorf("working-name path still exists: %s", oldPath)
		}
	}
}

func assertNoWorkingNameTokens(t *testing.T, root, path string, oldTokens []string) {
	t.Helper()
	relative, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(relative), "velox") {
		t.Errorf("working-name path remains in technical surface: %s", relative)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.IndexByte(string(data), 0) >= 0 {
		return
	}
	text := string(data)
	for _, token := range oldTokens {
		if strings.Contains(text, token) {
			t.Errorf("%s retains working-name token %q", relative, token)
		}
	}
}
