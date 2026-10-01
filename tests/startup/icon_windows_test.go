package startup_test

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"

	"golang.org/x/sys/windows"
)

func testBuiltHostIcons(t *testing.T) {
	module, err := windows.LoadLibraryEx(requiredExecutable(t, "VELOX_BUILT_HOST"), 0, 0x22)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.FreeLibrary(module)
	resource := func(kind, id uintptr) []byte {
		t.Helper()
		entry, _, _ := kernel32.NewProc("FindResourceW").Call(uintptr(module), id, kind)
		if entry == 0 {
			t.Fatalf("missing icon resource kind=%d id=%d", kind, id)
		}
		data, err := windows.LoadResourceData(module, windows.Handle(entry))
		if err != nil || len(data) == 0 || len(data) > 1<<20 {
			t.Fatalf("invalid resource size=%d: %v", len(data), err)
		}
		return bytes.Clone(data)
	}
	want := []int{16, 20, 24, 32, 40, 48, 64, 128, 256}
	for _, groupID := range []uintptr{1, 11} {
		group := resource(14, groupID)
		if len(group) != 6+14*len(want) || binary.LittleEndian.Uint16(group[2:]) != 1 || int(binary.LittleEndian.Uint16(group[4:])) != len(want) {
			t.Fatalf("unexpected group icon header: %x", group)
		}
		seen := make(map[int]bool)
		for i := range want {
			entry := group[6+14*i:]
			size := int(entry[0])
			if size == 0 {
				size = 256
			}
			data := resource(3, uintptr(binary.LittleEndian.Uint16(entry[12:])))
			config, err := png.DecodeConfig(bytes.NewReader(data))
			if err != nil || config.Width != size || config.Height != size || seen[size] {
				t.Fatalf("icon group=%d size=%d config=%+v err=%v", groupID, size, config, err)
			}
			seen[size] = true
		}
		for _, size := range want {
			if !seen[size] {
				t.Fatalf("icon group=%d missing size=%d", groupID, size)
			}
		}
	}
}
