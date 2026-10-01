//go:build !windows

package installer

import "errors"

func userEnvironment() (environment, error) {
	return environment{}, errors.New("per-user installation requires Windows")
}
