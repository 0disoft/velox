package externalurl

import (
	"errors"
	"sync/atomic"
)

type Scheduler struct {
	Dispatch      func(func())
	IsClosing     func() bool
	Opener        *Opener
	NotifyFailure func()
	pending       atomic.Bool
}

// Open acknowledges one queued confirmation, not a completed browser launch.
// Running modal UI after the WebView event returns avoids nested event callbacks.
func (s *Scheduler) Open(raw string) error {
	target, err := Validate(raw)
	if err != nil {
		return err
	}
	if s.Dispatch == nil || s.IsClosing == nil || s.Opener == nil || s.IsClosing() {
		return errors.New("external browser is unavailable")
	}
	if !s.pending.CompareAndSwap(false, true) {
		return ErrBusy
	}
	s.Dispatch(func() {
		defer s.pending.Store(false)
		if s.IsClosing() {
			return
		}
		if _, err := s.Opener.Open(target); err != nil && !s.IsClosing() && s.NotifyFailure != nil {
			s.NotifyFailure()
		}
	})
	return nil
}
