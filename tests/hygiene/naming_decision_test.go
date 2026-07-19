package hygiene_test

import (
	"strings"
	"testing"
)

func TestVeloxPublicIdentityDecisionIsComplete(t *testing.T) {
	review := readNormalized(t, repositoryPath("docs", "product", "05-naming-review.md"))
	for _, required := range []string{
		"Selected public name: Velox",
		"Selected CLI spelling: `velox`",
		"The public command is `velox`",
		"`0.5.10-alpha.1`",
		"not a legal or",
		"trademark opinion",
		"project maintainer never approved",
		"changing the product away from Velox",
	} {
		if !strings.Contains(review, required) {
			t.Errorf("naming review lacks %q", required)
		}
	}

	decision := readNormalized(t, repositoryPath("docs", "adr", "0015-retain-velox-public-identity.md"))
	for _, required := range []string{
		"Status: Accepted",
		"product: `Velox`",
		"CLI and executable: `velox` and `velox.exe`",
		"Go module and repository: `github.com/0disoft/velox`",
		"JavaScript bridge: `window.velox`",
		"environment prefix: `VELOX_`",
		"first unpublished candidate remains `0.5.10-alpha.1`",
		"do not by themselves block the first",
		"developer preview",
	} {
		if !strings.Contains(decision, required) {
			t.Errorf("ADR 0015 lacks %q", required)
		}
	}

	rejected := readNormalized(t, repositoryPath("docs", "adr", "0014-adopt-actutum-public-identity.md"))
	if !strings.Contains(rejected, "Status: Superseded by ADR 0015") {
		t.Fatal("ADR 0014 is not marked as superseded")
	}

	index := readNormalized(t, repositoryPath("docs", "adr", "README.md"))
	if !strings.Contains(index, "| 0013 | Superseded by 0015 |") ||
		!strings.Contains(index, "| 0014 | Superseded by 0015 |") ||
		!strings.Contains(index, "| 0015 | Accepted |") {
		t.Fatal("ADR index does not bind the public-name supersession")
	}
}
