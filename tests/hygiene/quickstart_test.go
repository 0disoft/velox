package hygiene_test

import (
	"strings"
	"testing"
)

func TestReleaseQuickstartStartsFromImmutablePublicBytes(t *testing.T) {
	doc := strings.Join(strings.Fields(readNormalized(t, repositoryPath("docs", "QUICKSTART.md"))), " ")
	for _, required := range []string{
		"choose one immutable `vX.Y.Z-alpha.N` or `vX.Y.Z-beta.N` release",
		"velox-windows-x64.zip",
		"checksums.sha256",
		"$ChecksumMatches.Count -ne 1",
		"Get-FileHash",
		"Expand-Archive",
		"tool\\velox-windows-x64\\velox.exe",
		"& $Velox version --json",
		"& $Velox init",
		"& $Velox validate",
		"& $Velox doctor",
		"& $Velox build",
		"& $Velox inspect",
		"& $Velox run",
		"Close the application window",
		"A visible window alone is not proof of usable content",
		"Do not install another toolchain",
	} {
		if !strings.Contains(doc, required) {
			t.Errorf("release quickstart lacks %q", required)
		}
	}

	for _, forbidden := range []string{
		"releases/latest",
		"git clone",
		"go install",
		"go build",
		"npm install",
		"pnpm install",
		"yarn install",
		"bun install",
		"actions/cache",
	} {
		if strings.Contains(strings.ToLower(doc), strings.ToLower(forbidden)) {
			t.Errorf("release quickstart contains forbidden consumer path %q", forbidden)
		}
	}
}

func TestLLMAgentTaskUsesPublicQuickstartWithoutLocalFallback(t *testing.T) {
	task := strings.Join(strings.Fields(readNormalized(t, repositoryPath("evals", "llm-agent", "v1", "task.md"))), " ")
	for _, required := range []string{
		"public `docs/QUICKSTART.md`",
		"Do not use a local copy",
		"may not clone or download the Velox source tree",
	} {
		if !strings.Contains(task, required) {
			t.Errorf("LLM agent task lacks quickstart boundary %q", required)
		}
	}
}
