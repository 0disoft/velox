//go:build windows

package edge

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestCOMInitializationDiagnostic(t *testing.T) {
	for _, tc := range []struct {
		result uintptr
		code   string
	}{
		{0, ""},
		{1, ""},
		{0x80010106, "80010106"},
		{0x80004005, "80004005"},
		{0x800401f0, "800401f0"},
	} {
		var output bytes.Buffer
		reportCOMInitialization(tc.result, log.New(&output, "", 0))
		if tc.code == "" {
			if output.Len() != 0 {
				t.Errorf("successful HRESULT 0x%08x logged a warning: %s", tc.result, &output)
			}
			continue
		}
		if !strings.Contains(output.String(), "CoInitializeEx call failed: E="+tc.code) {
			t.Errorf("failure HRESULT 0x%08x produced no matching warning: %q", tc.result, output.String())
		}
	}
}
