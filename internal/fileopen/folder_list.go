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
		selected, err := f.connectedTarget(target, generation, revision)
		if err != nil {
			done(FolderListing{}, err)
			return
		}
		result, err := f.Read(selected.path, selected.identity)
		if !active() {
			done(FolderListing{}, ErrInactive)
			return
		}
		if _, targetErr := f.connectedTarget(target, generation, revision); targetErr != nil {
			done(FolderListing{}, targetErr)
			return
		}
		if err != nil {
			done(FolderListing{}, err)
			return
		}
		done(result, nil)
	})
}
