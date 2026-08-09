package archive

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/0disoft/velox/internal/buildphase"
	"github.com/0disoft/velox/internal/safefs"
)

var normalizedTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

type Result struct {
	FileCount int
	Size      int64
	SHA256    string
}

type Input struct {
	Source string
	Name   string
}

func Create(sourceDirectory, destination, rootName string) (Result, error) {
	return CreateObserved(sourceDirectory, destination, rootName, nil)
}

func CreateObserved(sourceDirectory, destination, rootName string, observer buildphase.Observer) (Result, error) {
	totalStarted := time.Now()
	defer buildphase.Record(observer, "archive.total", totalStarted)
	collectStarted := time.Now()
	paths, err := collectFiles(sourceDirectory)
	buildphase.Record(observer, "archive.collect", collectStarted)
	if err != nil {
		return Result{}, err
	}
	inputs := make([]Input, 0, len(paths))
	for _, relative := range paths {
		inputs = append(inputs, Input{Source: filepath.Join(sourceDirectory, relative), Name: rootName + "/" + filepath.ToSlash(relative)})
	}
	return createFiles(destination, inputs, observer)
}

func CreateFiles(destination string, inputs []Input) (Result, error) {
	return createFiles(destination, append([]Input(nil), inputs...), nil)
}

func createFiles(destination string, inputs []Input, observer buildphase.Observer) (Result, error) {
	if len(inputs) == 0 {
		return Result{}, errors.New("archive requires at least one input")
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].Name < inputs[j].Name })
	for index, input := range inputs {
		if input.Source == "" || !safeEntryName(input.Name) {
			return Result{}, fmt.Errorf("unsafe archive input %q", input.Name)
		}
		if index > 0 && strings.EqualFold(inputs[index-1].Name, input.Name) {
			return Result{}, fmt.Errorf("duplicate archive entry %s", input.Name)
		}
	}
	stream, err := NewStream(destination)
	if err != nil {
		return Result{}, err
	}
	defer stream.Abort()
	entriesStarted := time.Now()
	for _, input := range inputs {
		entry, err := stream.CreateEntry(input.Name, 0o644)
		if err != nil {
			return Result{}, err
		}
		source, info, err := safefs.OpenVerifiedRegular(input.Source)
		if err != nil {
			return Result{}, fmt.Errorf("open archive input %s: %w", input.Name, err)
		}
		written, copyErr := io.Copy(entry, source)
		closeErr := source.Close()
		if copyErr != nil {
			return Result{}, fmt.Errorf("write archive entry %s: %w", input.Name, copyErr)
		}
		if closeErr != nil {
			return Result{}, fmt.Errorf("close archive input %s: %w", input.Name, closeErr)
		}
		if written != info.Size() {
			return Result{}, fmt.Errorf("archive input %s changed while reading", input.Name)
		}
	}
	buildphase.Record(observer, "archive.entries", entriesStarted)
	return stream.CloseObserved(observer)
}

func compressionMethod(name string) uint16 {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".7z", ".avif", ".br", ".gif", ".gz", ".jpeg", ".jpg", ".mp3", ".mp4", ".pdf", ".png", ".webm", ".webp", ".woff", ".woff2", ".zip":
		return zip.Store
	default:
		return zip.Deflate
	}
}

func safeEntryName(name string) bool {
	return safefs.ValidateArchiveEntry(name) == nil
}

func collectFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
			return fmt.Errorf("archive input escaped root: %s", path)
		}
		paths = append(paths, relative)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan archive input: %w", err)
	}
	sort.Slice(paths, func(i, j int) bool {
		return filepath.ToSlash(paths[i]) < filepath.ToSlash(paths[j])
	})
	return paths, nil
}
