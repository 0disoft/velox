package fileopen

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrFolderUnsupported = errors.New("the selected folder is not a supported local directory")
	ErrFolderTarget      = errors.New("the folder connection is no longer valid; select a folder again")
)

type FolderResult struct {
	Cancelled bool   `json:"cancelled"`
	Name      string `json:"name"`
	Target    uint32 `json:"target"`
}

// DirectoryID is host-private identity, not an IPC result or a persisted grant.
type DirectoryID struct{ Volume, IndexHigh, IndexLow uint32 }

type FolderSnapshot struct {
	Name string
	ID   DirectoryID
}

type folderTarget struct {
	id         uint32
	path       string
	generation uint64
	identity   DirectoryID
}

type Folder struct {
	Dispatch   func(func())
	Active     func() bool
	Generation func() uint64
	Choose     func() (string, error)
	Inspect    func(string) (FolderSnapshot, error)
	Read       func(string, DirectoryID) (FolderListing, error)
	pending    atomic.Bool
	mu         sync.Mutex
	serial     uint32
	revision   uint64
	target     *folderTarget
}

func (f *Folder) queue(work func(func() bool, uint64, uint64)) error {
	if f.Dispatch == nil || f.Active == nil || f.Generation == nil || !f.Active() {
		return ErrInactive
	}
	if !f.pending.CompareAndSwap(false, true) {
		return ErrBusy
	}
	generation := f.Generation()
	f.mu.Lock()
	revision := f.revision
	f.mu.Unlock()
	active := func() bool { return f.Active() && f.Generation() == generation }
	f.Dispatch(func() {
		defer f.pending.Store(false)
		work(active, generation, revision)
	})
	return nil
}

func (f *Folder) Select(done func(FolderResult, error)) error {
	if done == nil || f.Choose == nil || f.Inspect == nil {
		return ErrInactive
	}
	return f.queue(func(active func() bool, generation, revision uint64) {
		if !active() {
			done(FolderResult{}, ErrInactive)
			return
		}
		path, err := f.Choose()
		if !active() {
			done(FolderResult{}, ErrInactive)
			return
		}
		if err != nil {
			done(FolderResult{}, err)
			return
		}
		if path == "" {
			done(FolderResult{Cancelled: true}, nil)
			return
		}
		snapshot, err := f.Inspect(path)
		if !active() {
			done(FolderResult{}, ErrInactive)
			return
		}
		if err != nil {
			done(FolderResult{}, err)
			return
		}
		f.mu.Lock()
		if f.revision != revision {
			err = ErrFolderTarget
		} else if f.serial == ^uint32(0) {
			err = ErrBusy
		} else {
			f.serial++
			f.target = &folderTarget{id: f.serial, path: path, generation: generation, identity: snapshot.ID}
		}
		result := FolderResult{Name: snapshot.Name, Target: f.serial}
		f.mu.Unlock()
		if err != nil {
			done(FolderResult{}, err)
			return
		}
		done(result, nil)
	})
}

func (f *Folder) ReleaseTarget(target uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.target == nil || target == 0 || f.target.id != target || f.Generation == nil || f.target.generation != f.Generation() {
		return ErrFolderTarget
	}
	f.target = nil
	f.revision++
	return nil
}

func (f *Folder) ClearTarget() {
	f.mu.Lock()
	f.target = nil
	f.revision++
	f.mu.Unlock()
}
