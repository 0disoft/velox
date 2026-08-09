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

	"github.com/0disoft/velox/internal/safefs"
)

type File struct {
	RelativePath     string
	SourcePath       string
	Size             int64
	ModifiedUnixNano int64
	SHA256           string
}

type Tree struct {
	Files      []File
	TotalBytes int64
	Digest     string
}

// ValidateResolvedEntry verifies the runtime asset boundary without scanning or
// hashing unrelated assets. Both paths must already be absolute and lexical
// containment must be established by the caller.
func ValidateResolvedEntry(root, entry string) error {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("inspect asset root: %w", err)
	}
	if linked, err := safefs.IsLinkOrReparse(root, rootInfo); err != nil {
		return err
	} else if linked {
		return errors.New("asset root must not be a link or reparse point")
	}
	if !rootInfo.IsDir() {
		return errors.New("asset root is not a directory")
	}

	relative, err := filepath.Rel(root, entry)
	if err != nil {
		return fmt.Errorf("resolve entry relative path: %w", err)
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("entry point must stay inside the asset root")
	}

	current := root
	for _, segment := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect entry point: %w", err)
		}
		linked, err := safefs.IsLinkOrReparse(current, info)
		if err != nil {
			return err
		}
		if linked {
			return fmt.Errorf("entry path is a link or reparse point: %s", filepath.ToSlash(relative))
		}
		if current != entry && !info.IsDir() {
			return fmt.Errorf("entry parent is not a directory: %s", filepath.ToSlash(relative))
		}
		if current == entry && !info.Mode().IsRegular() {
			return errors.New("entry point is not a regular file")
		}
	}
	return nil
}

func Scan(root string) (Tree, error) {
	return scan(root, true)
}

// ScanMetadata validates the complete tree without reading file contents.
func ScanMetadata(root string) (Tree, error) {
	return scan(root, false)
}

// RevalidateSnapshot checks shape and sizes without rehashing every source.
// The builder verifies each planned digest while copying immediately afterward.
func RevalidateSnapshot(root string, expected Tree) error {
	current, err := scan(root, false)
	if err != nil {
		return err
	}
	if len(current.Files) != len(expected.Files) || current.TotalBytes != expected.TotalBytes {
		return errors.New("asset tree shape or size changed")
	}
	for index := range current.Files {
		if current.Files[index].RelativePath != expected.Files[index].RelativePath ||
			current.Files[index].Size != expected.Files[index].Size ||
			current.Files[index].ModifiedUnixNano != expected.Files[index].ModifiedUnixNano {
			return errors.New("asset tree shape, size, or modification time changed")
		}
	}
	return nil
}

func scan(root string, hashContents bool) (Tree, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return Tree{}, fmt.Errorf("inspect asset root: %w", err)
	}
	if !info.IsDir() {
		return Tree{}, errors.New("asset root is not a directory")
	}
	if linked, err := safefs.IsLinkOrReparse(root, info); err != nil {
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
		linked, err := safefs.IsLinkOrReparse(path, info)
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
		if err := safefs.ValidateRelativePath(relative); err != nil {
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
		digest := ""
		if hashContents {
			digest, err = hashFile(path)
			if err != nil {
				return fmt.Errorf("hash asset %s: %w", filepath.ToSlash(relative), err)
			}
		}
		files = append(files, File{
			RelativePath:     filepath.ToSlash(relative),
			SourcePath:       path,
			Size:             info.Size(),
			ModifiedUnixNano: info.ModTime().UnixNano(),
			SHA256:           digest,
		})
		return nil
	})
	if err != nil {
		return Tree{}, err
	}
	if hashContents {
		return Summarize(files), nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].RelativePath < files[j].RelativePath })
	var total int64
	for _, file := range files {
		total += file.Size
	}
	return Tree{Files: files, TotalBytes: total}, nil
}

func Summarize(files []File) Tree {
	files = append([]File(nil), files...)
	sort.Slice(files, func(i, j int) bool { return files[i].RelativePath < files[j].RelativePath })
	hash := sha256.New()
	var total int64
	for _, file := range files {
		fmt.Fprintf(hash, "%s\x00%d\x00%s\n", file.RelativePath, file.Size, file.SHA256)
		total += file.Size
	}
	return Tree{Files: files, TotalBytes: total, Digest: hex.EncodeToString(hash.Sum(nil))}
}

func hashFile(path string) (string, error) {
	file, _, err := safefs.OpenVerifiedRegular(path)
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
