//go:build !windows

package installer

import "os"

func replaceStateFile(source, target string) error {
	return os.Rename(source, target)
}
