package webview2

import (
	"errors"
	"testing"
)

func TestInitializationResultPreservesFailure(t *testing.T) {
	failure := errors.New("controller failure")
	for _, canceled := range []bool{false, true} {
		if got := initializationResultError(canceled, failure); got != failure {
			t.Fatalf("failure replaced when canceled=%v: %v", canceled, got)
		}
	}
	if got := initializationResultError(true, nil); got != ErrInitializationCanceled {
		t.Fatalf("user cancellation not preserved: %v", got)
	}
	if got := initializationResultError(false, nil); got != ErrInitializationFailed {
		t.Fatalf("unclassified failure became success: %v", got)
	}
}
