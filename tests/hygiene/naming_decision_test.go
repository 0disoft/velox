package hygiene_test

import (
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
