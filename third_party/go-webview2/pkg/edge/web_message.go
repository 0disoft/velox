package edge

import "github.com/jchv/go-webview2/internal/w32"

func decodeWebMessage(message *uint16, maxBytes int) (string, bool) {
	if maxBytes <= 0 {
		return w32.Utf16PtrToString(message), true
	}
	// UTF-8 needs at least as many bytes as UTF-16 has code units, including
	// surrogate replacements. This bound cannot reject a valid in-budget string.
	decoded, terminated := w32.Utf16PtrToStringBounded(message, maxBytes)
	if !terminated || len(decoded) > maxBytes {
		return "", false
	}
	return decoded, true
}
