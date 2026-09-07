package archive

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/velox/internal/artifactlimits"
)

func TestStreamRejectsBudgetOverflowAndRemovesPartialOutput(t *testing.T) {
	for _, kind := range []string{"entry", "total", "files"} {
		t.Run(kind, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "limited.zip")
			stream, err := NewStream(destination)
			if err != nil {
				t.Fatal(err)
			}
			defer stream.Abort()
			entry, err := stream.CreateEntry("app/file.txt", 0o644)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "entry":
				entry.(*boundedEntry).written = artifactlimits.MaxEntryBytes
			case "total":
				stream.budget.Bytes = artifactlimits.MaxTotalBytes
			case "files":
				stream.budget.Files = artifactlimits.MaxFiles
				if _, err := stream.CreateEntry("app/extra.txt", 0o644); err == nil {
					t.Fatal("accepted excessive file count")
				}
				return
			}
			if n, err := entry.Write([]byte("x")); n != 0 || err == nil {
				t.Fatalf("Write() = %d, %v", n, err)
			}
			if _, err := stream.Close(); err == nil {
				t.Fatal("published failed stream")
			}
			if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatalf("partial output remains: %v", err)
			}
		})
	}
}

func TestStreamMatchesIndependentCompressors(t *testing.T) {
	var expected bytes.Buffer
	baseline := zip.NewWriter(&expected)
	baseline.RegisterCompressor(zip.Deflate, func(writer io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(writer, flate.BestSpeed)
	})
	destination := filepath.Join(t.TempDir(), "multiple.zip")
	stream, err := NewStream(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Abort()
	for _, fixture := range []struct{ name, content string }{
		{"first.txt", strings.Repeat("first repeated content", 100)},
		{"empty.txt", ""},
		{"stored.png", "already compressed fixture"},
		{"last.txt", strings.Repeat("different repeated content", 100)},
	} {
		name := "app/" + fixture.name
		header := &zip.FileHeader{Name: name, Method: compressionMethod(name), Modified: normalizedTime}
		header.SetMode(0o644)
		reference, err := baseline.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		entry, err := stream.CreateEntry(name, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		for _, output := range []io.Writer{reference, entry} {
			if _, err := io.WriteString(output, fixture.content); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := baseline.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected.Bytes()) {
		t.Fatal("stream bytes differ from independently compressed entries")
	}
}

func BenchmarkSmallFileCompression(b *testing.B) {
	const files = 100
	payload := bytes.Repeat([]byte("0123456789abcdef"), 64)
	destination := filepath.Join(b.TempDir(), "small-files.zip")
	b.ReportAllocs()
	b.SetBytes(int64(files * len(payload)))
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		stream, err := NewStream(destination)
		if err != nil {
			b.Fatal(err)
		}
		for index := 0; index < files; index++ {
			entry, err := stream.CreateEntry(fmt.Sprintf("app/%03d.txt", index), 0o644)
			if err != nil {
				stream.Abort()
				b.Fatal(err)
			}
			if _, err := entry.Write(payload); err != nil {
				stream.Abort()
				b.Fatal(err)
			}
		}
		if _, err := stream.Close(); err != nil {
			b.Fatal(err)
		}
		if err := os.Remove(destination); err != nil {
			b.Fatal(err)
		}
	}
}

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
