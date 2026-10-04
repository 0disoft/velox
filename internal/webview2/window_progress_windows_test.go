//go:build windows

package webview2

import (
	"errors"
	"testing"
	"time"
	"unsafe"
)

type progressTestClient struct {
	calls    []string
	values   []uint32
	flags    []uintptr
	releases int
	err      error
	onState  func()
}

func (c *progressTestClient) value(_ uintptr, value uint32) error {
	c.calls = append(c.calls, "value")
	c.values = append(c.values, value)
	return c.err
}
func (c *progressTestClient) state(_ uintptr, flag uintptr) error {
	c.calls = append(c.calls, "state")
	c.flags = append(c.flags, flag)
	if c.onState != nil {
		c.onState()
	}
	return c.err
}
func (c *progressTestClient) close() { c.releases++ }

func progressSend(hwnd, message uintptr) {
	stateUser32.NewProc("SendMessageW").Call(hwnd, message, 0, 0)
}

func TestTaskbarProgressReadinessRecreateFailureAndCleanup(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	client := &progressTestClient{}
	created := 0
	if err := installProgressOwner(hwnd, func() (progressClient, error) { created++; return client, nil }); err != nil {
		t.Fatal(err)
	}
	owner := progressOwnerFor(hwnd)
	if created != 0 || owner.ready {
		t.Fatal("installation created COM or assumed readiness")
	}
	if err := owner.update(hwnd, progressValue{"normal", 25}); err != nil {
		t.Fatal(err)
	}
	if created != 0 || len(client.calls) != 0 {
		t.Fatal("called COM before TaskbarButtonCreated")
	}
	progressSend(hwnd, progressCreatedMessage)
	if created != 1 || len(client.calls) != 2 || client.calls[0] != "value" || client.flags[0] != 2 || client.values[0] != 25 {
		t.Fatal(created, client)
	}
	if err := owner.update(hwnd, progressValue{"normal", 25}); err != nil || len(client.calls) != 2 {
		t.Fatal("duplicate was not deduplicated", err)
	}
	for _, tc := range []struct {
		state string
		value uint32
		flag  uintptr
	}{{"error", 50, 4}, {"paused", 60, 8}, {"indeterminate", 0, 1}, {"none", 0, 0}} {
		if err := owner.update(hwnd, progressValue{tc.state, tc.value}); err != nil {
			t.Fatal(err)
		}
		if client.flags[len(client.flags)-1] != tc.flag {
			t.Fatal(tc, client.flags)
		}
	}
	if err := owner.update(hwnd, progressValue{"paused", 60}); err != nil {
		t.Fatal(err)
	}
	before := len(client.calls)
	progressSend(hwnd, progressCreatedMessage)
	if client.releases != 1 || created != 2 || len(client.calls) != before+2 || client.flags[len(client.flags)-1] != 8 {
		t.Fatal("recreation did not restore state", created, client)
	}
	client.err = errors.New("temporary taskbar failure")
	if err := owner.update(hwnd, progressValue{"normal", 80}); err == nil || owner.desired.state != "paused" || owner.applied {
		t.Fatal("failure was cached as applied")
	}
	client.err = nil
	if err := owner.update(hwnd, progressValue{"normal", 80}); err != nil || owner.desired.value != 80 {
		t.Fatal("explicit retry failed", err)
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	if progressOwnerFor(hwnd) != nil || client.releases != 2 || client.flags[len(client.flags)-1] != 0 {
		t.Fatal("destruction did not clear/release", client)
	}
	if err := owner.update(hwnd, progressValue{"normal", 1}); err == nil {
		t.Fatal("closed owner accepted update")
	}
}

func TestTaskbarProgressNoWorkUntilRequestedAndCreationRetry(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	client := &progressTestClient{}
	failed := true
	creates := 0
	if err := installProgressOwner(hwnd, func() (progressClient, error) {
		creates++
		if failed {
			return nil, errors.New("COM unavailable")
		}
		return client, nil
	}); err != nil {
		t.Fatal(err)
	}
	owner := progressOwnerFor(hwnd)
	progressSend(hwnd, progressCreatedMessage)
	if creates != 0 {
		t.Fatal("ready with no progress created COM")
	}
	if err := owner.update(hwnd, progressValue{"normal", 50}); err == nil || owner.client != nil {
		t.Fatal("creation failure ignored")
	}
	failed = false
	if err := owner.update(hwnd, progressValue{"normal", 50}); err != nil || creates != 2 {
		t.Fatal("creation retry failed", creates, err)
	}
	if err := installProgressOwner(hwnd, createTaskbarClient); err == nil {
		t.Fatal("duplicate owner accepted")
	}
	for _, value := range []progressValue{{"other", 0}, {"normal", 101}, {"none", 1}} {
		if err := owner.update(hwnd, value); err == nil {
			t.Fatal("invalid native progress accepted", value)
		}
	}
	if err := installTaskbarProgress(0); err == nil {
		t.Fatal("invalid window accepted")
	}
	if err := (nativeWindow{}).SetProgress("none", 0); err == nil {
		t.Fatal("missing native window accepted")
	}
}

func TestTaskbarProgressReentrantDestroyReleasesOnce(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	client := &progressTestClient{}
	if err := installProgressOwner(hwnd, func() (progressClient, error) { return client, nil }); err != nil {
		t.Fatal(err)
	}
	owner := progressOwnerFor(hwnd)
	progressSend(hwnd, progressCreatedMessage)
	client.onState = func() {
		client.onState = nil
		stateUser32.NewProc("DestroyWindow").Call(hwnd)
		if client.releases != 0 {
			t.Error("released COM during its active call")
		}
	}
	if err := owner.update(hwnd, progressValue{"normal", 50}); err == nil {
		t.Fatal("destroyed window reported success")
	}
	if client.releases != 1 || owner.client != nil || progressOwnerFor(hwnd) != nil {
		t.Fatal("reentrant destruction leaked COM", client)
	}
}

func TestTaskbarProgressReentrantReadinessQueuesRestore(t *testing.T) {
	stateTestThread(t)
	hwnd := stateTestWindow(t)
	client := &progressTestClient{}
	creates := 0
	if err := installProgressOwner(hwnd, func() (progressClient, error) { creates++; return client, nil }); err != nil {
		t.Fatal(err)
	}
	owner := progressOwnerFor(hwnd)
	progressSend(hwnd, progressCreatedMessage)
	client.onState = func() {
		client.onState = nil
		progressSend(hwnd, progressCreatedMessage)
		if client.releases != 0 {
			t.Error("released COM during a reentrant call")
		}
	}
	if err := owner.update(hwnd, progressValue{"error", 40}); err != nil || !owner.replayPosted || !owner.reset {
		t.Fatal("reentrant reset was lost", err)
	}
	var message struct {
		window         uintptr
		id             uint32
		wparam, lparam uintptr
		timestamp      uint32
		x, y           int32
		private        uint32
	}
	if got, _, _ := stateUser32.NewProc("PeekMessageW").Call(uintptr(unsafe.Pointer(&message)), hwnd, progressReplayMessage, progressReplayMessage, 1); got == 0 {
		t.Fatal("restore was not posted")
	}
	stateUser32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&message)))
	if owner.replayPosted || owner.reset || creates != 2 || client.releases != 1 || client.flags[len(client.flags)-1] != 4 {
		t.Fatal("queued reset failed", creates, client)
	}
}

func TestTaskbarProgressNativeCOM(t *testing.T) {
	stateTestThread(t)
	hr, _, _ := taskbarOle32.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		t.Fatalf("CoInitializeEx %#x", hr)
	}
	t.Cleanup(func() { taskbarOle32.NewProc("CoUninitialize").Call() })
	hwnd := stateTestWindow(t)
	if err := installTaskbarProgress(hwnd); err != nil {
		t.Fatal(err)
	}
	stateUser32.NewProc("ShowWindow").Call(hwnd, 8)
	owner := progressOwnerFor(hwnd)
	deadline := time.Now().Add(3 * time.Second)
	// MSG on Windows x64 includes trailing alignment padding.
	var message struct {
		window         uintptr
		id             uint32
		wparam, lparam uintptr
		timestamp      uint32
		x, y           int32
		private        uint32
	}
	for !owner.ready && time.Now().Before(deadline) {
		if ok, _, _ := stateUser32.NewProc("PeekMessageW").Call(uintptr(unsafe.Pointer(&message)), hwnd, progressCreatedMessage, progressCreatedMessage, 1); ok != 0 {
			stateUser32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&message)))
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if !owner.ready {
		t.Skip("Explorer did not create a taskbar button; native COM progress not exercised")
	}
	if unsafe.Offsetof(taskbarVTable{}.SetProgressValue) != 9*unsafe.Sizeof(uintptr(0)) || unsafe.Offsetof(taskbarVTable{}.SetProgressState) != 10*unsafe.Sizeof(uintptr(0)) {
		t.Fatal("ITaskbarList3 ABI mismatch")
	}
	for _, value := range []progressValue{{"normal", 25}, {"error", 50}, {"paused", 75}, {"indeterminate", 0}, {"none", 0}} {
		if err := owner.update(hwnd, value); err != nil {
			t.Fatal(value, err)
		}
	}
	stateUser32.NewProc("DestroyWindow").Call(hwnd)
	if progressOwnerFor(hwnd) != nil || owner.client != nil {
		t.Fatal("native COM retained after destruction")
	}
}
