package fileopen

const (
	MaxFolderEntries     = 128
	MaxFolderResultBytes = 32 << 10
)

type FolderEntry struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type FolderListing struct {
	Entries   []FolderEntry `json:"entries"`
	Truncated bool          `json:"truncated"`
	Skipped   int           `json:"skipped"`
}

func (f *Folder) List(target uint32, done func(FolderListing, error)) error {
	if done == nil || f.Read == nil {
		return ErrInactive
	}
	return f.queue(func(active func() bool, generation, revision uint64) {
		if !active() {
			done(FolderListing{}, ErrInactive)
			return
		}
		f.mu.Lock()
		connected := f.target
		if connected == nil || target == 0 || connected.id != target || connected.generation != generation || f.revision != revision {
			f.mu.Unlock()
			done(FolderListing{}, ErrFolderTarget)
			return
		}
		selected := *connected
		f.mu.Unlock()
		result, err := f.Read(selected.path, selected.identity)
		if !active() {
			done(FolderListing{}, ErrInactive)
			return
		}
		f.mu.Lock()
		revoked := f.revision != revision || f.target == nil || f.target.id != target
		f.mu.Unlock()
		if revoked {
			done(FolderListing{}, ErrFolderTarget)
			return
		}
		if err != nil {
			done(FolderListing{}, err)
			return
		}
		done(result, nil)
	})
}
