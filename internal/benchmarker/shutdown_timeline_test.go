package benchmarker

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestShutdownTimelineStartsAtFirstPhase(t *testing.T) {
	current := time.Unix(0, 0)
	recorder := newShutdownTimelineRecorder(true, func() time.Time { return current })
	current = current.Add(time.Second)
	recorder.Mark("shutdown-requested")
	current = current.Add(1250 * time.Microsecond)
	recorder.Mark("environment-released")

	var output bytes.Buffer
	if err := recorder.Emit(&output); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	line := strings.TrimSpace(output.String())
	if !strings.HasPrefix(line, ShutdownTimelinePrefix) {
		t.Fatalf("shutdown timeline prefix missing: %q", line)
	}
	var timeline ShutdownTimeline
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, ShutdownTimelinePrefix)), &timeline); err != nil {
		t.Fatalf("decode shutdown timeline: %v", err)
	}
	if timeline.SchemaVersion != ShutdownTimelineSchemaVersion || timeline.Clock != "time-since-shutdown-request-monotonic" {
		t.Fatalf("shutdown timeline metadata = %#v", timeline)
	}
	want := []TimelinePhase{{Name: "shutdown-requested", ElapsedMS: 0}, {Name: "environment-released", ElapsedMS: 1.25}}
	if len(timeline.Phases) != len(want) {
		t.Fatalf("phase count = %d, want %d", len(timeline.Phases), len(want))
	}
	for index := range want {
		if timeline.Phases[index] != want[index] {
			t.Fatalf("phase %d = %#v, want %#v", index, timeline.Phases[index], want[index])
		}
	}
}

func TestDisabledShutdownTimelineProducesNoOutput(t *testing.T) {
	recorder := NewShutdownTimelineRecorder(false)
	recorder.Mark("shutdown-requested")
	var output bytes.Buffer
	if err := recorder.Emit(&output); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled shutdown timeline output = %q", output.String())
	}
}
