package archive

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
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

func Create(sourceDirectory, destination, rootName string) (Result, error) {
	paths, err := collectFiles(sourceDirectory)
	if err != nil {
		return Result{}, err
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
	for _, relative := range paths {
		header := &zip.FileHeader{
			Name:     rootName + "/" + filepath.ToSlash(relative),
			Method:   zip.Deflate,
			Modified: normalizedTime,
		}
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			writer.Close()
			return Result{}, fmt.Errorf("create archive entry %s: %w", relative, err)
		}
		input, err := os.Open(filepath.Join(sourceDirectory, relative))
		if err != nil {
			writer.Close()
			return Result{}, fmt.Errorf("open archive input %s: %w", relative, err)
		}
		_, copyErr := io.Copy(entry, input)
		closeErr := input.Close()
		if copyErr != nil {
			writer.Close()
			return Result{}, fmt.Errorf("write archive entry %s: %w", relative, copyErr)
		}
		if closeErr != nil {
			writer.Close()
			return Result{}, fmt.Errorf("close archive input %s: %w", relative, closeErr)
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
	return Result{FileCount: len(paths), Size: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
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
