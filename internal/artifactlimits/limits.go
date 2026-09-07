package artifactlimits

import "fmt"

const (
	MaxFiles         = 100_000
	MaxMetadataBytes = 1 << 20
	MaxEntryBytes    = 512 << 20
	MaxTotalBytes    = 1 << 30
	MaxExpandRatio   = 1_000
)

type Budget struct {
	Files int
	Bytes uint64
}

func (budget *Budget) Add(name string, size uint64) error {
	if budget.Files >= MaxFiles {
		return fmt.Errorf("artifact exceeds file count limit of %d", MaxFiles)
	}
	if err := budget.Grow(name, 0, size); err != nil {
		return err
	}
	budget.Files++
	return nil
}

func (budget *Budget) Grow(name string, entryBytes, additional uint64) error {
	if entryBytes > MaxEntryBytes || additional > MaxEntryBytes-entryBytes {
		return fmt.Errorf("artifact entry exceeds %d-byte size limit: %s", MaxEntryBytes, name)
	}
	if budget.Bytes > MaxTotalBytes || additional > MaxTotalBytes-budget.Bytes {
		return fmt.Errorf("artifact exceeds %d-byte total size limit", MaxTotalBytes)
	}
	budget.Bytes += additional
	return nil
}

func CheckCompression(name string, uncompressed, compressed uint64) error {
	if uncompressed > 0 && compressed == 0 {
		return fmt.Errorf("ZIP entry has an invalid compression size: %s", name)
	}
	if compressed > 0 && uncompressed/compressed > MaxExpandRatio {
		return fmt.Errorf("ZIP entry exceeds compression ratio limit: %s", name)
	}
	return nil
}
