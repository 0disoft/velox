package webview2

import "testing"

func TestAppThemeHighContrastPriority(t *testing.T) {
	for _, test := range []struct {
		contrast bool
		light    uint64
		dark     bool
	}{
		{false, 0, true}, {false, 1, false}, {false, 2, false}, {true, 0, false}, {true, 1, false},
	} {
		if got := appThemeIsDark(test.contrast, test.light); got != test.dark {
			t.Fatal(test, got)
		}
	}
}
