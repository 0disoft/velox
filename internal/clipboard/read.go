package clipboard

import "sync/atomic"

type ReadResult struct {
	Cancelled bool    `json:"cancelled"`
	Text      *string `json:"text,omitempty"`
}

type Reader struct {
	Dispatch   func(func())
	Active     func() bool
	Generation func() uint64
	Confirm    func() (bool, error)
	Read       func() (string, error)
	pending    atomic.Bool
}

// ReadText defers confirmation and reads only after approval in the same document.
func (r *Reader) ReadText(done func(ReadResult, error)) error {
	if done == nil || r.Dispatch == nil || r.Active == nil || r.Generation == nil || r.Confirm == nil || r.Read == nil || !r.Active() {
		return ErrInactive
	}
	if !r.pending.CompareAndSwap(false, true) {
		return ErrPending
	}
	generation := r.Generation()
	active := func() bool { return r.Active() && r.Generation() == generation }
	r.Dispatch(func() {
		defer r.pending.Store(false)
		if !active() {
			done(ReadResult{}, ErrInactive)
			return
		}
		approved, err := r.Confirm()
		if !active() {
			done(ReadResult{}, ErrInactive)
			return
		}
		if err != nil {
			done(ReadResult{}, err)
			return
		}
		if !approved {
			done(ReadResult{Cancelled: true}, nil)
			return
		}
		text, err := r.Read()
		if !active() {
			done(ReadResult{}, ErrInactive)
			return
		}
		if err == nil {
			err = Validate(text)
		}
		if err != nil {
			done(ReadResult{}, err)
			return
		}
		done(ReadResult{Text: &text}, nil)
	})
	return nil
}
