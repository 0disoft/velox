package hygiene_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type publicEvidenceFixture struct {
	SchemaVersion string         `json:"schemaVersion"`
	AssetName     string         `json:"assetName"`
	PayloadSHA256 string         `json:"payloadSha256"`
	Payload       map[string]any `json:"payload"`
}

func TestLLMAgentPublicEvidenceKeepsOnePublicImmutableContract(t *testing.T) {
	root := repositoryRoot(t)

	schemaPath := filepath.Join(root, "schema", "llm-agent-public-evidence-v1.schema.json")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		`"$id": "https://schemas.0disoft.dev/velox/llm-agent-public-evidence-v1.schema.json"`,
		`"const": "velox.llm-agent-public-evidence/v1"`,
		`"const": "permanent"`,
		`"const": "velox.llm-agent-evaluation-attestation/v2"`,
		`"const": "appcontainer-explicit-acl"`,
		`"const": "job-object-no-breakaway"`,
		`"minItems": 3`,
		`"maxItems": 3`,
		`"additionalProperties": false`,
	} {
		if !strings.Contains(string(schema), marker) {
			t.Errorf("public evidence schema lacks %q", marker)
		}
	}

	fixtureRoot := filepath.Join(root, "tests", "fixtures", "llm-agent-public-evidence")
	validBody, err := os.ReadFile(filepath.Join(fixtureRoot, "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	var valid publicEvidenceFixture
	if err := json.Unmarshal(validBody, &valid); err != nil {
		t.Fatal(err)
	}
	if valid.SchemaVersion != "velox.llm-agent-public-evidence/v1" {
		t.Fatalf("public evidence fixture schema = %q", valid.SchemaVersion)
	}
	canonicalPayload, err := json.Marshal(valid.Payload)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonicalPayload)
	if got := hex.EncodeToString(digest[:]); got != valid.PayloadSHA256 {
		t.Fatalf("public evidence fixture payload digest = %s, want %s", got, valid.PayloadSHA256)
	}
	series, ok := valid.Payload["series"].(map[string]any)
	if !ok {
		t.Fatal("public evidence fixture lacks series object")
	}
	seriesID, ok := series["seriesId"].(string)
	if !ok || seriesID == "" {
		t.Fatal("public evidence fixture lacks series ID")
	}
	expectedAssetName := "velox-llm-agent-evidence-" + seriesID + "-" + valid.PayloadSHA256 + ".json"
	if valid.AssetName != expectedAssetName {
		t.Fatalf("public evidence fixture asset = %q, want %q", valid.AssetName, expectedAssetName)
	}

	verifier, err := os.ReadFile(filepath.Join(root, "scripts", "llm-agent-public-evidence.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"PUBLIC_EVIDENCE_REQUIRED_FIELD_MISSING",
		"PUBLIC_EVIDENCE_PAYLOAD_DIGEST_MISMATCH",
		"PUBLIC_EVIDENCE_DUPLICATE_TRIAL_ID",
		"PUBLIC_EVIDENCE_ATTESTATION_SCHEMA_INCOMPATIBLE",
		"PUBLIC_EVIDENCE_FORBIDDEN_PRIVATE_FIELD",
		"canonicalJSON",
		"maximumPacketBytes",
	} {
		if !strings.Contains(string(verifier), marker) {
			t.Errorf("public evidence verifier lacks %q", marker)
		}
	}

	for _, name := range []string{
		"missing.json",
		"altered.json",
		"duplicate.json",
		"incompatible.json",
		"forbidden-private-field.json",
	} {
		if _, err := os.Stat(filepath.Join(fixtureRoot, name)); err != nil {
			t.Errorf("public evidence rejection fixture %s: %v", name, err)
		}
	}

	documents := map[string][]string{
		filepath.Join(root, "docs", "ops", "llm-agent-public-evidence.md"): {
			"same immutable GitHub Release",
			"permanent retention",
			"Do not delete it, replace it",
			"public download passes",
			"session identifiers even when hashed",
		},
		filepath.Join(root, "PRIVACY.md"): {
			"durable public packet",
			"does not publish a raw or hashed session identifier",
		},
		filepath.Join(root, "docs", "README.md"): {
			"docs/ops/llm-agent-public-evidence.md",
		},
	}
	for path, markers := range documents {
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
