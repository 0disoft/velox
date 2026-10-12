package fileopen

func (f *Folder) OpenText(target uint32, name string, done func(Result, error)) error {
	if err := ValidateFolderTextName(name); err != nil {
		return err
	}
	if done == nil || f.ReadText == nil {
		return ErrInactive
	}
	return f.queue(func(active func() bool, generation, revision uint64) {
		if !active() {
			done(Result{}, ErrInactive)
			return
		}
		selected, err := f.connectedTarget(target, generation, revision)
		if err != nil {
			done(Result{}, err)
			return
		}
		result, err := f.ReadText(selected.path, selected.identity, name)
		if !active() {
			done(Result{}, ErrInactive)
			return
		}
		if _, targetErr := f.connectedTarget(target, generation, revision); targetErr != nil {
			done(Result{}, targetErr)
			return
		}
		if err != nil {
			done(Result{}, err)
			return
		}
		done(result, nil)
	})
}
