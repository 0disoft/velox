package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/0disoft/velox/internal/appidentity"
	"github.com/0disoft/velox/internal/inspector"
	"github.com/0disoft/velox/internal/installer"
	"github.com/0disoft/velox/internal/setuppayload"
	"golang.org/x/sys/windows"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	flags := flag.NewFlagSet("Velox Setup", flag.ContinueOnError)
	silent := flags.Bool("silent", false, "install or uninstall without confirmation dialogs")
	remove := flags.String("uninstall", "", "uninstall this application ID")
	helper := flags.Bool("remove-helper", false, "internal removal helper")
	waitPID := flags.Uint("wait-pid", 0, "internal parent process ID")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	if (*helper && (*remove == "" || *waitPID == 0)) || (!*helper && *waitPID != 0) {
		return 2
	}
	self, err := os.Executable()
	if err != nil {
		return failed(err, *silent)
	}
	if *remove != "" {
		if err := appidentity.Validate(*remove); err != nil {
			return failed(err, *silent)
		}
		if *helper {
			if err := waitForParent(uint32(*waitPID)); err != nil {
				return failed(err, *silent)
			}
			_, err := installer.Uninstall(*remove)
			if err != nil {
				return failed(err, *silent)
			}
			if !*silent {
				message("The application was removed. Documents and recovery data were kept.", "Velox Uninstall", 0x40)
			}
			return 0
		}
		if !*silent && message("Remove "+*remove+"? Documents and recovery data will be kept.", "Velox Uninstall", 0x24) != 6 {
			return 0
		}
		if err := launchRemovalHelper(self, *remove, *silent); err != nil {
			return failed(err, *silent)
		}
		return 0
	}
	if *helper {
		return 2
	}
	payload, err := setuppayload.Open(self)
	if err != nil {
		return failed(err, *silent)
	}
	defer payload.Close()
	work, err := os.MkdirTemp("", "velox-setup-")
	if err != nil {
		return failed(err, *silent)
	}
	defer os.RemoveAll(work)
	source, err := payload.Extract(filepath.Join(work, "payload"))
	if err != nil {
		return failed(err, *silent)
	}
	inspection, err := inspector.Inspect(source)
	if err != nil {
		return failed(err, *silent)
	}
	if !*silent && message("Install "+inspection.App.Name+" "+inspection.App.Version+" for your Windows account?", "Velox Setup", 0x24) != 6 {
		return 0
	}
	uninstaller := filepath.Join(work, "uninstall.exe")
	if err := payload.CopyTemplate(uninstaller); err != nil {
		return failed(err, *silent)
	}
	result, err := installer.Install(source, uninstaller)
	if err != nil {
		return failed(err, *silent)
	}
	fmt.Fprintln(os.Stdout, result.Directory)
	if !*silent {
		message(inspection.App.Name+" is installed. Open it from the Start menu.", "Velox Setup", 0x40)
	}
	return 0
}

func launchRemovalHelper(self, appID string, silent bool) error {
	work, err := os.MkdirTemp("", "velox-uninstall-")
	if err != nil {
		return err
	}
	path := filepath.Join(work, "remove.exe")
	input, err := os.Open(self)
	if err != nil {
		os.RemoveAll(work)
		return err
	}
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		input.Close()
		os.RemoveAll(work)
		return err
	}
	_, copyErr := io.Copy(output, input)
	err = errors.Join(copyErr, input.Close(), output.Close())
	if err != nil {
		os.RemoveAll(work)
		return err
	}
	args := []string{"--uninstall", appID, "--remove-helper", "--wait-pid", strconv.Itoa(os.Getpid())}
	if silent {
		args = append(args, "--silent")
	}
	command := exec.Command(path, args...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		os.RemoveAll(work)
		return err
	}
	if silent {
		fmt.Fprintln(os.Stdout, work)
	}
	return command.Process.Release()
}

func waitForParent(pid uint32) error {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return nil
	}
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	result, err := windows.WaitForSingleObject(handle, 30_000)
	if err != nil {
		return err
	}
	if result != windows.WAIT_OBJECT_0 {
		return errors.New("uninstaller parent did not exit")
	}
	return nil
}

func failed(err error, silent bool) int {
	fmt.Fprintln(os.Stderr, err)
	if !silent {
		message(err.Error(), "Velox Setup", 0x10)
	}
	return 6
}

func message(text, title string, flags uintptr) uintptr {
	body, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return 0
	}
	caption, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return 0
	}
	result, _, _ := windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(caption)), flags)
	return result
}
