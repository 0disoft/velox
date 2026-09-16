package webview2

import "errors"

var ErrInitializationCanceled = errors.New("webview initialization canceled by user")
var ErrInitializationFailed = errors.New("webview initialization failed")

func initializationResultError(canceled bool, failure error) error {
	if failure != nil {
		return failure
	}
	if canceled {
		return ErrInitializationCanceled
	}
	return ErrInitializationFailed
}
