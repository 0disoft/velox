package webview2

import "testing"

func TestMinimumDPIAndSmallWorkArea(t *testing.T) {
	work := windowRect{0, 0, 1920, 1040}
	for _, test := range []struct {
		dpi  uint32
		want minimumPoint
	}{
		{96, minimumPoint{640, 480}}, {120, minimumPoint{800, 600}}, {144, minimumPoint{960, 720}}, {0, minimumPoint{640, 480}},
	} {
		if got := scaledWindowMinimum(640, 480, test.dpi, work); got != test.want {
			t.Fatalf("dpi %d: %+v", test.dpi, got)
		}
	}
	if got := scaledWindowMinimum(640, 480, 144, windowRect{-500, 0, 0, 400}); got != (minimumPoint{500, 400}) {
		t.Fatalf("tiny display = %+v", got)
	}
	if got := scaledWindowMinimum(0, 480, 120, work); got != (minimumPoint{0, 600}) {
		t.Fatal(got)
	}
	if got := scaledWindowMinimum(640, 0, 120, work); got != (minimumPoint{800, 0}) {
		t.Fatal(got)
	}
}

func TestFitMinimumKeepsWindowUsable(t *testing.T) {
	for _, test := range []struct {
		rect, work, want windowRect
		width, height    uint
		dpi              uint32
	}{
		{windowRect{10, 20, 410, 320}, windowRect{0, 0, 1920, 1040}, windowRect{10, 20, 810, 620}, 640, 480, 120},
		{windowRect{-50, -50, 350, 250}, windowRect{0, 0, 500, 400}, windowRect{0, 0, 500, 400}, 640, 480, 144},
		{windowRect{-1000, 500, -500, 900}, windowRect{-1200, 0, 0, 800}, windowRect{-1000, 320, -360, 800}, 640, 480, 96},
		{windowRect{10, 20, 1010, 720}, windowRect{0, 0, 1920, 1040}, windowRect{10, 20, 1010, 720}, 640, 480, 96},
		{windowRect{10, 20, 410, 320}, windowRect{0, 0, 1920, 1040}, windowRect{10, 20, 650, 320}, 640, 0, 96},
	} {
		if got := fitWindowMinimum(test.rect, test.work, test.width, test.height, test.dpi); got != test.want {
			t.Fatalf("fit %+v = %+v, want %+v", test.rect, got, test.want)
		}
	}
}
