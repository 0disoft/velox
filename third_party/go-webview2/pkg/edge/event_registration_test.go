//go:build windows

package edge

import (
	"testing"
	"unsafe"
)

func TestPartialEventRegistrationRemovesOnlySuccessfulTokens(t *testing.T) {
	names := []string{"message", "permission", "resource", "navigation-completed", "accelerator", "navigation", "frame", "popup", "download"}
	for failAt := -1; failAt < len(names); failAt++ {
		name := "success"
		if failAt >= 0 {
			name = names[failAt]
		}
		t.Run(name, func(t *testing.T) {
			e := NewChromium()
			e.NavigationAllowed = func(string) bool { return false }
			e.DenyFrames, e.DenyNewWindows, e.DenyDownloads = true, true, true
			e.retainCallbackOwner()
			defer e.Destroy()
			var added, removed [9]int
			var installed [9]bool
			closeCalls, scriptCalls := 0, 0
			add := func(index int) ComProc {
				return NewComProc(func(_ uintptr, _ uintptr, token *_EventRegistrationToken) uintptr {
					added[index]++
					// A failed API may write its out parameter. It is not a registration.
					token.Value = int64(index)
					if index == failAt {
						return 0x80004005
					}
					installed[index] = true
					e.AddRef()
					return 0
				})
			}
			remove := func(index int) ComProc {
				return NewComProc(func(_ uintptr, token uintptr) uintptr {
					removed[index]++
					if !installed[index] || token != uintptr(index) {
						t.Errorf("removed invalid %s token: %d", names[index], token)
					} else {
						installed[index] = false
						e.Release()
					}
					return 0
				})
			}
			noop := NewComProc(func(uintptr) uintptr { return 1 })
			v4 := &ICoreWebView2_4{vtbl: &iCoreWebView2_4Vtbl{
				AddDownloadStarting: add(8), RemoveDownloadStarting: remove(8),
			}}
			v4.vtbl.Release = noop
			webview := &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{
				_IUnknownVtbl: _IUnknownVtbl{
					Release: noop,
					QueryInterface: NewComProc(func(_ uintptr, _ uintptr, out **ICoreWebView2_4) uintptr {
						*out = v4
						return 0
					}),
				},
				AddWebMessageReceived: add(0), RemoveWebMessageReceived: remove(0),
				AddPermissionRequested: add(1), RemovePermissionRequested: remove(1),
				AddWebResourceRequested: add(2), RemoveWebResourceRequested: remove(2),
				AddNavigationCompleted: add(3), RemoveNavigationCompleted: remove(3),
				AddNavigationStarting: add(5), RemoveNavigationStarting: remove(5),
				AddFrameNavigationStarting: add(6), RemoveFrameNavigationStarting: remove(6),
				AddNewWindowRequested: add(7), RemoveNewWindowRequested: remove(7),
				AddScriptToExecuteOnDocumentCreated: NewComProc(func(_ uintptr, _ uintptr, _ uintptr) uintptr {
					scriptCalls++
					return 0
				}),
			}}
			controller := &ICoreWebView2Controller{vtbl: &_ICoreWebView2ControllerVtbl{
				_IUnknownVtbl: _IUnknownVtbl{AddRef: noop, Release: noop},
				GetCoreWebView2: NewComProc(func(_ uintptr, out **ICoreWebView2) uintptr {
					*out = webview
					return 0
				}),
				Close:                    NewComProc(func(uintptr) uintptr { closeCalls++; return 0 }),
				AddAcceleratorKeyPressed: add(4), RemoveAcceleratorKeyPressed: remove(4),
			}}
			e.CreateCoreWebView2ControllerCompleted(0, controller)
			if got := e.finishInitialization(); got != (failAt < 0) {
				t.Fatalf("initialization success = %v, error = %v", got, e.initializationError)
			}
			e.Destroy()
			e.Destroy()
			for index := range names {
				wantAdd, wantRemove := 0, 0
				if failAt < 0 || index <= failAt {
					wantAdd = 1
				}
				if failAt < 0 || index < failAt {
					wantRemove = 1
				}
				if added[index] != wantAdd || removed[index] != wantRemove {
					t.Errorf("%s: add/remove=%d/%d, want %d/%d", names[index], added[index], removed[index], wantAdd, wantRemove)
				}
			}
			if closeCalls != 1 || callbackReferenceCount(e) != 0 {
				t.Fatal("teardown did not close controller once and release every callback")
			}
			if (failAt < 0 && scriptCalls != 1) || (failAt >= 0 && scriptCalls != 0) {
				t.Fatalf("script calls = %d", scriptCalls)
			}
		})
	}
}

func TestCallbackInvokeAfterDestroyRemainsPinned(t *testing.T) {
	e := NewChromium()
	e.retainCallbackOwner()
	e.AddRef()
	e.Destroy()
	result, _, _ := e.envCompleted.vtbl.Invoke.Call(uintptr(unsafe.Pointer(e.envCompleted)), 0x80004005, 0)
	if result != 0 || e.initializationError != nil {
		t.Fatalf("late callback changed destroyed state: result=%x error=%v", result, e.initializationError)
	}
	e.Release()
}
