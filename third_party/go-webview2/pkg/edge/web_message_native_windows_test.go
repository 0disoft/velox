package edge

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

func TestNativeWebMessageBoundaries(t *testing.T) {
	if os.Getenv("VELOX_NATIVE_WEB_MESSAGES") != "1" {
		t.Skip("opt-in installed WebView2 raw-message test")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := w32.Ole32CoInitializeEx.Call(0, 2)
	if err := hresult(hr); err != nil {
		t.Fatal(err)
	}
	defer windows.NewLazySystemDLL("ole32.dll").NewProc("CoUninitialize").Call()
	class, _ := windows.UTF16PtrFromString("STATIC")
	hwnd, _, err := w32.User32CreateWindowExW.Call(0, uintptr(unsafe.Pointer(class)), 0, w32.WSOverlappedWindow, 0, 0, 320, 240, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatal(err)
	}
	defer w32.User32DestroyWindow.Call(hwnd)
	chromium := NewChromium()
	chromium.DataPath = filepath.Join(t.TempDir(), "profile")
	chromium.MaxWebMessageBytes = 64 << 10
	var browser windows.Handle
	defer func() {
		chromium.Destroy()
		if browser != 0 {
			defer windows.CloseHandle(browser)
			state, err := windows.WaitForSingleObject(browser, 10000)
			if err != nil || state != windows.WAIT_OBJECT_0 {
				t.Errorf("owned browser cleanup: %x %v", state, err)
			}
		}
	}()
	if !chromium.Embed(hwnd) {
		t.Fatalf("Embed: %v", chromium.InitializationError())
	}
	pid, err := chromium.BrowserProcessID()
	if err != nil {
		t.Fatal(err)
	}
	browser, err = windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		t.Fatal(err)
	}
	var accepted []string
	blocked := 0
	result := ""
	chromium.MessageSourceAllowed = func(source string) bool { return source == "about:blank" }
	chromium.PolicyBlocked = func(kind string) {
		if kind == "message-size" {
			blocked++
		} else {
			t.Errorf("unexpected policy block: %s", kind)
		}
	}
	chromium.MessageCallback = func(message string) {
		switch {
		case message == "barrier":
			chromium.Eval(`setTimeout(() => chrome.webview.postMessage("result:" + window.echoCount), 100)`)
		case strings.HasPrefix(message, "result:"):
			result = message
		default:
			accepted = append(accepted, message)
		}
	}
	chromium.NavigateToString(`<!doctype html><script>
window.echoCount = 0;
chrome.webview.addEventListener("message", () => { window.echoCount++; });
const valid = ["", "normal", "a".repeat(65536), "\ud55c".repeat(21845) + "a", "\ud83d\ude00".repeat(16384), "\n\t\r".repeat(21845) + "a"];
const invalid = ["a".repeat(65537), "\ud55c".repeat(21845) + "ab", "\ud83d\ude00".repeat(16384) + "a", "\n\t\r".repeat(21845) + "ab", "a".repeat(2*1024*1024)];
for (const value of valid) chrome.webview.postMessage(value);
for (const value of invalid) chrome.webview.postMessage(value);
chrome.webview.postMessage("barrier");
</script>`)
	if !pumpNativeCancellationUntil(10*time.Second, func() bool { return result != "" }) {
		t.Fatal("native message result timed out")
	}
	want := []string{"", "normal", strings.Repeat("a", 65536), strings.Repeat("\ud55c", 21845) + "a", strings.Repeat("\U0001f600", 16384), strings.Repeat("\n\t\r", 21845) + "a"}
	if !slices.Equal(accepted, want) || blocked != 5 || result != "result:0" {
		t.Fatalf("accepted=%d blocked=%d result=%q", len(accepted), blocked, result)
	}
	t.Log("real WebView2: six exact/in-budget messages preserved; five raw oversized messages blocked; zero echo events")
}
