package benchmarker

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	ShutdownTimelinePrefix        = "actutum-bench-shutdown-timeline "
	ShutdownTimelineSchemaVersion = "actutum.host-shutdown-timeline/v1"
)

type ShutdownTimeline struct {
	SchemaVersion string          `json:"schemaVersion"`
	Clock         string          `json:"clock"`
	Phases        []TimelinePhase `json:"phases"`
}

type ShutdownTimelineRecorder struct {
	mu      sync.Mutex
	enabled bool
	started time.Time
	now     func() time.Time
	phases  []TimelinePhase
}

func NewShutdownTimelineRecorder(enabled bool) *ShutdownTimelineRecorder {
	return newShutdownTimelineRecorder(enabled, time.Now)
}

func newShutdownTimelineRecorder(enabled bool, now func() time.Time) *ShutdownTimelineRecorder {
	return &ShutdownTimelineRecorder{enabled: enabled, now: now}
}

func (r *ShutdownTimelineRecorder) Mark(name string) {
	if r == nil || !r.enabled {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.now()
	if len(r.phases) == 0 {
		r.started = current
		r.phases = append(r.phases, TimelinePhase{Name: name, ElapsedMS: 0})
		return
	}
	r.phases = append(r.phases, TimelinePhase{
		Name:      name,
		ElapsedMS: float64(current.Sub(r.started)) / float64(time.Millisecond),
	})
}

func (r *ShutdownTimelineRecorder) Emit(writer io.Writer) error {
	if r == nil || !r.enabled {
		return nil
	}
	r.mu.Lock()
	timeline := ShutdownTimeline{
		SchemaVersion: ShutdownTimelineSchemaVersion,
		Clock:         "time-since-shutdown-request-monotonic",
		Phases:        append([]TimelinePhase(nil), r.phases...),
	}
	r.mu.Unlock()
	if len(timeline.Phases) == 0 {
		return nil
	}
	body, err := json.Marshal(timeline)
	if err != nil {
		return fmt.Errorf("encode shutdown timeline: %w", err)
	}
	if _, err := fmt.Fprintf(writer, "%s%s\n", ShutdownTimelinePrefix, body); err != nil {
		return fmt.Errorf("write shutdown timeline: %w", err)
	}
	return nil
}
