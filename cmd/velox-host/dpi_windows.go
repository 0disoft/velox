//go:build windows

package main

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

func enablePerMonitorDPI() error {
	user32 := windows.NewLazySystemDLL("user32.dll")
	set := user32.NewProc("SetProcessDpiAwarenessContext")
	get := user32.NewProc("GetThreadDpiAwarenessContext")
	awareness := user32.NewProc("GetAwarenessFromDpiAwarenessContext")
	return configureDPI(func(context uintptr) error {
		ok, _, err := set.Call(context)
		if ok == 0 {
			return err
		}
		return nil
	}, func() int {
		context, _, _ := get.Call()
		value, _, _ := awareness.Call(context)
		return int(int32(value))
	})
}

func configureDPI(set func(uintptr) error, current func() int) error {
	// Server 2016 supports per-monitor V1, but not the V2 context.
	err := set(^uintptr(3))
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		err = set(^uintptr(2))
	}
	if err == nil || (errors.Is(err, windows.ERROR_ACCESS_DENIED) && current() == 2) {
		return nil
	}
	return fmt.Errorf("enable per-monitor DPI awareness: %w", err)
}
