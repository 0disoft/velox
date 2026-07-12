//go:build !windows

package webview2

func InstalledVersion() (string, error) {
	return "", nil
}
