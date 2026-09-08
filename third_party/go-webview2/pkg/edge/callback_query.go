//go:build windows

package edge

import "unsafe"

var (
	callbackIUnknownIID      = NewGUID("00000000-0000-0000-C000-000000000046")
	environmentCompletedIID  = NewGUID("4E8A3389-C9D8-4BD2-B6B5-124FEE6CC14D")
	controllerCompletedIID   = NewGUID("6C4819F3-C9B7-4260-8127-C9F5BDE7F68C")
	webMessageReceivedIID    = NewGUID("57213F19-00E6-49FA-8E07-898EA01ECBD2")
	permissionRequestedIID   = NewGUID("15E1C6A3-C72A-4DF3-91D7-D097FBEC6BFD")
	webResourceRequestedIID  = NewGUID("AB00B74C-15F1-4646-80E8-E76341D25D71")
	acceleratorKeyPressedIID = NewGUID("B29C7E28-FA79-41A8-8E44-65811C76DCB2")
	navigationCompletedIID   = NewGUID("D33A35BF-1C49-4F98-93AB-006E0533FE1C")
	navigationStartingIID    = NewGUID("9ADBE429-F36D-432B-9DDC-F8881FBD76E3")
	newWindowRequestedIID    = NewGUID("D4C185FE-C81C-4989-97AF-2D3FA7AB5651")
	downloadStartingIID      = NewGUID("EFEDC989-C396-41CA-83F7-07F845A55724")
)

func queryCallbackInterface(impl _IUnknownImpl, self unsafe.Pointer, requested *GUID, out *unsafe.Pointer, supported *GUID) uintptr {
	if out == nil {
		return 0x80004003 // E_POINTER
	}
	*out = nil
	if requested == nil {
		return 0x80004003
	}
	if *requested != *callbackIUnknownIID && *requested != *supported {
		return 0x80004002 // E_NOINTERFACE, including IAgileObject on these STA handlers.
	}
	if impl.AddRef() == 0 {
		return 0x8000ffff // E_UNEXPECTED: the callback has no live owner.
	}
	*out = self
	return 0
}
