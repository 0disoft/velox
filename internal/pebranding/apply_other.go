//go:build !windows

package pebranding

import "errors"

func Apply(path, filename string, options Options) error {
	if !options.Enabled {
		return nil
	}
	return errors.New("executable branding requires Windows")
}
