package webview2

import "testing"

func TestFixedWindowStateFit(t *testing.T) {
	s := sampleWindowState()
	s.Normal = windowRect{1700, 900, 3100, 2000}
	work := windowRect{-1920, 0, 0, 1040}
	for _, dpi := range []uint32{96, 120, 144} {
		got := s.fitSize(work, dpi, 480, 360)
		if got.Right-got.Left != int32(480*dpi/96) || got.Bottom-got.Top != int32(360*dpi/96) {
			t.Fatal(dpi, got)
		}
		if got.Left < work.Left || got.Top < work.Top || got.Right > work.Right || got.Bottom > work.Bottom {
			t.Fatal("outside work area", got)
		}
	}
	got := s.fitSize(windowRect{0, 0, 300, 200}, 144, 480, 360)
	if got != (windowRect{0, 0, 300, 200}) {
		t.Fatal("small work area", got)
	}
	if s.fitSize(work, 144, 0, 0) != s.fit(work, 144) {
		t.Fatal("resizable behavior changed")
	}
}
