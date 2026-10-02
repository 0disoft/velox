package webview2

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"
)

const windowStateFile = "velox-window-state.json"
const maxWindowStateBytes = 4096

type windowRect struct {
	Left, Top, Right, Bottom int32
}

type windowState struct {
	Version   int        `json:"version"`
	AppID     string     `json:"appId"`
	Normal    windowRect `json:"normal"`
	Work      windowRect `json:"work"`
	DPI       uint32     `json:"dpi"`
	Maximized bool       `json:"maximized"`
}

func (r windowRect) valid() bool {
	return r.Left >= -1000000 && r.Top >= -1000000 && r.Right <= 1000000 && r.Bottom <= 1000000 &&
		r.Right > r.Left && r.Bottom > r.Top && r.Right-r.Left <= 100000 && r.Bottom-r.Top <= 100000
}

func (s windowState) valid(appID string) bool {
	return s.Version == 1 && s.AppID == appID && s.DPI >= 48 && s.DPI <= 768 && s.Normal.valid() && s.Work.valid()
}

func loadWindowState(profile, appID string) (windowState, error) {
	root, err := os.OpenRoot(profile)
	if err != nil {
		return windowState{}, err
	}
	defer root.Close()
	file, err := root.Open(windowStateFile)
	if err != nil {
		return windowState{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxWindowStateBytes {
		return windowState{}, errors.New("invalid window state file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxWindowStateBytes+1))
	if err != nil || len(data) > maxWindowStateBytes {
		return windowState{}, errors.New("window state exceeds size limit")
	}
	var state windowState
	if err := json.Unmarshal(data, &state); err != nil || !state.valid(appID) {
		return windowState{}, errors.New("invalid window state")
	}
	return state, nil
}

func saveWindowState(profile string, state windowState) error {
	if !state.valid(state.AppID) {
		return errors.New("invalid window state")
	}
	root, err := os.OpenRoot(profile)
	if err != nil {
		return err
	}
	defer root.Close()
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if len(data) > maxWindowStateBytes {
		return errors.New("window state exceeds size limit")
	}
	name := "velox-window-state-" + rand.Text() + ".tmp"
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer root.Remove(name)
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return root.Rename(name, windowStateFile)
}

// Normalize screen coordinates relative to the old work area, not the desktop origin.
func (s windowState) fit(work windowRect, dpi uint32) windowRect {
	scale := func(value int32) int32 { return int32((int64(value)*int64(dpi) + int64(s.DPI)/2) / int64(s.DPI)) }
	width := min(max(scale(s.Normal.Right-s.Normal.Left), int32(320*dpi/96)), work.Right-work.Left)
	height := min(max(scale(s.Normal.Bottom-s.Normal.Top), int32(240*dpi/96)), work.Bottom-work.Top)
	left := min(max(work.Left+scale(s.Normal.Left-s.Work.Left), work.Left), work.Right-width)
	top := min(max(work.Top+scale(s.Normal.Top-s.Work.Top), work.Top), work.Bottom-height)
	return windowRect{left, top, left + width, top + height}
}
