package startup_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func testBuiltHostEarlyClose(t *testing.T) {
	host := goHost(t, repositoryRoot(t))
	user32 := windows.NewLazySystemDLL("user32.dll")
	for attempt := 0; attempt < 3; attempt++ {
		profile := managedProfileRoot(t, "velox-early-close-")
		name := fmt.Sprintf(`\\.\pipe\velox-early-%d-%d`, os.Getpid(), time.Now().UnixNano())
		pipe, err := createPipe(name)
		if err != nil {
			t.Fatal(err)
		}
		func() {
			defer windows.CloseHandle(pipe)
			defer cancelIoEx.Call(uintptr(pipe), 0)
			ready := make(chan bool, 1)
			go func() {
				if acceptPipe(pipe) != nil {
					return
				}
				buf := make([]byte, 128)
				n, err := windows.Read(pipe, buf)
				if err == nil && n > 0 {
					ready <- true
				}
			}()
			cmd := exec.Command(host.executable, host.arguments(profile)...)
			cmd.Env = append(os.Environ(), host.environment(profile)...)
			cmd.Env = append(cmd.Env, "VELOX_BENCH_MODE=1", "VELOX_BENCH_PIPE="+name, "VELOX_BENCH_EXIT_AFTER_READY=0")
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			waited := false
			defer func() {
				if !waited {
					_ = cmd.Process.Kill()
					<-done
				}
			}()
			deadline := time.Now().Add(10 * time.Second)
			var target uintptr
			for target == 0 && time.Now().Before(deadline) {
				hwnd, _, _ := user32.NewProc("GetTopWindow").Call(0)
				for hwnd != 0 {
					var pid uint32
					user32.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&pid)))
					var class [64]uint16
					user32.NewProc("GetClassNameW").Call(hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
					if pid == uint32(cmd.Process.Pid) && windows.UTF16ToString(class[:]) == "webview" {
						target = hwnd
						break
					}
					hwnd, _, _ = user32.NewProc("GetWindow").Call(hwnd, 2)
				}
				if target == 0 {
					time.Sleep(time.Millisecond)
				}
			}
			if target == 0 {
				t.Fatal("owned webview window not found")
			}
			select {
			case <-ready:
				t.Fatal("ready before early-close request")
			default:
			}
			ok, _, err := user32.NewProc("PostMessageW").Call(target, 0x0010, 0, 0)
			if ok == 0 {
				t.Fatalf("close request failed: %v", err)
			}
			select {
			case err := <-done:
				waited = true
				if err != nil {
					t.Fatalf("early close failed: %v; %s", err, output.String())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("early close exceeded 5 seconds")
			}
			if output.Len() != 0 {
				t.Fatalf("early close emitted unexpected diagnostics: %s", output.String())
			}
			select {
			case <-ready:
				t.Fatal("early-close case reached readiness")
			default:
			}
		}()
		run := mustRunHost(t, host, profile)
		mustAwaitBrowserExit(t, run, 10*time.Second)
		mustWaitForProfileRelease(t, profile, 10*time.Second)
	}
}
