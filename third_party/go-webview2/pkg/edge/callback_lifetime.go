//go:build windows

package edge

import (
	"runtime"
	"sync"
	"unsafe"
)

type callbackLifetime struct {
	pins      runtime.Pinner
	refs      uintptr
	ownerHeld bool
}

// Native pointers are invisible to the Go collector. Keep both the owner and
// its Pinner reachable until Destroy and every native Release have completed.
var callbackLifetimes = struct {
	sync.Mutex
	owners map[*Chromium]*callbackLifetime
}{owners: make(map[*Chromium]*callbackLifetime)}

func (e *Chromium) callbackPointers() []unsafe.Pointer {
	return []unsafe.Pointer{
		unsafe.Pointer(e.envCompleted),
		unsafe.Pointer(e.controllerCompleted),
		unsafe.Pointer(e.webMessageReceived),
		unsafe.Pointer(e.permissionRequested),
		unsafe.Pointer(e.webResourceRequested),
		unsafe.Pointer(e.acceleratorKeyPressed),
		unsafe.Pointer(e.navigationCompleted),
		unsafe.Pointer(e.navigationStarting),
		unsafe.Pointer(e.frameNavigation),
		unsafe.Pointer(e.newWindowRequested),
		unsafe.Pointer(e.downloadStarting),
	}
}

func (e *Chromium) retainCallbackOwner() bool {
	callbackLifetimes.Lock()
	defer callbackLifetimes.Unlock()
	if _, exists := callbackLifetimes.owners[e]; exists || e.destroyed {
		return false
	}
	lifetime := &callbackLifetime{refs: 1, ownerHeld: true}
	lifetime.pins.Pin(e)
	for _, pointer := range e.callbackPointers() {
		lifetime.pins.Pin(pointer)
		// The first word of every callback is its native-readable vtable.
		// Other fields are used only by Go callback thunks, never by WebView2.
		lifetime.pins.Pin(*(*unsafe.Pointer)(pointer))
	}
	callbackLifetimes.owners[e] = lifetime
	return true
}

// All callback interfaces share the Chromium allocation lifetime. Counts may
// retain unrelated handlers longer, but cannot free any live native interface.
func (e *Chromium) AddRef() uintptr {
	callbackLifetimes.Lock()
	defer callbackLifetimes.Unlock()
	if lifetime := callbackLifetimes.owners[e]; lifetime != nil {
		lifetime.refs++
		return lifetime.refs
	}
	return 0
}

func (e *Chromium) Release() uintptr {
	callbackLifetimes.Lock()
	defer callbackLifetimes.Unlock()
	return e.releaseCallbackReference()
}

func (e *Chromium) releaseCallbackOwner() {
	callbackLifetimes.Lock()
	defer callbackLifetimes.Unlock()
	if lifetime := callbackLifetimes.owners[e]; lifetime != nil && lifetime.ownerHeld {
		lifetime.ownerHeld = false
		e.releaseCallbackReference()
	}
}

// Caller holds callbackLifetimes.Mutex; no COM calls occur under that lock.
func (e *Chromium) releaseCallbackReference() uintptr {
	lifetime := callbackLifetimes.owners[e]
	if lifetime == nil {
		return 0
	}
	lifetime.refs--
	if lifetime.refs == 0 {
		lifetime.pins.Unpin()
		delete(callbackLifetimes.owners, e)
	}
	return lifetime.refs
}
