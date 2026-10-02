package fileopen

import (
	"errors"
	"sync/atomic"
)

const MaxTextBytes = 2 << 20

var (
	ErrBusy        = errors.New("a file selection is already pending")
	ErrTooLarge    = errors.New("the selected file exceeds 2 MiB")
	ErrUnsupported = errors.New("the selected file is not supported local UTF-8 text")
	ErrInactive    = errors.New("the requesting document is no longer active")
)

type Result struct {
	Cancelled bool   `json:"cancelled"`
	Name      string `json:"name"`
	Text      string `json:"text"`
	Bytes     int    `json:"bytes"`
}

type Opener struct {
	Dispatch   func(func())
	Active     func() bool
	Generation func() uint64
	Select     func() (string, error)
	Read       func(string) (Result, error)
	pending    atomic.Bool
}

// Open queues modal UI outside the WebView callback. Only the native selection
// supplies a path; no application-supplied path or reusable file grant is kept.
func (o *Opener) Open(done func(Result, error)) error {
	if done == nil || o.Dispatch == nil || o.Active == nil || o.Generation == nil || o.Select == nil || o.Read == nil || !o.Active() {
		return ErrInactive
	}
	if !o.pending.CompareAndSwap(false, true) {
		return ErrBusy
	}
	generation := o.Generation()
	active := func() bool { return o.Active() && o.Generation() == generation }
	o.Dispatch(func() {
		defer o.pending.Store(false)
		if !active() {
			done(Result{}, ErrInactive)
			return
		}
		path, err := o.Select()
		if !active() {
			done(Result{}, ErrInactive)
			return
		}
		if err != nil {
			done(Result{}, err)
			return
		}
		if path == "" {
			done(Result{Cancelled: true}, nil)
			return
		}
		result, err := o.Read(path)
		if !active() {
			done(Result{}, ErrInactive)
			return
		}
		done(result, err)
	})
	return nil
}
