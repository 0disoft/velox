package benchmarker

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	TimelinePrefix        = "velox-bench-timeline "
	TimelineSchemaVersion = "velox.host-startup-timeline/v1"
)

type TimelinePhase struct {
	Name      string  `json:"name"`
	ElapsedMS float64 `json:"elapsedMs"`
}

type StartupTimeline struct {
	SchemaVersion string          `json:"schemaVersion"`
	Clock         string          `json:"clock"`
	Phases        []TimelinePhase `json:"phases"`
}

type TimelineRecorder struct {
	mu      sync.Mutex
	enabled bool
	started time.Time
	now     func() time.Time
	phases  []TimelinePhase
}

func NewTimelineRecorder(enabled bool) *TimelineRecorder {
	return newTimelineRecorder(enabled, time.Now)
}

func newTimelineRecorder(enabled bool, now func() time.Time) *TimelineRecorder {
	started := now()
	recorder := &TimelineRecorder{enabled: enabled, started: started, now: now}
	if enabled {
		recorder.phases = append(recorder.phases, TimelinePhase{Name: "host-entry", ElapsedMS: 0})
	}
	return recorder
}

func (r *TimelineRecorder) Mark(name string) {
	if r == nil || !r.enabled {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.phases = append(r.phases, TimelinePhase{
		Name:      name,
		ElapsedMS: float64(r.now().Sub(r.started)) / float64(time.Millisecond),
	})
}

func (r *TimelineRecorder) Emit(writer io.Writer) error {
	if r == nil || !r.enabled {
		return nil
	}
	r.mu.Lock()
	timeline := StartupTimeline{
		SchemaVersion: TimelineSchemaVersion,
		Clock:         "time-since-host-entry-monotonic",
		Phases:        append([]TimelinePhase(nil), r.phases...),
	}
	r.mu.Unlock()
	body, err := json.Marshal(timeline)
	if err != nil {
		return fmt.Errorf("encode startup timeline: %w", err)
	}
	if _, err := fmt.Fprintf(writer, "%s%s\n", TimelinePrefix, body); err != nil {
		return fmt.Errorf("write startup timeline: %w", err)
	}
	return nil
}
