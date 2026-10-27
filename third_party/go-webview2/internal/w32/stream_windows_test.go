//go:build windows

package w32

import "testing"

func TestNativeStreamReleaseClearsItsOwnedReference(t *testing.T) {
	stream, err := SHCreateMemStream([]byte{0, 1, 2, 255})
	if err != nil || stream == 0 {
		t.Fatalf("create native stream: %v", err)
	}
	ReleaseNativeIUnknown(&stream)
	if stream != 0 {
		t.Fatal("released stream retained an address")
	}
	ReleaseNativeIUnknown(&stream) // A cleared reference must not release twice.
}
