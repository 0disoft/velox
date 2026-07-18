package archive

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	paths, err := collectFiles(sourceDirectory)
	if err != nil {
		return Result{}, err
	}
	inputs := make([]Input, 0, len(paths))
	for _, relative := range paths {
		inputs = append(inputs, Input{Source: filepath.Join(sourceDirectory, relative), Name: rootName + "/" + filepath.ToSlash(relative)})
	}
	return createFiles(destination, inputs)
}

func CreateFiles(destination string, inputs []Input) (Result, error) {
	return createFiles(destination, append([]Input(nil), inputs...))
}

func createFiles(destination string, inputs []Input) (Result, error) {
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
	if _, err := os.Lstat(destination); err == nil {
		return Result{}, errors.New("archive output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return Result{}, fmt.Errorf("inspect archive output: %w", err)
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return Result{}, fmt.Errorf("create archive: %w", err)
	}
	success := false
	defer func() {
		output.Close()
		if !success {
			os.Remove(destination)
		}
	}()

	hash := sha256.New()
	writer := zip.NewWriter(io.MultiWriter(output, hash))
	for _, input := range inputs {
		header := &zip.FileHeader{
			Name:     input.Name,
			Method:   zip.Deflate,
			Modified: normalizedTime,
		}
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			writer.Close()
			return Result{}, fmt.Errorf("create archive entry %s: %w", input.Name, err)
		}
		linkInfo, err := os.Lstat(input.Source)
		if err != nil {
			writer.Close()
			return Result{}, fmt.Errorf("inspect archive input %s: %w", input.Name, err)
		}
		if !linkInfo.Mode().IsRegular() || linkInfo.Mode()&os.ModeSymlink != 0 {
			writer.Close()
			return Result{}, fmt.Errorf("archive input %s must be a regular file", input.Name)
		}
		source, err := os.Open(input.Source)
		if err != nil {
			writer.Close()
			return Result{}, fmt.Errorf("open archive input %s: %w", input.Name, err)
		}
		info, statErr := source.Stat()
		if statErr != nil || !info.Mode().IsRegular() || !os.SameFile(linkInfo, info) {
			source.Close()
			writer.Close()
			if statErr != nil {
				return Result{}, fmt.Errorf("inspect opened archive input %s: %w", input.Name, statErr)
			}
			return Result{}, fmt.Errorf("archive input %s changed while opening", input.Name)
		}
		written, copyErr := io.Copy(entry, source)
		closeErr := source.Close()
		if copyErr != nil {
			writer.Close()
			return Result{}, fmt.Errorf("write archive entry %s: %w", input.Name, copyErr)
		}
		if closeErr != nil {
			writer.Close()
			return Result{}, fmt.Errorf("close archive input %s: %w", input.Name, closeErr)
		}
		if written != info.Size() {
			writer.Close()
			return Result{}, fmt.Errorf("archive input %s changed while reading", input.Name)
		}
	}
	if err := writer.Close(); err != nil {
		return Result{}, fmt.Errorf("finalize archive: %w", err)
	}
	if err := output.Sync(); err != nil {
		return Result{}, fmt.Errorf("sync archive: %w", err)
	}
	if err := output.Close(); err != nil {
		return Result{}, fmt.Errorf("close archive: %w", err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		return Result{}, fmt.Errorf("inspect archive: %w", err)
	}
	success = true
	return Result{FileCount: len(inputs), Size: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func safeEntryName(name string) bool {
	return name != "" &&
		!strings.Contains(name, "\\") &&
		!strings.Contains(name, ":") &&
		!path.IsAbs(name) &&
		!filepath.IsAbs(name) &&
		filepath.VolumeName(name) == "" &&
		path.Clean(name) == name &&
		name != "." &&
		!strings.HasPrefix(name, "../")
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
