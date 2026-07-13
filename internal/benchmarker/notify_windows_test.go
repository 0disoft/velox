package benchmarker

import "testing"

func TestNotifyReadyValidatesLifecycleMarker(t *testing.T) {
	t.Setenv(PipeEnvironment, "")

	if err := NotifyReady("dom-2raf", 42); err != nil {
		t.Fatalf("NotifyReady(valid) error = %v", err)
	}
	if err := NotifyReady("load", 42); err == nil {
		t.Fatal("NotifyReady accepted an unexpected phase")
	}
	if err := NotifyReady("dom-2raf", 0); err == nil {
		t.Fatal("NotifyReady accepted a zero browser process ID")
	}
}
