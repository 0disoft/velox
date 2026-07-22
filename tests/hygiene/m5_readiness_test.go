package hygiene_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type maintenanceRecord struct {
	SchemaVersion     string `json:"schemaVersion"`
	ReleaseVersion    string `json:"releaseVersion"`
	CapturedOn        string `json:"capturedOn"`
	ObservationWindow struct {
		From           string `json:"from"`
		Through        string `json:"through"`
		BaselineCommit string `json:"baselineCommit"`
		HeadCommit     string `json:"headCommit"`
		Commits        int    `json:"commits"`
		ChangedFiles   int    `json:"changedFiles"`
		Insertions     int    `json:"insertions"`
		Deletions      int    `json:"deletions"`
	} `json:"observationWindow"`
	RepositorySurface struct {
		ProductionGoFiles          int `json:"productionGoFiles"`
		ProductionGoLines          int `json:"productionGoLines"`
		TestGoFiles                int `json:"testGoFiles"`
		TestGoLines                int `json:"testGoLines"`
		VendoredGoFiles            int `json:"vendoredGoFiles"`
		VendoredGoLines            int `json:"vendoredGoLines"`
		WorkflowFiles              int `json:"workflowFiles"`
		WorkflowLines              int `json:"workflowLines"`
		SchemaFiles                int `json:"schemaFiles"`
		DocumentationMarkdownFiles int `json:"documentationMarkdownFiles"`
		DirectModuleDependencies   int `json:"directModuleDependencies"`
		IndirectModuleDependencies int `json:"indirectModuleDependencies"`
		SupportedTargets           int `json:"supportedTargets"`
		PublicCLICommands          int `json:"publicCLICommands"`
		NativeIPCMethods           int `json:"nativeIPCMethods"`
	} `json:"repositorySurface"`
	RecurringHostedWork struct {
		ScheduledWorkflowsPerWeek       int `json:"scheduledWorkflowsPerWeek"`
		WindowsJobsPerWeek              int `json:"windowsJobsPerWeek"`
		MaximumWindowsJobMinutesPerWeek int `json:"maximumWindowsJobMinutesPerWeek"`
		UbuntuJobsPerWeek               int `json:"ubuntuJobsPerWeek"`
		MaximumUbuntuJobMinutesPerWeek  int `json:"maximumUbuntuJobMinutesPerWeek"`
	} `json:"recurringHostedWork"`
	ManualPreviewSteps []string `json:"manualPreviewSteps"`
	NonClaims          []string `json:"nonClaims"`
}

func TestMaintenanceCostRecordIsBoundedAndMachineReadable(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "product", "maintenance-cost-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record maintenanceRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	if record.SchemaVersion != "velox.maintenance-cost/v1" || record.ReleaseVersion != "0.5.10-alpha.1" {
		t.Fatalf("maintenance record identity = %q %q", record.SchemaVersion, record.ReleaseVersion)
	}
	for _, value := range []string{record.CapturedOn, record.ObservationWindow.From, record.ObservationWindow.Through} {
		if _, err := time.Parse("2006-01-02", value); err != nil {
			t.Fatalf("maintenance date %q is invalid: %v", value, err)
		}
	}
	for name, value := range map[string]string{
		"baseline": record.ObservationWindow.BaselineCommit,
		"head":     record.ObservationWindow.HeadCommit,
	} {
		if len(value) != 40 || strings.Trim(value, "0123456789abcdef") != "" {
			t.Fatalf("%s commit %q is not a lowercase full SHA", name, value)
		}
	}
	if record.ObservationWindow.Commits != 42 || record.ObservationWindow.ChangedFiles != 271 || record.ObservationWindow.Insertions != 29416 || record.ObservationWindow.Deletions != 0 {
		t.Fatalf("maintenance observation window drifted: %+v", record.ObservationWindow)
	}
	if record.RepositorySurface.ProductionGoFiles != 46 || record.RepositorySurface.ProductionGoLines != 6909 ||
		record.RepositorySurface.TestGoFiles != 39 || record.RepositorySurface.TestGoLines != 5145 ||
		record.RepositorySurface.VendoredGoFiles != 42 || record.RepositorySurface.VendoredGoLines != 3859 ||
		record.RepositorySurface.WorkflowFiles != 4 || record.RepositorySurface.WorkflowLines != 825 ||
		record.RepositorySurface.SchemaFiles != 20 || record.RepositorySurface.DocumentationMarkdownFiles != 43 ||
		record.RepositorySurface.DirectModuleDependencies != 2 || record.RepositorySurface.IndirectModuleDependencies != 1 ||
		record.RepositorySurface.SupportedTargets != 1 || record.RepositorySurface.PublicCLICommands != 7 ||
		record.RepositorySurface.NativeIPCMethods != 6 {
		t.Fatalf("maintenance repository surface drifted: %+v", record.RepositorySurface)
	}
	if record.RecurringHostedWork.ScheduledWorkflowsPerWeek != 1 || record.RecurringHostedWork.WindowsJobsPerWeek != 12 ||
		record.RecurringHostedWork.MaximumWindowsJobMinutesPerWeek != 63 || record.RecurringHostedWork.UbuntuJobsPerWeek != 1 ||
		record.RecurringHostedWork.MaximumUbuntuJobMinutesPerWeek != 5 {
		t.Fatalf("maintenance hosted-work boundary drifted: %+v", record.RecurringHostedWork)
	}
	if len(record.ManualPreviewSteps) != 3 {
		t.Fatalf("manual preview steps = %d, want 3", len(record.ManualPreviewSteps))
	}
	nonClaims := strings.Join(record.NonClaims, "\n")
	for _, fragment := range []string{"does not estimate human engineering hours", "ceilings, not billed", "does not predict external support volume", "counted separately"} {
		if !strings.Contains(nonClaims, fragment) {
			t.Errorf("maintenance record is missing non-claim %q", fragment)
		}
	}
}

func TestM5ReadinessDocumentsStaySynchronized(t *testing.T) {
	root := repositoryRoot(t)
	assertSourceMarkers(t, filepath.Join(root, "docs", "engineering", "08-m4-security-review.md"), []string{
		"penetration test, independent audit",
		"SEC-001 | Medium | Resolved",
		"SEC-002 | High | Accepted for preview",
		"SEC-003 | High | Monitoring after public verification",
		"29715002921",
		"SEC-004 | Medium | Monitoring",
		"No unowned internal security finding blocks",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "product", "04-maintenance-cost-record.md"), []string{
		"implementation and evidence commits, 6,909 production Go lines",
		"63 job-minutes",
		"Invented person-hours are not",
		"M5 must not read fast consumer builds",
		"first public preview required",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "product", "03-risk-register.md"), []string{
		"| R-008 | Full benchmark CI consumes excessive Actions resources | Medium | Medium | Mitigated |",
		"| R-010 | External WebView2 policy or runtime availability blocks users | Medium | High | Mitigated |",
		"public-download verifier exercised the supported path",
		"ADR 0018 permits agent-evaluated beta readiness",
		"| R-019 | LLM evaluation overfits the maintainer prompt or one model family | High | High | Monitoring |",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "product", "01-roadmap.md"), []string{
		"maintenance-cost record, internal security review, and public M4 distribution",
		"evidence gate revised by ADR 0018",
		"three consecutive clean-room LLM agent",
		"No qualifying LLM trial series",
		"public identity decision is complete under ADR 0015",
		"no human adoption claim",
		"29736140250",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "ops", "ci.md"), []string{
		"maintainer-owned direct pushes to `main`",
		"does not make them merge gates",
		"On 2026-07-20, the GitHub repository settings showed zero classic branch",
		"protection rules and zero rulesets",
		"not an automated drift check",
		"Before beta or external contributors receive write access",
		"must not claim branch protection before",
		"Cross-framework immediate-relaunch cause classification",
		"public-alpha availability does not change that evidence",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "engineering", "03-performance-budget.md"), []string{
		"M3 cold-build gate passed; startup guardrail active",
		"hosted consumer workflow contains no cache action",
		"not a working-set, transfer-byte, or",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "engineering", "06-dependency-and-change-policy.md"), []string{
		"replaces that module",
		"third_party/go-webview2",
		"`cmd/velox-host` composes that",
		"not a disposable feasibility spike",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "product", "02-spec.md"), []string{
		"Public name: Velox",
		"Windows 10 version 1709 build 16299",
		"Minimum WebView2 Runtime `92.0.902.49`",
		"explicitly unsigned",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "product", "05-naming-review.md"), []string{
		"released Go internet-speed-test CLI",
		"ships `velox.exe`",
		"Selected public name: Velox",
		"unsigned developer preview is `0.5.10-alpha.1`",
		"project maintainer never approved",
		"changing the product away from Velox",
		"accepts the developer-discovery, PATH, package, and support-search",
		"Actutum is not an alias",
		"not a publication prerequisite",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "adr", "0015-retain-velox-public-identity.md"), []string{
		"Retain **Velox**",
		"Supersedes: ADR 0013 and ADR 0014",
		"do not by themselves block the first",
		"developer preview",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "ops", "release.md"), []string{
		"M4 unsigned distribution complete; M5 adoption evidence remains empty",
		"ADR 0015 removes the replacement-name gate",
		"maintainerControlled: true",
		"29736140250",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "engineering", "08-m4-security-review.md"), []string{
		"public-download",
		"29715002921",
		"29736140250",
		"independent-user adoption evidence is recorded",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "ops", "00-operational-contract.md"), []string{
		"Narrow alpha active; beta gated by clean-room LLM agent evaluation",
		"29714173324",
		"29715002921",
		"29736140250",
		"externalUserAttempt: false",
		"29806946109",
		"three consecutive clean-room LLM agent trials",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "adr", "0017-continue-as-a-narrow-static-packager.md"), []string{
		"Status: Accepted",
		"zero qualifying independent-user attempts",
		"Continue Velox toward a stable **narrow static desktop packager**",
		"does **not** approve an application-specific Go backend",
		"Beta or stable admission requires either",
		"29806946109",
		"disabled and skipped",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "adr", "0018-use-clean-room-llm-agent-evaluation.md"), []string{
		"Status: Accepted",
		"Supersedes in part",
		"Replace the uncontrollable human-attempt requirement",
		"three consecutive passing trials",
		"humanAdoptionClaim` to `false",
	})
	assertSourceMarkers(t, filepath.Join(root, "docs", "ops", "rollback.md"), []string{
		"Active for unsigned developer-preview releases",
		"v0.5.10-alpha.1",
		"No rollback incident",
		"rollback controls remain dormant",
	})
	assertSourceMarkers(t, filepath.Join(root, ".github", "workflows", "alpha-evidence.yml"), []string{
		"if: github.repository == '0disoft/velox' && github.event_name == 'workflow_dispatch' && inputs.publish_preview",
	})

	for path, forbidden := range map[string][]string{
		filepath.Join(root, "docs", "engineering", "04-security-baseline.md"): {
			"Public release executables are unsigned",
		},
		filepath.Join(root, "docs", "product", "02-spec.md"): {
			"- Minimum supported Windows release.",
			"- Minimum supported WebView2 Runtime version.",
			"Working name: Velox",
			"- Public product name and package namespaces.",
		},
		filepath.Join(root, "docs", "product", "01-roadmap.md"): {
			"needs M4 distribution evidence, external user attempts, a bounded",
			"public executable blocked pending rename",
			"identity decision, the public M4 distribution evidence",
			"The only remaining M4 gate",
			"M4 remains open only until",
		},
		filepath.Join(root, "docs", "ops", "release.md"): {
			"create the candidate tag as a shortcut around the rename",
			"independent external-user attempt pending",
		},
		filepath.Join(root, "docs", "ops", "00-operational-contract.md"): {
			"No public distribution",
			"only remaining M4 operational gate",
		},
		filepath.Join(root, "docs", "ops", "rollback.md"): {
			"exercised it against a public tag",
		},
		filepath.Join(root, "docs", "product", "05-naming-review.md"): {
			"The first unpublished",
		},
		filepath.Join(root, "docs", "product", "03-risk-register.md"): {
			"| R-008 | Full benchmark CI consumes excessive Actions resources | Medium | Medium | Open |",
			"| R-010 | External WebView2 policy or runtime availability blocks users | Medium | High | Open |",
		},
		filepath.Join(root, "docs", "engineering", "03-performance-budget.md"): {
			"A future hosted workflow must",
		},
		filepath.Join(root, "docs", "ops", "ci.md"): {
			"remains unpublished until immutable release",
		},
		filepath.Join(root, "docs", "engineering", "06-dependency-and-change-policy.md"): {
			"not yet for the product host",
			"confined to `cmd/velox-host` during M0",
		},
		filepath.Join(root, "docs", "engineering", "08-m4-security-review.md"): {
			"does not close the separate M4",
			"independent external-user gate remains open",
		},
		filepath.Join(root, "docs", "product", "04-maintenance-cost-record.md"): {
			"first public preview requires",
		},
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range forbidden {
			if strings.Contains(string(data), fragment) {
				t.Errorf("%s retains stale contract %q", path, fragment)
			}
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
