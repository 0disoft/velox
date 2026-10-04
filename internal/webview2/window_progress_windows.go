//go:build windows

package webview2

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const progressSubclassID = 0x1f19
const progressReplayMessage = 0x8120

var progressOnce sync.Once
var progressCallback, progressCreatedMessage uintptr
var progressPost = user32Window.NewProc("PostMessageW")
var progressOwners = struct {
	sync.Mutex
	items map[uintptr]*progressOwner
}{items: make(map[uintptr]*progressOwner)}

type progressClient interface {
	state(uintptr, uintptr) error
	value(uintptr, uint32) error
	close()
}

type progressValue struct {
	state string
	value uint32
}

type progressOwner struct {
	create                                                func() (progressClient, error)
	client                                                progressClient
	desired                                               progressValue
	ready, applied, updating, closed, reset, replayPosted bool
}

func progressFlag(state string) (uintptr, bool) {
	switch state {
	case "none":
		return 0, true
	case "indeterminate":
		return 1, true
	case "normal":
		return 2, true
	case "error":
		return 4, true
	case "paused":
		return 8, true
	default:
		return 0, false
	}
}

func installTaskbarProgress(hwnd uintptr) error {
	if err := taskbarCreate.Find(); err != nil {
		return fmt.Errorf("resolve taskbar COM creation: %w", err)
	}
	return installProgressOwner(hwnd, createTaskbarClient)
}

func installProgressOwner(hwnd uintptr, create func() (progressClient, error)) error {
	if err := progressPost.Find(); err != nil {
		return err
	}
	if valid, _, _ := isNativeWindow.Call(hwnd); valid == 0 {
		return errors.New("progress window is unavailable")
	}
	progressOnce.Do(func() {
		progressCallback = windows.NewCallback(progressWindowProc)
		name, _ := windows.UTF16PtrFromString("TaskbarButtonCreated")
		progressCreatedMessage, _, _ = user32Window.NewProc("RegisterWindowMessageW").Call(uintptr(unsafe.Pointer(name)))
	})
	if progressCreatedMessage == 0 {
		return errors.New("register taskbar button message")
	}
	owner := &progressOwner{create: create, desired: progressValue{state: "none"}}
	progressOwners.Lock()
	if progressOwners.items[hwnd] != nil {
		progressOwners.Unlock()
		return errors.New("taskbar progress already installed")
	}
	progressOwners.items[hwnd] = owner
	progressOwners.Unlock()
	if ok, _, err := stateSetSubclass.Call(hwnd, progressCallback, progressSubclassID, 0); ok == 0 {
		progressOwners.Lock()
		delete(progressOwners.items, hwnd)
		progressOwners.Unlock()
		return fmt.Errorf("install taskbar progress handler: %v", err)
	}
	return nil
}

func progressOwnerFor(hwnd uintptr) *progressOwner {
	progressOwners.Lock()
	defer progressOwners.Unlock()
	return progressOwners.items[hwnd]
}

func (w nativeWindow) SetProgress(state string, value uint32) error {
	hwnd, err := w.handle()
	if err != nil {
		return err
	}
	owner := progressOwnerFor(hwnd)
	if owner == nil {
		return errors.New("taskbar progress is unavailable")
	}
	return owner.update(hwnd, progressValue{state: state, value: value})
}

func (owner *progressOwner) release() {
	if owner.client != nil {
		client := owner.client
		owner.client = nil
		client.close()
	}
}

func (owner *progressOwner) dispose(hwnd uintptr) {
	if owner.client != nil {
		// Best effort; the window or Explorer may already be disappearing.
		_ = owner.client.state(hwnd, 0)
		owner.release()
	}
}

func (owner *progressOwner) update(hwnd uintptr, next progressValue) error {
	flag, valid := progressFlag(next.state)
	if !valid || next.value > 100 || ((next.state == "none" || next.state == "indeterminate") && next.value != 0) {
		return errors.New("invalid taskbar progress")
	}
	if owner.closed || owner.updating {
		return errors.New("taskbar progress is unavailable")
	}
	if !owner.ready {
		owner.desired = next
		return nil
	}
	owner.updating = true
	defer func() {
		owner.updating = false
		if owner.closed {
			owner.dispose(hwnd)
		}
	}()
	if owner.reset {
		owner.reset, owner.applied = false, false
		owner.release()
		if owner.closed {
			return errors.New("progress window closed during reset")
		}
	}
	if owner.applied && owner.desired == next {
		return nil
	}
	if owner.client == nil && next.state != "none" {
		client, err := owner.create()
		if err != nil {
			return err
		}
		if client == nil {
			return errors.New("taskbar interface is unavailable")
		}
		owner.client = client
		if owner.closed {
			return errors.New("progress window closed during creation")
		}
	}
	if owner.client != nil {
		if next.state == "normal" || next.state == "error" || next.state == "paused" {
			if err := owner.client.value(hwnd, next.value); err != nil {
				owner.applied = false
				return err
			}
			if owner.closed {
				return errors.New("progress window closed during update")
			}
		}
		if err := owner.client.state(hwnd, flag); err != nil {
			owner.applied = false
			return err
		}
		if owner.closed {
			return errors.New("progress window closed during update")
		}
	}
	owner.desired, owner.applied = next, true
	return nil
}

func progressWindowProc(hwnd, message, wparam, lparam, subclassID, reference uintptr) uintptr {
	if subclassID != progressSubclassID || reference != 0 {
		result, _, _ := stateDefSubclass.Call(hwnd, message, wparam, lparam)
		return result
	}
	owner := progressOwnerFor(hwnd)
	if owner != nil {
		switch message {
		case 0x82: // WM_NCDESTROY: release on the same UI/COM thread.
			progressOwners.Lock()
			delete(progressOwners.items, hwnd)
			progressOwners.Unlock()
			owner.closed = true
			if !owner.updating {
				owner.dispose(hwnd)
			}
			stateRemoveSubclass.Call(hwnd, progressCallback, progressSubclassID)
		case progressCreatedMessage, progressReplayMessage:
			if message == progressReplayMessage {
				owner.replayPosted = false
			} else {
				owner.ready, owner.reset = true, true
			}
			if owner.updating {
				if !owner.replayPosted {
					owner.replayPosted = true
					if ok, _, _ := progressPost.Call(hwnd, progressReplayMessage, 0, 0); ok == 0 {
						owner.replayPosted = false
					}
				}
			} else if err := owner.update(hwnd, owner.desired); err != nil {
				fmt.Fprintln(os.Stderr, "velox-host: taskbar progress restoration failed")
			}
		}
	}
	result, _, _ := stateDefSubclass.Call(hwnd, message, wparam, lparam)
	return result
}
