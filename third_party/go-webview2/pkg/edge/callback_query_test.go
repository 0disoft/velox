//go:build windows

package edge

import (
	"testing"
	"unsafe"
)

func TestNativeCallbackQueryInterfaceOwnsReturnedPointer(t *testing.T) {
	e := NewChromium()
	e.retainCallbackOwner()
	defer e.Destroy()
	iids := []*GUID{environmentCompletedIID, controllerCompletedIID, webMessageReceivedIID,
		permissionRequestedIID, webResourceRequestedIID, acceleratorKeyPressedIID,
		navigationCompletedIID, navigationStartingIID, navigationStartingIID,
		newWindowRequestedIID, downloadStartingIID}
	for index, pointer := range e.callbackPointers() {
		vtbl := *(**_IUnknownVtbl)(pointer)
		for _, iid := range []*GUID{callbackIUnknownIID, iids[index]} {
			var out unsafe.Pointer
			result, _, _ := vtbl.QueryInterface.Call(uintptr(pointer), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
			if result != 0 || out != pointer || callbackReferenceCount(e) != 2 {
				t.Fatalf("callback %d returned wrong identity or ownership: HRESULT=%x", index, result)
			}
			vtbl.Release.Call(uintptr(out))
		}
		unsupported := &GUID{}
		out := pointer
		result, _, _ := vtbl.QueryInterface.Call(uintptr(pointer), uintptr(unsafe.Pointer(unsupported)), uintptr(unsafe.Pointer(&out)))
		if result != 0x80004002 || out != nil || callbackReferenceCount(e) != 1 {
			t.Fatalf("callback %d accepted unsupported interface or retained a stale out pointer", index)
		}
		result, _, _ = vtbl.QueryInterface.Call(uintptr(pointer), uintptr(unsafe.Pointer(iids[index])), 0)
		if result != 0x80004003 || callbackReferenceCount(e) != 1 {
			t.Fatalf("callback %d accepted a missing output pointer", index)
		}
	}
}
