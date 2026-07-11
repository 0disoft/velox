package assettree

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type File struct {
	RelativePath string
	SourcePath   string
	Size         int64
	SHA256       string
}

type Tree struct {
	Files      []File
	TotalBytes int64
	Digest     string
}

func Scan(root string) (Tree, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return Tree{}, fmt.Errorf("inspect asset root: %w", err)
	}
	if !info.IsDir() {
		return Tree{}, errors.New("asset root is not a directory")
	}
	if linked, err := isLinkOrReparse(root, info); err != nil {
		return Tree{}, err
	} else if linked {
		return Tree{}, errors.New("asset root must not be a link or reparse point")
	}

	var files []File
	casePaths := make(map[string]string)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		linked, err := isLinkOrReparse(path, info)
		if err != nil {
			return err
		}
		if linked {
			return fmt.Errorf("asset path is a link or reparse point: %s", relativeDisplay(root, path))
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if err := validateRelativePath(relative); err != nil {
			return fmt.Errorf("invalid asset path %q: %w", filepath.ToSlash(relative), err)
		}
		key := strings.ToLower(filepath.ToSlash(relative))
		if previous, exists := casePaths[key]; exists {
			return fmt.Errorf("case-colliding asset paths %q and %q", previous, filepath.ToSlash(relative))
		}
		casePaths[key] = filepath.ToSlash(relative)
		if entry.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("asset is not a regular file: %s", filepath.ToSlash(relative))
		}
		digest, err := hashFile(path)
		if err != nil {
			return fmt.Errorf("hash asset %s: %w", filepath.ToSlash(relative), err)
		}
		files = append(files, File{
			RelativePath: filepath.ToSlash(relative),
			SourcePath:   path,
			Size:         info.Size(),
			SHA256:       digest,
		})
		return nil
	})
	if err != nil {
		return Tree{}, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].RelativePath < files[j].RelativePath })
	hash := sha256.New()
	var total int64
	for _, file := range files {
		fmt.Fprintf(hash, "%s\x00%d\x00%s\n", file.RelativePath, file.Size, file.SHA256)
		total += file.Size
	}
	return Tree{Files: files, TotalBytes: total, Digest: hex.EncodeToString(hash.Sum(nil))}, nil
}

func validateRelativePath(path string) error {
	for _, part := range strings.FieldsFunc(filepath.ToSlash(path), func(r rune) bool { return r == '/' }) {
		if part == "" || part == "." || part == ".." {
			return errors.New("empty or traversal segment")
		}
		if strings.Contains(part, ":") {
			return errors.New("alternate data streams are not allowed")
		}
		trimmed := strings.TrimRight(part, ". ")
		if trimmed != part {
			return errors.New("trailing dots or spaces are not allowed")
		}
		base := strings.ToUpper(strings.TrimSuffix(trimmed, filepath.Ext(trimmed)))
		if isReservedName(base) {
			return fmt.Errorf("reserved Windows name %q", part)
		}
	}
	return nil
}

func isReservedName(name string) bool {
	if name == "CON" || name == "PRN" || name == "AUX" || name == "NUL" {
		return true
	}
	for i := 1; i <= 9; i++ {
		if name == fmt.Sprintf("COM%d", i) || name == fmt.Sprintf("LPT%d", i) {
			return true
		}
	}
	return false
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func relativeDisplay(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.ToSlash(relative)
}
