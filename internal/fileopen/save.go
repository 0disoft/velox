package fileopen

import (
	"errors"
	"strings"
	"sync/atomic"
	"unicode/utf8"
)

var ErrRecovery = errors.New("save failed; recovery files were retained in the selected folder")

type SaveResult struct {
	Cancelled bool   `json:"cancelled"`
	Name      string `json:"name"`
	Bytes     int    `json:"bytes"`
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
	Dispatch   func(func())
	Active     func() bool
	Generation func() uint64
	Select     func(string) (string, error)
	Write      func(string, string) (SaveResult, error)
	pending    atomic.Bool
}

func (s *Saver) Save(text, name string, done func(SaveResult, error)) error {
	if err := ValidateSaveName(name); err != nil {
		return err
	}
	if err := ValidateSaveText(text); err != nil {
		return err
	}
	if done == nil || s.Dispatch == nil || s.Active == nil || s.Generation == nil || s.Select == nil || s.Write == nil || !s.Active() {
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
		path, err := s.Select(name)
		if !active() {
			done(SaveResult{}, ErrInactive)
			return
		}
		if err != nil {
			done(SaveResult{}, err)
			return
		}
		if path == "" {
			done(SaveResult{Cancelled: true}, nil)
			return
		}
		result, err := s.Write(path, text)
		done(result, err)
	})
	return nil
}
