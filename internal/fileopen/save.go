package fileopen

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
)

var ErrRecovery = errors.New("save failed; recovery files were retained in the selected folder")

type SaveResult struct {
	Cancelled bool   `json:"cancelled"`
	Name      string `json:"name"`
	Bytes     int    `json:"bytes"`
	Target    uint32 `json:"target,omitempty"`
}

func ValidateSaveName(name string) error {
	if name == "" || len(name) > 240 || !utf8.ValidString(name) || strings.ContainsAny(name, `\/:*?"<>|`) || strings.TrimRight(name, ". ") != name || name == "." || name == ".." {
		return ErrUnsupported
	}
	for _, char := range name {
		if char < 32 {
			return ErrUnsupported
		}
	}
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
		return ErrUnsupported
	}
	return nil
}

func ValidateSaveText(text string) error {
	if len(text) > MaxTextBytes {
		return ErrTooLarge
	}
	if !utf8.ValidString(text) || strings.IndexByte(text, 0) >= 0 {
		return ErrUnsupported
	}
	return nil
}

type Saver struct {
	Dispatch     func(func())
	Active       func() bool
	Generation   func() uint64
	Select       func(string) (string, error)
	Write        func(string, string) (SaveResult, error)
	Snapshot     func(string) (FileVersion, error)
	WriteVersion func(string, string, FileVersion) (SaveResult, error)
	pending      atomic.Bool
	mu           sync.Mutex
	target       *saveTarget
	targetSerial uint32
}

func (s *Saver) Save(text, name string, done func(SaveResult, error)) error {
	return s.saveAs(text, name, false, done)
}

func (s *Saver) saveAs(text, name string, connect bool, done func(SaveResult, error)) error {
	if err := ValidateSaveName(name); err != nil {
		return err
	}
	if s.Select == nil || s.Write == nil {
		return ErrInactive
	}
	if connect && (s.Snapshot == nil || s.WriteVersion == nil) {
		return ErrInactive
	}
	return s.queueSave(text, done, func(active func() bool, generation uint64) (SaveResult, error) {
		path, err := s.Select(name)
		if !active() {
			return SaveResult{}, ErrInactive
		}
		if err != nil {
			return SaveResult{}, err
		}
		if path == "" {
			return SaveResult{Cancelled: true}, nil
		}
		result, err := s.Write(path, text)
		if err == nil && connect {
			result, err = s.connectResult(path, text, generation, result, active)
		}
		return result, err
	})
}

func (s *Saver) queueSave(text string, done func(SaveResult, error), operation func(func() bool, uint64) (SaveResult, error)) error {
	if err := ValidateSaveText(text); err != nil {
		return err
	}
	if done == nil || s.Dispatch == nil || s.Active == nil || s.Generation == nil || !s.Active() {
		return ErrInactive
	}
	if !s.pending.CompareAndSwap(false, true) {
		return ErrBusy
	}
	generation := s.Generation()
	active := func() bool { return s.Active() && s.Generation() == generation }
	s.Dispatch(func() {
		defer s.pending.Store(false)
		if !active() {
			done(SaveResult{}, ErrInactive)
			return
		}
		result, err := operation(active, generation)
		done(result, err)
	})
	return nil
}
