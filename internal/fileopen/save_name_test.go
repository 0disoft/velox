package fileopen

import (
	"strings"
	"testing"
)

func TestSaveNameUTF8ByteBoundary(t *testing.T) {
	for _, tc := range []struct {
		name  string
		valid bool
	}{
		{strings.Repeat("\uD55C", 78) + ".txt", true},
		{strings.Repeat("\uD55C", 80) + ".txt", false},
		{strings.Repeat("\U0001F642", 59) + ".txt", true},
		{strings.Repeat("\U0001F642", 60) + ".txt", false},
		{strings.Repeat("a", 236) + ".txt", true},
		{strings.Repeat("a", 237) + ".txt", false},
		{"Untitled.txt", true},
	} {
		if valid := ValidateSaveName(tc.name) == nil; valid != tc.valid {
			t.Fatalf("UTF-8 bytes=%d valid=%v want %v", len(tc.name), valid, tc.valid)
		}
	}
}
