//go:build windows

package externalurl

import "testing"

func TestNativeOpenerRejectsMissingOwnerWithoutUI(t *testing.T) {
	opened, err := NewWindows(0, nil).Open("https://example.com")
	if opened || err == nil {
		t.Fatal("native confirmation accepted a missing owner")
	}
	NotifyWindowsFailure(0)
}
