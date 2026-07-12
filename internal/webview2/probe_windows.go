//go:build windows

package webview2

import "github.com/jchv/go-webview2/webviewloader"

func InstalledVersion() (string, error) {
	return webviewloader.GetInstalledVersion()
}
