package archive

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestStreamWritesVerifiedArchive(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "stream.zip")
	stream, err := NewStream(destination)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := stream.CreateEntry("app/web/index.html", 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, "<title>Velox</title>"); err != nil {
		t.Fatal(err)
	}
	result, err := stream.Close()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	if result.FileCount != 1 || result.Size != int64(len(data)) || result.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("result = %#v, bytes = %d, digest = %x", result, len(data), digest)
	}
	reader, err := zip.OpenReader(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 1 || reader.File[0].Name != "app/web/index.html" || reader.File[0].Method != zip.Deflate {
		t.Fatalf("entries = %#v", entryNames(reader.File))
	}
}

func TestStreamRejectsUnsafeDuplicateAndClosedEntries(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "stream.zip")
	stream, err := NewStream(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Abort()
	if _, err := stream.CreateEntry("../escape.txt", 0o644); err == nil {
		t.Fatal("stream accepted unsafe entry")
	}
	if _, err := stream.CreateEntry("app/file.txt", 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.CreateEntry("APP/FILE.TXT", 0o644); err == nil {
		t.Fatal("stream accepted case-colliding entry")
	}
	if _, err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.CreateEntry("app/late.txt", 0o644); err == nil {
		t.Fatal("closed stream accepted entry")
	}
}

func TestStreamAbortAndEmptyCloseRemovePartialOutput(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"aborted.zip", "empty.zip"} {
		destination := filepath.Join(root, name)
		stream, err := NewStream(destination)
		if err != nil {
			t.Fatal(err)
		}
		if name == "aborted.zip" {
			if _, err := stream.CreateEntry("app/file.txt", 0o644); err != nil {
				t.Fatal(err)
			}
			if err := stream.Abort(); err != nil {
				t.Fatal(err)
			}
		} else if _, err := stream.Close(); err == nil {
			t.Fatal("empty stream closed successfully")
		}
		if _, err := os.Stat(destination); !os.IsNotExist(err) {
			t.Fatalf("partial output remains for %s: %v", name, err)
		}
	}
}
