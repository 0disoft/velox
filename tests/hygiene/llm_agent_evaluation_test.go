package hygiene_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type llmAgentSchema struct {
	Schema     string                     `json:"$schema"`
	ID         string                     `json:"$id"`
	Required   []string                   `json:"required"`
	Properties map[string]json.RawMessage `json:"properties"`
	AllOf      []json.RawMessage          `json:"allOf"`
}

func TestLLMAgentEvaluationSchemaKeepsEvidenceHonest(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "schema", "llm-agent-evaluation-v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema llmAgentSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.Schema != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "https://schemas.0disoft.dev/velox/llm-agent-evaluation-v1.schema.json" {
		t.Fatalf("unexpected LLM evaluation schema identity: %q %q", schema.Schema, schema.ID)
	}
	for _, field := range []string{
		"trialId", "seriesId", "sequence", "promptSha256", "evaluator", "control", "release", "application", "environment",
		"outcome", "gates", "trajectory", "artifacts", "failure", "evidenceLevel", "humanAdoptionClaim",
	} {
		if !containsString(schema.Required, field) {
			t.Errorf("LLM evaluation schema does not require %s", field)
		}
	}
	assertJSONConst(t, schema.Properties["schemaVersion"], "velox.llm-agent-evaluation/v1")
	assertJSONConst(t, schema.Properties["promptVersion"], "velox.llm-agent-task/v1")
	assertJSONConst(t, schema.Properties["evidenceLevel"], "maintainer-orchestrated-clean-room-llm-agent")
	assertJSONConst(t, schema.Properties["humanAdoptionClaim"], false)

	var evaluator, control struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(schema.Properties["evaluator"], &evaluator); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(schema.Properties["control"], &control); err != nil {
		t.Fatal(err)
	}
	assertJSONConst(t, evaluator.Properties["freshSession"], true)
	assertJSONConst(t, evaluator.Properties["memoryCarryover"], false)
	assertJSONConst(t, control.Properties["maintainerOrchestrated"], true)
	assertJSONConst(t, control.Properties["externalHuman"], false)
	assertJSONConst(t, control.Properties["veloxSourceCheckout"], false)
	assertJSONConst(t, control.Properties["unpublishedContext"], false)
	assertJSONConst(t, control.Properties["interactiveMaintainerHints"], float64(0))

	conditions := string(data)
	for _, marker := range []string{
		`"outcome": { "const": "passed" }`,
		`"deterministicBuild": { "const": true }`,
		`"appBehaviorVerified": { "const": true }`,
		`"forbiddenActions": { "maxItems": 0 }`,
		`"failure": { "type": "null" }`,
	} {
		if !strings.Contains(conditions, marker) {
			t.Errorf("LLM evaluation schema lacks pass condition %q", marker)
		}
	}
}

func TestLLMAgentEvaluationAttestationSchemaKeepsTrajectoryExternal(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "schema", "llm-agent-evaluation-attestation-v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema llmAgentSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.Schema != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "https://schemas.0disoft.dev/velox/llm-agent-evaluation-attestation-v1.schema.json" {
		t.Fatalf("unexpected LLM attestation schema identity: %q %q", schema.Schema, schema.ID)
	}
	for _, field := range []string{"trialId", "seriesId", "sequence", "evaluator", "startedAtUtc", "finishedAtUtc", "trajectory", "evidence"} {
		if !containsString(schema.Required, field) {
			t.Errorf("LLM attestation schema does not require %s", field)
		}
	}
	assertJSONConst(t, schema.Properties["schemaVersion"], "velox.llm-agent-evaluation-attestation/v1")
	markers := []string{
		`"toolCallBudget": { "const": 70 }`,
		`"NODE_RUNTIME_INVOKED"`,
		`"CONSUMER_COMPILER_INVOKED"`,
		`"PACKAGE_MANAGER_INVOKED"`,
		`"kind": { "const": "orchestrator-session-log" }`,
		`"observationLevel": { "const": "session-log-heuristic" }`,
		`"sandboxEnforced": { "const": false }`,
		`"projection"`,
	}
	for _, marker := range markers {
		if !strings.Contains(string(data), marker) {
			t.Errorf("LLM attestation schema lacks %q", marker)
		}
	}
}

func TestLLMAgentEvaluationAttestationV2RequiresEnforcedSandboxReceipt(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "schema", "llm-agent-evaluation-attestation-v2.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema llmAgentSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.Schema != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "https://schemas.0disoft.dev/velox/llm-agent-evaluation-attestation-v2.schema.json" {
		t.Fatalf("unexpected LLM attestation v2 schema identity: %q %q", schema.Schema, schema.ID)
	}
	assertJSONConst(t, schema.Properties["schemaVersion"], "velox.llm-agent-evaluation-attestation/v2")
	for _, marker := range []string{
		`"kind": { "const": "orchestrator-session-log-with-os-sandbox" }`,
		`"sandboxEnforced": { "const": true }`,
		`"schemaVersion": { "const": "velox.eval-sandbox-receipt/v1" }`,
		`"filesystemBoundary": { "const": "appcontainer-explicit-acl" }`,
		`"processBoundary": { "const": "job-object-no-breakaway" }`,
		`"networkCapability": { "const": "internet-client" }`,
		`"environmentSha256": { "$ref": "#/$defs/sha256" }`,
		`"cleanupCompleted": { "const": true }`,
		`"timedOut": { "const": false }`,
	} {
		if !strings.Contains(string(data), marker) {
			t.Errorf("LLM attestation v2 schema lacks %q", marker)
		}
	}
}

func TestLLMAgentSeriesSchemaRequiresThreePassesWithoutHumanClaim(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "schema", "llm-agent-evaluation-series-v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema llmAgentSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.Schema != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "https://schemas.0disoft.dev/velox/llm-agent-evaluation-series-v1.schema.json" {
		t.Fatalf("unexpected LLM series schema identity: %q %q", schema.Schema, schema.ID)
	}
	for _, field := range []string{"seriesId", "releaseSha256", "promptSha256", "trialIds", "modelIdentifiers", "outcome", "betaTechnicalGate", "diagnostics", "humanAdoptionClaim"} {
		if !containsString(schema.Required, field) {
			t.Errorf("LLM series schema does not require %s", field)
		}
	}
	assertJSONConst(t, schema.Properties["humanAdoptionClaim"], false)
	conditions := string(data)
	for _, marker := range []string{
		`"passedTrials": { "const": 3 }`,
		`"failedTrials": { "const": 0 }`,
		`"heldTrials": { "const": 0 }`,
		`"modelIdentifiers": { "minItems": 2 }`,
		`"betaTechnicalGate": { "const": true }`,
	} {
		if !strings.Contains(conditions, marker) {
			t.Errorf("LLM series schema lacks pass condition %q", marker)
		}
	}
}

func TestLLMAgentTaskAndDecisionStayBounded(t *testing.T) {
	root := repositoryRoot(t)
	checks := map[string][]string{
		filepath.Join(root, "evals", "llm-agent", "v1", "task.md"): {
			"fresh Windows workspace",
			"may not clone or download the Velox source",
			"Focus Ledger",
			"Build into two distinct clean output directories",
			"Do not expose hidden reasoning",
			"humanAdoptionClaim` must remain `false",
			"SESSION_ID_SHA256",
			"TOOL_CALL_BUDGET",
			"TRIAL_ID",
			"PROMPT_SHA256",
			"Tool-wrapper side effects count as invocations",
		},
		filepath.Join(root, "docs", "ops", "llm-agent-evaluation.md"): {
			"Three consecutive trials pass",
			"At least two distinct model identifiers",
			"Do not discard a failed trial",
			"not human adoption",
			"full transcript",
			"orchestrator attestation",
			"Hermes Attestation Generation",
			"read-only mode",
			"not operating-system process telemetry",
			"Series Orchestration",
			"never replaces a",
		},
		filepath.Join(root, "docs", "adr", "0018-use-clean-room-llm-agent-evaluation.md"): {
			"Status: Accepted",
			"Supersedes in part",
			"three consecutive passing trials",
			"humanAdoptionClaim` to `false",
			"do not prove that a person wants, trusts, or understands",
		},
		filepath.Join(root, "PRIVACY.md"): {
			"Clean-Room Agent Evaluation",
			"SHA-256",
			"must not retain provider credentials",
			"not a human adoption claim",
			"canonical projection",
		},
		filepath.Join(root, "scripts", "hermes-evaluation-attestation.ts"): {
			"readonly: true",
			"HERMES_TOOL_CALL_COUNT_MISMATCH",
			"ATTESTATION_OUTPUT_INSIDE_TRIAL_ROOT",
			"NODE_RUNTIME_INVOKED",
			"MAINTAINER_HINT_OBSERVED",
		},
		filepath.Join(root, "scripts", "llm-agent-orchestrator.ts"): {
			"velox.llm-agent-orchestrator/v1",
			"EVALUATION_ROOT_INSIDE_VELOX_REPOSITORY",
			"SESSION_BINDING_DIGEST_MISMATCH",
			"SERIES_SUMMARY_ALREADY_EXISTS",
			"live-smoke",
			"live-diagnose",
			"SESSION_BINDING_MANIFEST_MISMATCH",
			"PUBLIC_TASK_URL_NOT_IMMUTABLE",
			"RELEASE_URL_TAG_MISMATCH",
		},
	}
	for path, markers := range checks {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, marker := range markers {
			if !strings.Contains(string(body), marker) {
				t.Errorf("%s lacks %q", filepath.Base(path), marker)
			}
		}
	}
}

func assertJSONConst(t *testing.T, raw json.RawMessage, expected any) {
	t.Helper()
	var value struct {
		Const any `json:"const"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	if value.Const != expected {
		t.Fatalf("schema const = %#v, want %#v", value.Const, expected)
	}
}
