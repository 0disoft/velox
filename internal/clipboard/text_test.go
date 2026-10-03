package clipboard

import (
	"errors"
	"strings"
	"testing"
)

func TestTextValidation(t *testing.T) {
	for _, text := range []string{"", "hello\nworld", "\ud55c\uae00 \U0001f642", strings.Repeat("x", MaxTextBytes)} {
		if err := Validate(text); err != nil {
			t.Fatalf("valid text rejected: %v", err)
		}
	}
	for _, text := range []string{"a\x00b", string([]byte{0xff})} {
		if !errors.Is(Validate(text), ErrInvalidText) {
			t.Fatal("invalid text accepted")
		}
	}
	if !errors.Is(Validate(strings.Repeat("x", MaxTextBytes+1)), ErrTooLarge) ||
		!errors.Is(Validate(strings.Repeat("\ud55c", MaxTextBytes/3+1)), ErrTooLarge) {
		t.Fatal("UTF-8 byte budget not enforced")
	}
}
