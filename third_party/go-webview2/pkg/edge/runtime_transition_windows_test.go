package edge

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestRuntimeTransitionRetainsKnownTargetWhenDiscoveryFails(t *testing.T) {
	e := NewChromium()
	e.BrowserExecutableFolder = filepath.Join(t.TempDir(), "missing-runtime")
	e.runtimeTransition.required = "112.0.1722.68"
	e.runtimeTransition.deadline = time.Now().Add(runtimeTransitionTimeout)
	deadline := e.runtimeTransition.deadline
	e.prepareRuntimeTransition()
	if e.runtimeTransition.required != "112.0.1722.68" || e.runtimeTransition.deadline != deadline {
		t.Fatal("failed discovery abandoned or extended an active transition")
	}
	e.runtimeTransition.deadline = time.Time{}
	e.prepareRuntimeTransition()
	if e.runtimeTransition.required != "" || e.InitializationError() != nil {
		t.Fatal("initial discovery failure must retain the SDK initialization path")
	}
}

func TestRequestedRuntimeVersionDiscovery(t *testing.T) {
	for _, tc := range []struct {
		version, want string
		err           error
	}{
		{"112.0.1722.68", "112.0.1722.68", nil},
		{" 112.0.1722.68 beta ", "112.0.1722.68", nil},
		{"", "", nil},
		{"111.0.1661.62", "", errors.New("unavailable")},
	} {
		got := requestedRuntimeVersion("fixed-folder", func(folder string) (string, error) {
			if folder != "fixed-folder" {
				t.Fatal("version lookup used another runtime folder")
			}
			return tc.version, tc.err
		})
		if got != tc.want {
			t.Fatalf("version=%q error=%v: got %q, want %q", tc.version, tc.err, got, tc.want)
		}
	}
}

func TestRuntimeTransitionDeadlineBoundaries(t *testing.T) {
	now := time.Now()
	transition := runtimeTransition{}
	if transition.expired(now) || transition.retryDue(now) {
		t.Fatal("same-runtime startup must not acquire a wait")
	}
	transition.deadline = now.Add(runtimeTransitionTimeout)
	transition.retryAt = now.Add(runtimeTransitionPoll)
	for _, tc := range []struct {
		boundary time.Time
		due      func(time.Time) bool
	}{
		{transition.deadline, transition.expired},
		{transition.retryAt, transition.retryDue},
	} {
		if tc.due(tc.boundary.Add(-time.Nanosecond)) || !tc.due(tc.boundary) || !tc.due(tc.boundary.Add(time.Nanosecond)) {
			t.Fatal("wait boundary must fire at and after its deadline")
		}
	}
}

func TestRuntimeTransitionDoesNotCreateControllerForWrongVersion(t *testing.T) {
	for _, tc := range []struct {
		version, required string
		hr                uintptr
		wantController    bool
		wantError         bool
	}{
		{"112.0.1722.68", "112.0.1722.68", 0, true, false},
		{"112.0.1722.68 beta", "112.0.1722.68", 0, true, false},
		{"111.0.1661.62", "112.0.1722.68", 0, false, false},
		{"", "112.0.1722.68", 0, false, true},
		{"112.0.1722.68", "112.0.1722.68", 0x80004005, false, true},
	} {
		e := NewChromium()
		e.runtimeTransition.required = tc.required
		// Avoid creating a real timer here. Native tests cover its lifetime.
		e.runtimeTransition.deadline = time.Now().Add(runtimeTransitionTimeout)
		deadline := e.runtimeTransition.deadline
		controllers, refs := 0, 0
		vtbl := iCoreWebView2EnvironmentVtbl{
			_IUnknownVtbl: _IUnknownVtbl{
				AddRef:  NewComProc(func(uintptr) uintptr { refs++; return 1 }),
				Release: NewComProc(func(uintptr) uintptr { refs--; return 0 }),
			},
			GetBrowserVersionString: NewComProc(func(_ uintptr, out *uintptr) uintptr {
				if tc.hr != 0 {
					return tc.hr
				}
				input := windows.StringToUTF16Ptr(tc.version)
				hr, _, _ := duplicateEnvironmentString.Call(uintptr(unsafe.Pointer(input)), uintptr(unsafe.Pointer(out)))
				return hr
			}),
			CreateCoreWebView2Controller: NewComProc(func(uintptr, uintptr, uintptr) uintptr { controllers++; return 0 }),
		}
		env := ICoreWebView2Environment{vtbl: &vtbl}
		e.EnvironmentCompleted(0, &env)
		if (controllers == 1) != tc.wantController || (e.InitializationError() != nil) != tc.wantError {
			t.Fatalf("version=%q: controllers=%d error=%v", tc.version, controllers, e.InitializationError())
		}
		if !tc.wantController && !tc.wantError && (e.runtimeTransition.retryAt.IsZero() || e.environment != nil || refs != 0) {
			t.Fatal("wrong-runtime environment was retained or did not schedule a retry")
		}
		if e.runtimeTransition.deadline != deadline {
			t.Fatal("retry extended the total transition deadline")
		}
		e.Destroy()
		if refs != 0 {
			t.Fatal("environment reference leaked")
		}
	}
}
