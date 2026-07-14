package benchmarker

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTimelineRecorderWritesOrderedMonotonicPhases(t *testing.T) {
	current := time.Unix(0, 0)
	recorder := newTimelineRecorder(true, func() time.Time { return current })
	current = current.Add(1250 * time.Microsecond)
	recorder.Mark("config-loaded")
	current = current.Add(2 * time.Millisecond)
	recorder.Mark("dom-2raf")

	var output bytes.Buffer
	if err := recorder.Emit(&output); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	line := strings.TrimSpace(output.String())
	if !strings.HasPrefix(line, TimelinePrefix) {
		t.Fatalf("timeline prefix missing: %q", line)
	}
	var timeline StartupTimeline
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, TimelinePrefix)), &timeline); err != nil {
		t.Fatalf("decode timeline: %v", err)
	}
	if timeline.SchemaVersion != TimelineSchemaVersion || timeline.Clock != "time-since-host-entry-monotonic" {
		t.Fatalf("timeline metadata = %#v", timeline)
	}
	want := []TimelinePhase{
		{Name: "host-entry", ElapsedMS: 0},
		{Name: "config-loaded", ElapsedMS: 1.25},
		{Name: "dom-2raf", ElapsedMS: 3.25},
	}
	if len(timeline.Phases) != len(want) {
		t.Fatalf("phase count = %d, want %d", len(timeline.Phases), len(want))
	}
	for index := range want {
		if timeline.Phases[index] != want[index] {
			t.Fatalf("phase %d = %#v, want %#v", index, timeline.Phases[index], want[index])
		}
	}
}

func TestDisabledTimelineRecorderProducesNoOutput(t *testing.T) {
	recorder := NewTimelineRecorder(false)
	recorder.Mark("config-loaded")
	var output bytes.Buffer
	if err := recorder.Emit(&output); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled timeline output = %q", output.String())
	}
}
