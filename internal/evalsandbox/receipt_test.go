package evalsandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReceiptJSONContainsNoLocalPaths(t *testing.T) {
	receipt := Receipt{
		SchemaVersion: ReceiptVersion,
		TrialID:       "trial-20260816T010203Z-a1b2c3d4",
		SeriesID:      "series-20260816T010203Z-a1b2c3d4",
		Sequence:      1,
		Policy: Policy{
			SchemaVersion:      PolicyVersion,
			Platform:           "windows",
			FilesystemBoundary: "appcontainer-explicit-acl",
			ProcessBoundary:    "job-object-no-breakaway",
			NetworkCapability:  "internet-client",
		},
		Supervisor:          Supervisor{Version: "0.5.10-alpha.34", SHA256: "a"},
		CommandSHA256:       "b",
		EnvironmentSHA256:   "d",
		PromptSHA256:        "e",
		StateDatabaseSHA256: "f",
		StartedAtUTC:        time.Now().UTC().Format(time.RFC3339Nano),
		FinishedAtUTC:       time.Now().UTC().Format(time.RFC3339Nano),
		Containment:         Containment{FilesystemEnforced: true, ProcessTreeEnforced: true, CleanupCompleted: true},
		Grants:              []Grant{{Role: "trial-read-write-execute", PathSHA256: "c", Rights: "read-write-execute"}},
	}
	body, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) == "" || containsAny(string(body), `C:\\`, `C:/`, `\\\\`) {
		t.Fatalf("receipt exposes a local path: %s", body)
	}
}

func TestWriteReceiptExclusiveRefusesReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt.json")
	if err := WriteReceiptExclusive(path, Receipt{SchemaVersion: ReceiptVersion}); err != nil {
		t.Fatal(err)
	}
	if err := WriteReceiptExclusive(path, Receipt{SchemaVersion: ReceiptVersion}); err == nil {
		t.Fatal("expected exclusive receipt write to reject replacement")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAny(string(body), ReceiptVersion) {
		t.Fatalf("unexpected receipt: %s", body)
	}
}

func TestCommandDigestBindsArgumentBoundaries(t *testing.T) {
	first, err := commandDigest([]string{"tool.exe", "a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := commandDigest([]string{"tool.exe", "a b"})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("command digest must preserve argument boundaries")
	}
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
