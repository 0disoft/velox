package edge

import (
	"strings"
	"testing"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestDecodeWebMessageUTF8Limits(t *testing.T) {
	const limit = 64 << 10
	for _, tc := range []struct {
		name, text string
		budget     int
		allowed    bool
	}{
		{"empty", "", limit, true},
		{"ascii-exact", strings.Repeat("a", limit), limit, true},
		{"ascii-over", strings.Repeat("a", limit+1), limit, false},
		{"korean-exact", strings.Repeat("\ud55c", limit/3) + "a", limit, true},
		{"korean-over", strings.Repeat("\ud55c", limit/3) + "ab", limit, false},
		{"emoji-exact", strings.Repeat("\U0001f600", limit/4), limit, true},
		{"emoji-over", strings.Repeat("\U0001f600", limit/4) + "a", limit, false},
		{"controls-exact", strings.Repeat("\n\t\r", limit/3) + "a", limit, true},
		{"controls-over", strings.Repeat("\n\t\r", limit/3) + "ab", limit, false},
		{"no-limit", strings.Repeat("a", limit+1), 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			units := append(utf16.Encode([]rune(tc.text)), 0)
			got, allowed := decodeWebMessage(&units[0], tc.budget)
			if allowed != tc.allowed || allowed && got != tc.text || !allowed && got != "" {
				t.Fatalf("allowed=%t len=%d", allowed, len(got))
			}
		})
	}
	for _, tc := range []struct {
		units   []uint16
		budget  int
		allowed bool
	}{
		{[]uint16{0xd83d, 0}, 3, true},
		{[]uint16{0xd83d, 0}, 2, false},
		{[]uint16{0xde00, 'a', 0}, 4, true},
		{[]uint16{0xde00, 'a', 0}, 3, false},
	} {
		got, allowed := decodeWebMessage(&tc.units[0], tc.budget)
		if allowed != tc.allowed || allowed && len(got) > tc.budget {
			t.Fatalf("surrogate decoding: %q %t", got, allowed)
		}
	}
}

func TestMessageReceivedChecksOriginAndSizeWithoutEcho(t *testing.T) {
	const limit = 64 << 10
	const trusted = "https://app.velox.test/"
	for _, tc := range []struct {
		name, source, message, blocked string
		sourceResult, messageResult    uintptr
		nilMessage                     bool
	}{
		{name: "normal", source: trusted, message: `{"id":1,"method":"app.info"}`},
		{name: "korean", source: trusted, message: "\ud55c\uae00"},
		{name: "ascii-limit", source: trusted, message: strings.Repeat("a", limit)},
		{name: "ascii-over", source: trusted, message: strings.Repeat("a", limit+1), blocked: "message-size"},
		{name: "multibyte-over", source: trusted, message: strings.Repeat("\U0001f600", limit/4) + "a", blocked: "message-size"},
		{name: "remote-origin", source: "https://remote.test/", message: "secret", blocked: "message-source"},
		{name: "source-failure", sourceResult: 0x80004005},
		{name: "message-failure", source: trusted, messageResult: 0x80004005},
		{name: "nil-message", source: trusted, nilMessage: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chromium := NewChromium()
			defer chromium.Destroy()
			chromium.MaxWebMessageBytes = limit
			chromium.MessageSourceAllowed = func(source string) bool { return source == trusted }
			var blocked []string
			chromium.PolicyBlocked = func(kind string) { blocked = append(blocked, kind) }
			reads, callbacks, echoes := 0, 0, 0
			chromium.MessageCallback = func(message string) {
				callbacks++
				if message != tc.message {
					t.Fatal("callback content changed")
				}
			}
			args := &iCoreWebView2WebMessageReceivedEventArgs{vtbl: &iCoreWebView2WebMessageReceivedEventArgsVtbl{
				GetSource: NewComProc(func(_ uintptr, out **uint16) uintptr {
					if tc.sourceResult != 0 {
						return tc.sourceResult
					}
					return assignWebMessageString(out, tc.source)
				}),
				TryGetWebMessageAsString: NewComProc(func(_ uintptr, out **uint16) uintptr {
					reads++
					if tc.messageResult != 0 {
						return tc.messageResult
					}
					if tc.nilMessage {
						return 0
					}
					return assignWebMessageString(out, tc.message)
				}),
			}}
			sender := &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{
				PostWebMessageAsString: NewComProc(func(_ uintptr, _ *uint16) uintptr { echoes++; return 0 }),
			}}
			chromium.MessageReceived(sender, args)
			wantCallback := tc.blocked == "" && tc.sourceResult == 0 && tc.messageResult == 0 && !tc.nilMessage
			if (callbacks == 1) != wantCallback || callbacks > 1 || echoes != 0 {
				t.Fatalf("callbacks=%d echoes=%d", callbacks, echoes)
			}
			if tc.blocked == "" && len(blocked) != 0 || tc.blocked != "" && (len(blocked) != 1 || blocked[0] != tc.blocked) {
				t.Fatalf("blocked=%v", blocked)
			}
			if (tc.blocked == "message-source" || tc.sourceResult != 0) && reads != 0 {
				t.Fatal("untrusted message fetched")
			}
		})
	}
}

func assignWebMessageString(out **uint16, value string) uintptr {
	units, err := windows.UTF16FromString(value)
	if err != nil {
		return 0x80070057
	}
	ptr, _, _ := windows.NewLazySystemDLL("ole32.dll").NewProc("CoTaskMemAlloc").Call(uintptr(len(units) * 2))
	if ptr == 0 {
		return 0x8007000e
	}
	*out = (*uint16)(unsafe.Pointer(ptr))
	copy(unsafe.Slice(*out, len(units)), units)
	return 0
}
