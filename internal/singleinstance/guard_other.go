//go:build !windows

package singleinstance

import "errors"

type Guard struct{}

func Acquire(appID, profile string) (*Guard, bool, error) {
	return nil, false, errors.New("single instance is only supported on Windows")
}

func (g *Guard) Close()    {}
func (g *Guard) Activate() {}
func (g *Guard) Attach(hwnd uintptr) error {
	return errors.New("single instance is only supported on Windows")
}
