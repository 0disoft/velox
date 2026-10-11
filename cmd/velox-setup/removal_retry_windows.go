package main

import "fmt"

func retryRemoval(self, appID string, silent bool, remove func() error, ask func(string) bool) error {
	for {
		err := remove()
		if err == nil {
			return nil
		}
		detail := fmt.Errorf("%w\n\nTo retry removal after resolving this issue, run:\n\"%s\" --uninstall %s", err, self, appID)
		if silent || !ask(detail.Error()+"\n\nRetry now?") {
			return detail
		}
	}
}
