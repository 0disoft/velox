package clipboard

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const MaxTextBytes = 32 << 10

var (
	ErrInvalidText = errors.New("Clipboard text must be valid UTF-8 without NUL characters.")
	ErrTooLarge    = errors.New("Clipboard text exceeds the 32 KiB limit.")
	ErrBusy        = errors.New("The clipboard is currently unavailable.")
	ErrNative      = errors.New("The clipboard operation failed.")
)

func Validate(text string) error {
	if len(text) > MaxTextBytes {
		return ErrTooLarge
	}
	if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return ErrInvalidText
	}
	return nil
}
