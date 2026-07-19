package hygiene_test

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const artifactConfiguration = `<?xml version="1.0" encoding="utf-8"?>
<artifact-configuration xmlns="http://signpath.io/artifact-configuration/v1">
  <zip-file>
    <pe-file path="velox-host.exe">
      <authenticode-sign />
    </pe-file>
    <pe-file path="velox.exe">
      <authenticode-sign />
    </pe-file>
  </zip-file>
</artifact-configuration>
`

const sourcePolicy = `github-policies:
  runners:
    require_github_hosted: true
  build:
    disallow_reruns: true
`

func TestSignPathArtifactConfigurationIsExact(t *testing.T) {
	path := repositoryPath(".signpath", "artifact-configuration.xml")
	data := readNormalized(t, path)
	if data != artifactConfiguration {
		t.Fatalf("SignPath artifact configuration drifted:\n%s", data)
	}

	var configuration struct {
		XMLName xml.Name `xml:"artifact-configuration"`
		ZIP     struct {
			PEFiles []struct {
				Path  string     `xml:"path,attr"`
				Signs []struct{} `xml:"authenticode-sign"`
			} `xml:"pe-file"`
		} `xml:"zip-file"`
	}
	decoder := xml.NewDecoder(strings.NewReader(data))
	decoder.Strict = true
	if err := decoder.Decode(&configuration); err != nil {
		t.Fatal(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("artifact configuration has trailing XML: %v", err)
	}
	if configuration.XMLName.Space != "http://signpath.io/artifact-configuration/v1" || len(configuration.ZIP.PEFiles) != 2 {
		t.Fatalf("artifact configuration = %#v", configuration)
	}
	want := []string{"velox-host.exe", "velox.exe"}
	for index, file := range configuration.ZIP.PEFiles {
		if file.Path != want[index] || len(file.Signs) != 1 {
			t.Fatalf("PE file %d = %#v", index, file)
		}
	}
}

func TestSignPathSourcePolicyIsFailClosed(t *testing.T) {
	data := readNormalized(t, repositoryPath(".signpath", "policies", "velox", "release-signing.yml"))
	if data != sourcePolicy {
		t.Fatalf("SignPath source policy drifted:\n%s", data)
	}
}

func TestPublicProjectOwnershipAndPolicyFilesAreReady(t *testing.T) {
	checks := map[string][]string{
		"LICENSE":                         {"MIT OR Apache-2.0", "LICENSE-MIT", "LICENSE-APACHE"},
		"LICENSE-MIT":                     {"MIT License", "Copyright (c) 2026 0disoft"},
		"LICENSE-APACHE":                  {"Apache License", "Version 2.0, January 2004"},
		"SECURITY.md":                     {"security/advisories/new", "Do not open a public issue"},
		"PRIVACY.md":                      {"do not send telemetry", "SignPath"},
		".github/CODEOWNERS":              {"* @0disoft", "/.signpath/ @0disoft", "/.github/workflows/ @0disoft"},
		"docs/ops/signpath-onboarding.md": {"Status: Deferred until an ADR 0011 signing trigger", "SignPath organization ID:", "Never return the API token value", "Private vulnerability reporting | Enabled", "Confirm that you own or can license"},
	}
	for relative, required := range checks {
		data := readNormalized(t, repositoryPath(strings.Split(relative, "/")...))
		if strings.Contains(data, "REPLACE_WITH_OWNER") {
			t.Fatalf("%s still contains placeholder ownership", relative)
		}
		for _, value := range required {
			if !strings.Contains(data, value) {
				t.Fatalf("%s lacks %q", relative, value)
			}
		}
	}
}

func repositoryPath(elements ...string) string {
	return filepath.Join(append([]string{"..", ".."}, elements...)...)
}

func readNormalized(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}
