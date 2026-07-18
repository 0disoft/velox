package hygiene_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPublicPreviewWorkflowUsesOnlyPublishedAssetsAndActutum(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "public-preview-verification.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, forbidden := range []string{
		"actions/checkout@",
		"actions/setup-go@",
		"actions/cache@",
		"go build",
		"go run",
		"go test",
		"node ",
		"npm ",
		"bun ",
		"contents: write",
		"gh release",
	} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("public-preview verification contains forbidden surface %q", forbidden)
		}
	}
	for _, required := range []string{
		"release_tag:",
		"expected_release_sha256:",
		"contents: read",
		"runs-on: windows-2025",
		"https://github.com/$repository/releases/download/$tag",
		"Invoke-WebRequest",
		"github-release-public-url-no-checkout",
		"The public tag, release manifest version, and target do not agree.",
		"Invoke-ActutumJson -Arguments @('version', '--json')",
		"Invoke-ActutumJson -Arguments @('build'",
		"Invoke-ActutumJson -Arguments @('inspect'",
		"Invoke-ActutumJson -Arguments @('run'",
		"ACTUTUM_BENCH_EXIT_AFTER_READY",
		"schema/public-preview-verification-v1.schema.json",
		"evidenceLevel = 'same-repository-public-download'",
		"externalUserAttempt = $false",
		"retention-days: 30",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("public-preview verification lacks %q", required)
		}
	}

	usesPattern := regexp.MustCompile(`(?m)^\s*uses:\s+([^\s#]+)`)
	shaPattern := regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+@[0-9a-f]{40}$`)
	matches := usesPattern.FindAllStringSubmatch(workflow, -1)
	if len(matches) != 1 {
		t.Fatalf("public-preview verification action count = %d", len(matches))
	}
	if !shaPattern.MatchString(matches[0][1]) {
		t.Fatalf("public-preview verification action is not SHA-pinned: %s", matches[0][1])
	}
}

func TestPublicPreviewResultSchemaKeepsSameRepositoryEvidenceNonExternal(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "schema", "public-preview-verification-v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"releaseTag", "releaseVersion", "expectedReleaseSha256", "observedReleaseSha256", "startupReady", "externalUserAttempt"} {
		if !containsString(schema.Required, field) {
			t.Errorf("public-preview result schema does not require %s", field)
		}
	}
	var external struct {
		Const bool `json:"const"`
	}
	if err := json.Unmarshal(schema.Properties["externalUserAttempt"], &external); err != nil {
		t.Fatal(err)
	}
	if external.Const {
		t.Fatal("same-repository public verification must not claim an external-user attempt")
	}
}

func TestExternalAttemptIssueContractRequiresIdentityAndSafeEvidence(t *testing.T) {
	issue := readNormalized(t, repositoryPath(".github", "ISSUE_TEMPLATE", "external-user-attempt.yml"))
	doc := readNormalized(t, repositoryPath("docs", "ops", "external-user-attempt.md"))
	for _, required := range []string{
		"Release ZIP SHA-256",
		"Windows and WebView2 versions",
		"SmartScreen or managed-device result",
		"Was a compiler, Node.js, or package-manager command required?",
		"I removed tokens, private paths, proprietary assets, and personal data",
	} {
		if !strings.Contains(issue, required) {
			t.Errorf("external-attempt issue form lacks %q", required)
		}
	}
	for _, required := range []string{
		"externalUserAttempt: false",
		"An attempt can fail and still qualify",
		"account, or repository that is not controlled",
		"Do not paste local absolute paths",
	} {
		if !strings.Contains(doc, required) {
			t.Errorf("external-attempt contract lacks %q", required)
		}
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
