//go:build !windows

package authenticode

import (
	"context"
	"errors"
)

func probeAuthenticode(context.Context, string) (probeResult, error) {
	return probeResult{}, errors.New("Authenticode verification requires Windows")
}
