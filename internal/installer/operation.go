package installer

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
)

var errOperationInProgress = errors.New("installation or removal is already in progress; retry after it finishes")

func acquireOperation(root, appID string) (func(), error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	release, err := acquireOperationKey(strings.ToLower(filepath.Clean(absolute)) + "\x00" + appID)
	if err != nil {
		return nil, err
	}
	var once sync.Once
	return func() { once.Do(release) }, nil
}
