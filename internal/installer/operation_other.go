//go:build !windows

package installer

import "sync"

// Installation is Windows-only; serialize portable fixture tests in-process.
var operations = struct {
	sync.Mutex
	held map[string]bool
}{held: make(map[string]bool)}

func acquireOperationKey(key string) (func(), error) {
	operations.Lock()
	defer operations.Unlock()
	if operations.held[key] {
		return nil, errOperationInProgress
	}
	operations.held[key] = true
	return func() {
		operations.Lock()
		delete(operations.held, key)
		operations.Unlock()
	}, nil
}
