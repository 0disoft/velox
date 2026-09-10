package hygiene_test

import (
	"path/filepath"
	"testing"
)

func TestProductReadinessDoesNotConfuseMocksWithNativeEvidence(t *testing.T) {
	root := repositoryRoot(t)
	assertSourceMarkers(t, filepath.Join(root, "docs", "adr", "0019-gate-beta-on-product-workflows.md"), []string{
		"Status: Accepted",
		"Supersedes in part: ADR 0018",
		"AI evaluation is optional diagnostic evidence",
		"No automatic beta or stable promotion",
		"Human adoption is still",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "ops", "product-readiness.md"), []string{
		"Alpha active; beta not approved",
		"Public consumer path",
		"File Notes behavior",
		"Development loop",
		"Windows lifecycle",
		"Security and data integrity",
		"real picker and persistence checks remain unverified",
		"A failed or",
		"unverified required check keeps beta held",
		"AI trials and external user feedback",
	})
}
