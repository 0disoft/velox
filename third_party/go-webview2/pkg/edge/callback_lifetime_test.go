//go:build windows

package edge

import (
	"runtime"
	"testing"
	"unsafe"
)

func callbackReferenceCount(e *Chromium) uintptr {
	callbackLifetimes.Lock()
	defer callbackLifetimes.Unlock()
	if lifetime := callbackLifetimes.owners[e]; lifetime != nil {
		return lifetime.refs
	}
	return 0
}

func TestNativeCallbackReferencesOutliveDestroy(t *testing.T) {
	e := NewChromium()
	if !e.retainCallbackOwner() || e.retainCallbackOwner() {
		t.Fatal("callback owner must be retained exactly once")
	}
	pointers := e.callbackPointers()
	if len(pointers) != 11 {
		t.Fatal("expected all eleven native callback objects")
	}
	for i, pointer := range pointers {
		vtbl := *(**_IUnknownVtbl)(pointer)
		count, _, _ := vtbl.AddRef.Call(uintptr(pointer))
		if count != uintptr(i+2) {
			t.Fatalf("callback %d did not retain its owner: %d", i, count)
		}
	}
	e.Destroy()
	e.Destroy()
	runtime.GC()
	if got := callbackReferenceCount(e); got != uintptr(len(pointers)) {
		t.Fatalf("Destroy released native references: %d", got)
	}
	for i, pointer := range pointers {
		vtbl := *(**_IUnknownVtbl)(pointer)
		count, _, _ := vtbl.Release.Call(uintptr(pointer))
		if want := uintptr(len(pointers) - i - 1); count != want {
			t.Fatalf("callback %d release: got %d, want %d", i, count, want)
		}
	}
	if callbackReferenceCount(e) != 0 || e.retainCallbackOwner() {
		t.Fatal("final release must remove the root without allowing resurrection")
	}
	runtime.KeepAlive(pointers)
}

func TestOwnerRetainsCallbacksUntilNativeTeardown(t *testing.T) {
	e := NewChromium()
	e.retainCallbackOwner()
	pointer := unsafe.Pointer(e.envCompleted)
	vtbl := *(**_IUnknownVtbl)(pointer)
	vtbl.AddRef.Call(uintptr(pointer))
	vtbl.Release.Call(uintptr(pointer))
	runtime.GC()
	if callbackReferenceCount(e) != 1 {
		t.Fatal("temporary native reference must not release the owner's pins")
	}
	e.Destroy()
	if callbackReferenceCount(e) != 0 {
		t.Fatal("owner-only lifetime leaked")
	}
}
