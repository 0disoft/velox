package hygiene_test

import (
	"strings"
	"testing"
)

func TestUnsignedDeveloperPreviewPolicyKeepsSigningDeferred(t *testing.T) {
	checks := map[string][]string{
		"docs/adr/0011-publish-unsigned-developer-preview-before-signing.md": {
			"Publish the first public M4 artifact as an explicitly unsigned",
			"preview. Authenticode signing is not an M4",
			"publication gate",
			"SmartScreen can warn",
			"managed devices can block",
		},
		"docs/product/01-roadmap.md": {
			"Explicitly unsigned Velox CLI",
			"SignPath onboarding, Authenticode verification against real provider output",
			"deferred until a real adoption trigger",
		},
		"docs/ops/release.md": {
			"Hosted candidate evidence current; public preview pending",
			"ADR 0015 removes the replacement-name gate",
			"The isolated publication job",
			"alone receives `contents: write`",
			"not sign, attest, rebuild, or replace artifacts",
		},
		"docs/ops/signing.md": {
			"Dormant future-channel tooling",
			"Do not submit the provider application or add a signing workflow merely to",
		},
	}
	for relative, required := range checks {
		data := readNormalized(t, repositoryPath(strings.Split(relative, "/")...))
		for _, value := range required {
			if !strings.Contains(data, value) {
				t.Errorf("%s lacks unsigned-preview policy %q", relative, value)
			}
		}
	}
}

func TestM4RoadmapDoesNotRequireProviderAcceptance(t *testing.T) {
	roadmap := readNormalized(t, repositoryPath("docs", "product", "01-roadmap.md"))
	start := strings.Index(roadmap, "## M4: Alpha Distribution")
	end := strings.Index(roadmap, "## M5: Product Decision")
	if start < 0 || end <= start {
		t.Fatal("M4 roadmap section is missing")
	}
	m4 := roadmap[start:end]
	for _, forbidden := range []string{
		"Obtain SignPath Foundation project acceptance",
		"Signed Velox CLI and unchanged generic host",
		"provider-approved signing",
	} {
		if strings.Contains(m4, forbidden) {
			t.Errorf("M4 still contains signing prerequisite %q", forbidden)
		}
	}
}
