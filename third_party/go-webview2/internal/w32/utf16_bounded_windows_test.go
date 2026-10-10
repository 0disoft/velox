package w32

import (
	"strings"
	"testing"
	"unicode/utf16"
)

func TestUtf16PtrToStringBounded(t *testing.T) {
	for _, tc := range []struct {
		name    string
		units   []uint16
		limit   int
		want    string
		allowed bool
	}{
		{"nil", nil, 0, "", true},
		{"empty", []uint16{0}, 0, "", true},
		{"zero-limit", []uint16{'a', 0}, 0, "", false},
		{"negative-limit", []uint16{0}, -1, "", false},
		{"ascii-exact", []uint16{'a', 'b', 0}, 2, "ab", true},
		{"ascii-over", []uint16{'a', 'b', 'c', 0}, 2, "", false},
		{"no-terminator-in-prefix", []uint16{'a', 'b', 'c'}, 2, "", false},
		{"first-terminator", []uint16{'a', 0, 'b'}, 2, "a", true},
		{"korean", []uint16{0xd55c, 0xae00, 0}, 2, "\ud55c\uae00", true},
		{"surrogate-pair", []uint16{0xd83d, 0xde00, 0}, 2, "\U0001f600", true},
		{"unpaired-surrogate", []uint16{0xd83d, 0}, 1, "\ufffd", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p *uint16
			if len(tc.units) > 0 {
				p = &tc.units[0]
			}
			got, allowed := Utf16PtrToStringBounded(p, tc.limit)
			if got != tc.want || allowed != tc.allowed {
				t.Fatalf("got %q %t want %q %t", got, allowed, tc.want, tc.allowed)
			}
		})
	}
}

func TestUtf16OversizedPrefixDoesNotAllocate(t *testing.T) {
	units := utf16.Encode([]rune(strings.Repeat("a", 65537)))
	units = append(units, 0)
	allocations := testing.AllocsPerRun(20, func() {
		if decoded, allowed := Utf16PtrToStringBounded(&units[0], 65536); decoded != "" || allowed {
			t.Fatal("oversized prefix accepted")
		}
	})
	if allocations != 0 {
		t.Fatalf("oversized prefix allocated %g times", allocations)
	}
}
