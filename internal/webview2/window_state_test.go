package webview2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleWindowState() windowState {
	return windowState{Version: 1, AppID: "dev.velox.state-test", Normal: windowRect{100, 80, 900, 680},
		Work: windowRect{0, 0, 1920, 1040}, DPI: 96, Maximized: true}
}

func TestWindowStateRoundTripAndReplacement(t *testing.T) {
	profile := t.TempDir()
	state := sampleWindowState()
	for _, maximized := range []bool{true, false} {
		state.Maximized = maximized
		if err := saveWindowState(profile, state); err != nil {
			t.Fatal(err)
		}
		got, err := loadWindowState(profile, state.AppID)
		if err != nil || got != state {
			t.Fatalf("loaded = %+v, %v; want %+v", got, err, state)
		}
	}
	files, _ := os.ReadDir(profile)
	if len(files) != 1 || files[0].Name() != windowStateFile {
		t.Fatalf("unexpected profile files: %v", files)
	}
}

func TestWindowStateRejectsInvalidRecords(t *testing.T) {
	for _, body := range []string{"{", "{}", strings.Repeat(" ", maxWindowStateBytes+1),
		`{"version":2,"appId":"dev.velox.state-test","normal":{"Left":0,"Top":0,"Right":800,"Bottom":600},"work":{"Left":0,"Top":0,"Right":1920,"Bottom":1040},"dpi":96}`} {
		profile := t.TempDir()
		if err := os.WriteFile(filepath.Join(profile, windowStateFile), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadWindowState(profile, "dev.velox.state-test"); err == nil {
			t.Fatalf("accepted invalid record %q", body[:min(len(body), 80)])
		}
	}
	profile := t.TempDir()
	state := sampleWindowState()
	if _, err := loadWindowState(profile, state.AppID); err == nil {
		t.Fatal("accepted missing record")
	}
	if err := saveWindowState(profile, state); err != nil {
		t.Fatal(err)
	}
	if _, err := loadWindowState(profile, "dev.velox.other"); err == nil {
		t.Fatal("accepted foreign app state")
	}
	state.DPI = 0
	if err := saveWindowState(profile, state); err == nil {
		t.Fatal("accepted zero DPI")
	}
}

func TestWindowStateRejectsEscapingProfileLink(t *testing.T) {
	profile, outside := t.TempDir(), filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(profile, windowStateFile)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := loadWindowState(profile, sampleWindowState().AppID); err == nil {
		t.Fatal("loaded escaping link")
	}
	// Rename replaces the link itself rather than following it to an external document.
	if err := saveWindowState(profile, sampleWindowState()); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(outside)
	if err != nil || string(body) != "untouched" {
		t.Fatalf("outside file changed: %q, %v", body, err)
	}
}

func TestWindowStateFitDPIWorkAreaAndBounds(t *testing.T) {
	state := sampleWindowState()
	got := state.fit(windowRect{0, 0, 2560, 1400}, 144)
	if got != (windowRect{150, 120, 1350, 1020}) {
		t.Fatalf("150%% geometry = %+v", got)
	}
	state.Work = windowRect{0, 40, 1920, 1080}
	state.Normal = windowRect{100, 120, 900, 720}
	got = state.fit(windowRect{-1920, 0, 0, 1040}, 96)
	if got != (windowRect{-1820, 80, -1020, 680}) {
		t.Fatalf("work-area offset = %+v", got)
	}
	state.Normal = windowRect{3000, 2000, 7000, 5000}
	got = state.fit(windowRect{0, 0, 1280, 720}, 144)
	if got != (windowRect{0, 0, 1280, 720}) {
		t.Fatalf("offscreen large window = %+v", got)
	}
	state.Normal = windowRect{0, 40, 100, 90}
	got = state.fit(windowRect{0, 0, 1920, 1040}, 144)
	if got.Right-got.Left != 480 || got.Bottom-got.Top != 360 {
		t.Fatalf("minimum logical size = %+v", got)
	}
}
