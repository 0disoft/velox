package setuppayload

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/velox/internal/buildinfo"
	"github.com/0disoft/velox/internal/releasebundle"
)

func packedFixture(t *testing.T, name string) string {
	t.Helper()
	root := t.TempDir()
	archive := filepath.Join(root, "app.zip")
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "setup.exe")
	first, err := pack(bytes.NewReader([]byte("template")), 8, archive, output)
	if err != nil {
		t.Fatal(err)
	}
	second, err := pack(bytes.NewReader([]byte("template")), 8, archive, output)
	if err != nil || first.SHA256 != second.SHA256 || first.Bytes != second.Bytes {
		t.Fatalf("nondeterministic: %v", err)
	}
	return output
}

func TestPayloadRoundTripAndTamperRefusal(t *testing.T) {
	path := packedFixture(t, "dev.velox.test/file")
	payload, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	copy := filepath.Join(t.TempDir(), "uninstall.exe")
	if err := payload.CopyTemplate(copy); err != nil {
		t.Fatal(err)
	}
	payload.Close()
	data, _ := os.ReadFile(copy)
	if string(data) != "template" {
		t.Fatal("template copy includes payload")
	}
	data, _ = os.ReadFile(path)
	data[10] ^= 1
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if payload, err := Open(path); err == nil {
		payload.Close()
		t.Fatal("accepted modified payload")
	}
}

func TestUnsafeArchiveRefusedBeforeExtraction(t *testing.T) {
	for _, name := range []string{"../escape", "dev.velox.test/../../escape", "dev.velox.test/CON", "dev.velox.test/file:stream"} {
		t.Run(name, func(t *testing.T) {
			payload, err := Open(packedFixture(t, name))
			if err != nil {
				t.Fatal(err)
			}
			defer payload.Close()
			out := filepath.Join(t.TempDir(), "output")
			if _, err := payload.Extract(out); err == nil {
				t.Fatal("accepted unsafe archive")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatal("wrote before ZIP preflight")
			}
		})
	}
}

func TestPayloadFooterBounds(t *testing.T) {
	for _, mutate := range []func([]byte){
		func(data []byte) { data[len(data)-footerSize] ^= 1 },
		func(data []byte) { binary.LittleEndian.PutUint64(data[len(data)-48:], ^uint64(0)) },
		func(data []byte) { binary.LittleEndian.PutUint64(data[len(data)-40:], 0) },
	} {
		path := packedFixture(t, "dev.velox.test/file")
		data, _ := os.ReadFile(path)
		mutate(data)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if payload, err := Open(path); err == nil {
			payload.Close()
			t.Fatal("accepted malformed footer")
		}
	}
}

func TestTemplateReleaseBinding(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "velox-setup.exe")
	data := []byte("template")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	manifest := releasebundle.Manifest{
		SchemaVersion: releasebundle.SchemaVersion, ReleaseVersion: buildinfo.Version, Target: releasebundle.TargetWindowsX64,
		Contracts: releasebundle.Contracts{Manifest: 1, Runtime: 1, Host: 1, IPC: 1, BuildResult: 1},
		Artifacts: []releasebundle.Artifact{{File: "velox-setup.exe", Bytes: 8, SHA256: hex.EncodeToString(hash[:])}},
	}
	write := func() {
		t.Helper()
		data, _ := json.Marshal(manifest)
		if err := os.WriteFile(filepath.Join(root, "release-manifest.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	if err := VerifyTemplate(path); err != nil {
		t.Fatal(err)
	}
	manifest.ReleaseVersion = "old-release"
	write()
	if err := VerifyTemplate(path); err == nil {
		t.Fatal("accepted mismatched release")
	}
	manifest.ReleaseVersion = buildinfo.Version
	write()
	if err := os.WriteFile(path, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyTemplate(path); err == nil {
		t.Fatal("accepted mismatched setup digest")
	}
}
