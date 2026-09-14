package startup_test

import (
	"testing"

	"github.com/0disoft/velox/internal/benchmarker"
)

func TestShutdownResultRejectsControllerCloseFailure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		phases []string
		failed bool
	}{
		{"closed", []string{"controller-closed", "environment-released"}, false},
		{"failed-but-cleaned", []string{"controller-close-failed", "webview-released", "controller-released", "environment-released"}, true},
		{"failure-not-masked-by-success", []string{"controller-close-failed", "controller-closed"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			timeline := &benchmarker.ShutdownTimeline{}
			for _, name := range tc.phases {
				timeline.Phases = append(timeline.Phases, benchmarker.TimelinePhase{Name: name})
			}
			if err := validateShutdownResult(timeline); (err != nil) != tc.failed {
				t.Fatalf("shutdown error=%v want-failure=%t", err, tc.failed)
			}
		})
	}
}
