//go:build !windows

package assettree

import "os"

func isLinkOrReparse(_ string, info os.FileInfo) (bool, error) {
	return info.Mode()&os.ModeSymlink != 0, nil
}
