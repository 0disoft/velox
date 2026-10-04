package webview2

type minimumPoint struct{ X, Y int32 }

func scaledWindowMinimum(width, height uint, dpi uint32, work windowRect) minimumPoint {
	if dpi < 48 || dpi > 768 {
		dpi = 96
	}
	scale := func(value uint, available int32) int32 {
		return min(int32((int64(value)*int64(dpi)+48)/96), available)
	}
	return minimumPoint{scale(width, work.Right-work.Left), scale(height, work.Bottom-work.Top)}
}

func fitWindowMinimum(rect, work windowRect, width, height uint, dpi uint32) windowRect {
	limit := scaledWindowMinimum(width, height, dpi, work)
	w := min(max(rect.Right-rect.Left, limit.X), work.Right-work.Left)
	h := min(max(rect.Bottom-rect.Top, limit.Y), work.Bottom-work.Top)
	x := min(max(rect.Left, work.Left), work.Right-w)
	y := min(max(rect.Top, work.Top), work.Bottom-h)
	return windowRect{x, y, x + w, y + h}
}
