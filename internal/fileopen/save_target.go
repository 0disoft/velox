package fileopen

import (
	"crypto/sha256"
	"errors"
)

var (
	ErrTarget   = errors.New("the save target is no longer connected; use Save as")
	ErrConflict = errors.New("the connected file changed outside the app; use Save as without discarding your text")
)

// FileVersion is private host state, never an IPC result or a persisted grant.
type FileVersion struct {
	Volume, IndexHigh, IndexLow uint32
	Size                        uint64
	Modified                    uint64
	Digest                      [32]byte
}

type saveTarget struct {
	id         uint32
	path       string
	generation uint64
	version    FileVersion
}

func (s *Saver) SaveAs(text, name string, done func(SaveResult, error)) error {
	s.mu.Lock()
	exhausted := s.targetSerial == ^uint32(0)
	s.mu.Unlock()
	if exhausted {
		return ErrBusy
	}
	return s.saveAs(text, name, true, done)
}

func (s *Saver) connectResult(path, text string, generation uint64, result SaveResult, active func() bool) (SaveResult, error) {
	version, err := s.Snapshot(path)
	if err != nil {
		return SaveResult{}, err
	}
	if version.Size != uint64(len(text)) || version.Digest != sha256.Sum256([]byte(text)) {
		return SaveResult{}, ErrConflict
	}
	if !active() {
		return SaveResult{}, ErrInactive
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targetSerial++
	s.target = &saveTarget{id: s.targetSerial, path: path, generation: generation, version: version}
	result.Target = s.targetSerial
	return result, nil
}

func (s *Saver) SaveTo(text string, target uint32, done func(SaveResult, error)) error {
	if s.WriteVersion == nil || s.Snapshot == nil {
		return ErrInactive
	}
	return s.queueSave(text, done, func(active func() bool, generation uint64) (SaveResult, error) {
		s.mu.Lock()
		connected := s.target
		if connected == nil || target == 0 || connected.id != target || connected.generation != generation {
			s.mu.Unlock()
			return SaveResult{}, ErrTarget
		}
		selected := *connected
		s.mu.Unlock()
		result, err := s.WriteVersion(selected.path, text, selected.version)
		if err != nil {
			return SaveResult{}, err
		}
		version, err := s.Snapshot(selected.path)
		if err != nil {
			return SaveResult{}, err
		}
		if version.Size != uint64(len(text)) || version.Digest != sha256.Sum256([]byte(text)) {
			return SaveResult{}, ErrConflict
		}
		if !active() {
			return SaveResult{}, ErrInactive
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.target == nil || s.target.id != target {
			return SaveResult{}, ErrTarget
		}
		s.target.version = version
		result.Target = target
		return result, nil
	})
}

func (s *Saver) ClearTarget() { s.mu.Lock(); s.target = nil; s.mu.Unlock() }

func (s *Saver) ReleaseTarget(target uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.target == nil || target == 0 || s.target.id != target {
		return ErrTarget
	}
	s.target = nil
	return nil
}
