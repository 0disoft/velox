package hygiene_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestConsumerEvidenceWorkflowPinsActionsAndAvoidsCaches(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "consumer-evidence.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)

	if strings.Contains(workflow, "actions/cache@") {
		t.Fatal("consumer evidence workflow must not use actions/cache")
	}
	if !strings.Contains(workflow, "runs-on: windows-2025") {
		t.Fatal("consumer evidence workflow must pin the Windows runner label")
	}
	if !strings.Contains(workflow, "retention-days: 1") || !strings.Contains(workflow, "retention-days: 7") || !strings.Contains(workflow, "retention-days: 30") || !strings.Contains(workflow, "retention-days: 90") {
		t.Fatal("consumer evidence workflow must bound release and raw-result retention")
	}

	usesPattern := regexp.MustCompile(`(?m)^\s*uses:\s+([^\s#]+)`)
	shaPattern := regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+@[0-9a-f]{40}$`)
	matches := usesPattern.FindAllStringSubmatch(workflow, -1)
	if len(matches) == 0 {
		t.Fatal("consumer evidence workflow contains no actions")
	}
	for _, match := range matches {
		if !shaPattern.MatchString(match[1]) {
			t.Errorf("workflow action is not pinned to an immutable SHA: %s", match[1])
		}
	}
}

func TestConsumerEvidenceWorkflowKeepsConsumerBuildCompilerFree(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "consumer-evidence.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	consumerIndex := strings.Index(workflow, "\n  consumer:\n")
	if consumerIndex < 0 {
		t.Fatal("consumer job is missing")
	}
	consumer := workflow[consumerIndex:]
	for _, forbidden := range []string{"go build", "go run", "go test", "setup-go", "actions/cache"} {
		if strings.Contains(consumer, forbidden) {
			t.Errorf("consumer job contains forbidden toolchain surface %q", forbidden)
		}
	}
	if !strings.Contains(consumer, "scripts/measure-consumer-e2e.ps1") {
		t.Fatal("consumer job does not invoke the end-to-end measurement contract")
	}
	if !strings.Contains(consumer, "scripts/summarize-consumer-e2e.ps1") || !strings.Contains(consumer, "merge-multiple: true") {
		t.Fatal("consumer evidence workflow does not aggregate all raw sample artifacts")
	}
}

func TestConsumerEvidenceWorkflowOwnsLifecycleSummaryPolicy(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "consumer-evidence.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"schema/startup-lifecycle-v3.schema.json",
		"schema/startup-lifecycle-summary-v1.schema.json",
		"schema/startup-lifecycle-phase-summary-v1.schema.json",
		"go run ./cmd/velox-startup-summary",
		"--phase-output $phaseSummaryPath",
		"inputs.evidence_tier == 'full'",
		"&& '10' || '3'",
		"&& '10' || '1'",
		"type: choice",
		"- quick",
		"- full",
		"include_profile_comparison:",
		"TestStartupProfileComparisonEvidence$",
		"schema/startup-profile-comparison-v1.schema.json",
		"include_startup_history:",
		"go run ./cmd/velox-startup-history",
		"schema/startup-history-v1.schema.json",
		"--limit 12",
		"actions: read",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("consumer evidence workflow is missing lifecycle contract %q", required)
		}
	}
}

func TestDependabotChecksGitHubActionsWithoutAutoMerge(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "dependabot.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	config := string(data)
	for _, required := range []string{"package-ecosystem: github-actions", "interval: weekly", "timezone: Asia/Seoul"} {
		if !strings.Contains(config, required) {
			t.Errorf("Dependabot configuration is missing %q", required)
		}
	}
	if strings.Contains(strings.ToLower(config), "auto-merge") || strings.Contains(strings.ToLower(config), "automerge") {
		t.Fatal("Dependabot configuration must not enable auto-merge")
	}
}

func TestActionsWarningMonitorIsBoundedAndDiagnostic(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "actions-warning-monitor.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"workflow_run:",
		"- Consumer evidence",
		"actions: read",
		"contents: read",
		"github.event.workflow_run.event == 'schedule'",
		"github.event.workflow_run.event == 'push'",
		"runs-on: ubuntu-24.04",
		"go run ./cmd/velox-actions-warning-monitor",
		"schema/actions-warning-monitor-v1.schema.json",
		"known_warning_status=",
		"retention-days: 30",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("warning monitor workflow is missing %q", required)
		}
	}
	if strings.Contains(workflow, "actions/download-artifact@") {
		t.Fatal("warning monitor must fetch completed logs through the bounded GitHub API client")
	}
	if strings.Contains(workflow, "runs-on: windows-") {
		t.Fatal("platform-independent warning scanning must not consume a Windows runner")
	}
	if strings.Contains(workflow, "if ($document.status -eq 'present')") {
		t.Fatal("known upstream warning presence must remain diagnostic")
	}
}

func TestAlphaEvidenceWorkflowKeepsConsumerCheckoutAndToolchainFree(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "alpha-evidence.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	consumerIndex := strings.Index(workflow, "\n  clean-consumer:\n")
	if consumerIndex < 0 {
		t.Fatal("checkout-free consumer job is missing")
	}
	consumer := workflow[consumerIndex:]
	for _, forbidden := range []string{"actions/checkout@", "actions/setup-go@", "actions/cache@", "go build", "go run", "go test", "node ", "npm ", "bun "} {
		if strings.Contains(consumer, forbidden) {
			t.Errorf("checkout-free consumer job contains forbidden surface %q", forbidden)
		}
	}
	for _, required := range []string{"velox-release-evidence", "SPDX-2.3", "https://in-toto.io/Statement/v1", "Independent unsigned release builds are not byte-identical", "github-actions-artifact-no-checkout", "schema/consumer-clean-v1.schema.json", "retention-days: 7", "retention-days: 30"} {
		if !strings.Contains(workflow, required) {
			t.Errorf("alpha evidence workflow is missing %q", required)
		}
	}
	for _, required := range []string{"function Invoke-VeloxJson", "$firstBuild.result.archive", "$secondBuild.result.archive"} {
		if !strings.Contains(consumer, required) {
			t.Errorf("checkout-free consumer job does not use the CLI result contract %q", required)
		}
	}
	if strings.Contains(consumer, "Get-ChildItem -LiteralPath '.ci/first'") || strings.Contains(consumer, "Get-ChildItem -LiteralPath '.ci/second'") {
		t.Fatal("checkout-free consumer job must not guess build output directories")
	}
	for _, required := range []string{
		"publish_preview:",
		"publish unsigned developer preview",
		"github.repository == '0disoft/velox' && github.event_name == 'workflow_dispatch' && inputs.publish_preview",
		"github.ref_type",
		"^v[0-9]+[.][0-9]+[.][0-9]+-alpha[.][0-9]+$",
		"release_version=$($manifest.releaseVersion)",
		"v$env:VELOX_RELEASE_VERSION",
		"The selected tag does not match the release manifest version.",
		"name: Publish unsigned developer preview",
		"contents: write",
		"gh release create",
		"--prerelease",
		"--verify-tag",
		"The release already exists and will not be replaced.",
		"$PSNativeCommandUseErrorActionPreference = $false",
		"Windows SmartScreen may show an unknown-publisher warning",
		"Windows 10 version 1709 build 16299 or newer clients",
		"Evergreen WebView2 Runtime 92.0.902.49 or newer",
		"This preview does not provide sealed assets or local tamper resistance",
		"does not provide application-specific executable icons or metadata, an installer, or an updater",
		"velox command and velox.exe name also collide with unrelated released software",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("alpha evidence workflow is missing guarded preview publication contract %q", required)
		}
	}
	if strings.Count(workflow, "contents: write") != 1 {
		t.Fatal("only the isolated preview publication job may receive contents: write")
	}
	for _, forbidden := range []string{"SIGNPATH_", "velox-signing-record authenticode", "attest-build-provenance"} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("unsigned preview workflow contains deferred signing surface %q", forbidden)
		}
	}
}
