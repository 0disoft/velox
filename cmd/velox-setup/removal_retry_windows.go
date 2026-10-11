package main

import (
	"fmt"
	"strings"
)

func retryRemoval(self, appID string, silent bool, remove func() error, ask func(string) bool) error {
	for {
		err := remove()
		if err == nil {
			return nil
		}
		detail := fmt.Errorf("%w\n\nTo retry removal after resolving this issue, run in PowerShell:\n& '%s' --uninstall %s", err, strings.ReplaceAll(self, "'", "''"), appID)
		if silent || !ask(detail.Error()+"\n\nRetry now?") {
			return detail
		}
	}
}
