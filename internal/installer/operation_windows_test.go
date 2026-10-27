//go:build windows

package installer

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestOperationLeaseAcrossProcessesAndForcedExit(t *testing.T) {
	const variable = "VELOX_INSTALL_OPERATION_TEST_ROOT"
	const appID = "dev.test.operation"
	if root := os.Getenv(variable); root != "" {
		release, err := acquireOperation(root, appID)
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		fmt.Println("locked")
		var input [1]byte
		os.Stdin.Read(input[:])
		return
	}
	root := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestOperationLeaseAcrossProcessesAndForcedExit$")
	command.Env = append(os.Environ(), variable+"="+root)
	command.Stderr = os.Stderr
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.ProcessState == nil {
			command.Process.Kill()
			command.Wait()
		}
	})
	ready := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(output).ReadString('\n')
		if err == nil && line != "locked\n" {
			err = fmt.Errorf("unexpected child output: %q", line)
		}
		ready <- err
	}()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("child did not acquire the operation lease")
	}
	if release, err := acquireOperation(strings.ToUpper(root), appID); !errors.Is(err, errOperationInProgress) {
		if release != nil {
			release()
		}
		t.Fatalf("second process acquired the same operation: %v", err)
	}
	other, err := acquireOperation(root, "dev.test.other")
	if err != nil {
		t.Fatalf("unrelated app was blocked: %v", err)
	}
	other()
	other() // Release is idempotent and must not close a reused native handle.
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	command.Wait() // Forced exit is expected; no child defers execute.
	release, err := acquireOperation(root, appID)
	if err != nil {
		t.Fatalf("operation stayed locked after forced process exit: %v", err)
	}
	release()
}
