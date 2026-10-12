//go:build windows

package fileopen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListedLongWindowsBasenamesOpenWithoutChangingDiskBytes(t *testing.T) {
	for _, name := range []string{strings.Repeat("a", 251) + ".txt", strings.Repeat("\uD55C", 251) + ".txt", strings.Repeat("\U0001F642", 125) + "a.txt"} {
		root := t.TempDir()
		path := filepath.Join(root, name)
		data := []byte("\uD55C\uAE00 \U0001F642\r\n<script>literal</script>\r\n")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		snapshot, err := inspectSelectedFolder(root)
		if err != nil {
			t.Fatal(err)
		}
		listed, err := listSelectedFolder(root, snapshot.ID)
		if err != nil || len(listed.Entries) != 1 || listed.Entries[0].Name != name {
			t.Fatalf("list: %+v %v", listed, err)
		}
		result, err := readFolderText(root, snapshot.ID, listed.Entries[0].Name)
		if err != nil || result.Name != name || result.Text != string(data) || result.Bytes != len(data) {
			t.Fatalf("listed file cannot be read: bytes=%d result=%+v err=%v", len(name), result, err)
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(after, data) {
			t.Fatal("read changed file")
		}
	}
}
