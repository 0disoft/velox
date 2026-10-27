//go:build windows

package webview2

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/jchv/go-webview2/internal/w32"
)

func TestDPIScale(t *testing.T) {
	for _, tc := range []struct {
		value int32
		dpi   uint32
		want  int32
	}{
		{800, 96, 800}, {800, 120, 1000}, {800, 144, 1200}, {800, 192, 1600}, {101, 120, 126}, {800, 0, 800}, {0, 144, 0},
	} {
		if got := scaleForDPI(tc.value, tc.dpi); got != tc.want {
			t.Fatalf("scale(%d,%d)=%d want %d", tc.value, tc.dpi, got, tc.want)
		}
	}
}

type dpiBrowser struct {
	bindingResponseBrowser
	resizes int
	moves   int
}

func (b *dpiBrowser) Resize()                                  { b.resizes++ }
func (b *dpiBrowser) NotifyParentWindowPositionChanged() error { b.moves++; return nil }

func TestDPIChangedAppliesSuggestedBounds(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	setContext := dpiUser32.NewProc("SetThreadDpiAwarenessContext")
	previous, _, _ := setContext.Call(^uintptr(3))
	if previous == 0 {
		t.Skip("per-monitor V2 unavailable")
	}
	defer setContext.Call(previous)
	b := &dpiBrowser{}
	w := &webview{browser: b}
	if !w.CreateWithOptions(WindowOptions{Width: 300, Height: 200}) {
		t.Fatal("create test window")
	}
	defer func() { deleteWindowContext(w.hwnd); w32.User32DestroyWindow.Call(w.hwnd) }()
	b.resizes = 0
	r := w32.Rect{Left: -100, Top: 40, Right: 500, Bottom: 440}
	sendMessage := dpiUser32.NewProc("SendMessageW")
	sendMessage.Call(w.hwnd, wmDPIChanged, 144|144<<16, uintptr(unsafe.Pointer(&r)))
	var actual w32.Rect
	dpiUser32.NewProc("GetWindowRect").Call(w.hwnd, uintptr(unsafe.Pointer(&actual)))
	if actual != r || b.resizes == 0 || b.moves == 0 {
		t.Fatalf("rect=%+v resize=%d move=%d", actual, b.resizes, b.moves)
	}
	w.minsz = w32.Point{X: 200, Y: 100}
	limits := w32.MinMaxInfo{
		PtReserved: w32.Point{X: 1, Y: 2}, PtMaxSize: w32.Point{X: 1200, Y: 900},
		PtMaxPosition: w32.Point{X: -10, Y: 20}, PtMaxTrackSize: w32.Point{X: 1400, Y: 1000},
	}
	want := limits
	want.PtMinTrackSize = scalePointForDPI(w.minsz, windowDPI(w.hwnd))
	sendMessage.Call(w.hwnd, w32.WMGetMinMaxInfo, 0, uintptr(unsafe.Pointer(&limits)))
	if limits != want {
		t.Fatalf("window limits=%+v, want %+v", limits, want)
	}
}

func TestWindowMessageRejectsUnreadablePayload(t *testing.T) {
	b := &dpiBrowser{}
	w := &webview{hwnd: 1, browser: b, minsz: w32.Point{X: 200, Y: 100}}
	setWindowContext(w.hwnd, w)
	defer deleteWindowContext(w.hwnd)
	for _, message := range []uintptr{wmDPIChanged, w32.WMGetMinMaxInfo} {
		for _, address := range []uintptr{0, 1, ^uintptr(0)} {
			if result := wndproc(w.hwnd, message, 0, address); result != 0 {
				t.Fatal("invalid payload was not ignored")
			}
		}
	}
	if b.resizes != 0 || b.moves != 0 {
		t.Fatal("invalid payload changed window state")
	}
}

func TestWindowReturnsOpaqueHandle(t *testing.T) {
	w := &webview{hwnd: 1}
	if got := w.Window(); got != 1 {
		t.Fatal("handle changed")
	}
}
