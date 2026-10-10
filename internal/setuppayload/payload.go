package setuppayload

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/velox/internal/artifactlimits"
	"github.com/0disoft/velox/internal/inspector"
	"github.com/0disoft/velox/internal/safefs"
)

const magic = "VeloxSetupV1\x00\x00\x00\x00"
const footerSize = 64
const maxTemplateSize = 128 << 20

type Result struct {
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type Payload struct {
	file          *os.File
	TemplateBytes int64
	ArchiveBytes  int64
}

// Extraction carries the completed tree's inspection for display only.
// Installation must revalidate the directory rather than trust this snapshot.
type Extraction struct {
	Directory  string
	Inspection inspector.Result
}

// Build attaches a verified portable ZIP to an unsigned, prebuilt Windows GUI
// setup template. It does not compile application code or modify the template.
func Build(template, archive, output string) (Result, error) {
	if _, err := inspector.Inspect(archive); err != nil {
		return Result{}, err
	}
	input, info, err := safefs.OpenVerifiedRegular(template)
	if err != nil {
		return Result{}, err
	}
	defer input.Close()
	image, err := pe.NewFile(input)
	if err != nil {
		return Result{}, err
	}
	defer image.Close()
	header, ok := image.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || image.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || header.Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI {
		return Result{}, errors.New("setup template must be a Windows x64 GUI executable")
	}
	if header.DataDirectory[4].Size != 0 || header.DataDirectory[4].VirtualAddress != 0 {
		return Result{}, errors.New("signed setup templates cannot have an appended payload")
	}
	if info.Size() <= 0 || info.Size() > maxTemplateSize {
		return Result{}, errors.New("setup template size outside limits")
	}
	return pack(input, info.Size(), archive, output)
}

func pack(template io.Reader, templateSize int64, archive, output string) (Result, error) {
	input, info, err := safefs.OpenVerifiedRegular(archive)
	if err != nil {
		return Result{}, err
	}
	defer input.Close()
	if info.Size() <= 0 || info.Size() > artifactlimits.MaxTotalBytes {
		return Result{}, errors.New("setup archive size outside limits")
	}
	if err := safefs.EnsureDirectory(filepath.Dir(output), 0o755); err != nil {
		return Result{}, err
	}
	if err := safefs.RejectLinkedComponents(output); err != nil {
		return Result{}, err
	}
	stage, err := os.CreateTemp(filepath.Dir(output), ".velox-setup-")
	if err != nil {
		return Result{}, err
	}
	defer os.Remove(stage.Name())
	defer stage.Close()
	allHash, payloadHash := sha256.New(), sha256.New()
	writer := io.MultiWriter(stage, allHash)
	if size, err := io.CopyN(writer, template, templateSize); err != nil || size != templateSize {
		return Result{}, errors.New("copy setup template failed")
	}
	if size, err := io.Copy(io.MultiWriter(writer, payloadHash), io.LimitReader(input, info.Size()+1)); err != nil || size != info.Size() {
		return Result{}, errors.New("copy setup payload failed or source changed")
	}
	footer := make([]byte, footerSize)
	copy(footer, magic)
	binary.LittleEndian.PutUint64(footer[16:24], uint64(templateSize))
	binary.LittleEndian.PutUint64(footer[24:32], uint64(info.Size()))
	copy(footer[32:], payloadHash.Sum(nil))
	if _, err := writer.Write(footer); err != nil {
		return Result{}, err
	}
	if err := stage.Close(); err != nil {
		return Result{}, err
	}
	if err := os.Rename(stage.Name(), output); err != nil {
		return Result{}, err
	}
	return Result{File: output, Bytes: templateSize + info.Size() + footerSize, SHA256: hex.EncodeToString(allHash.Sum(nil))}, nil
}

func Open(executable string) (*Payload, error) {
	file, info, err := safefs.OpenVerifiedRegular(executable)
	if err != nil {
		return nil, err
	}
	failed := true
	defer func() {
		if failed {
			file.Close()
		}
	}()
	if info.Size() < footerSize {
		return nil, errors.New("setup payload is missing")
	}
	footer := make([]byte, footerSize)
	if _, err := file.ReadAt(footer, info.Size()-footerSize); err != nil {
		return nil, err
	}
	base, size := binary.LittleEndian.Uint64(footer[16:24]), binary.LittleEndian.Uint64(footer[24:32])
	if string(footer[:16]) != magic || base == 0 || base > maxTemplateSize || size == 0 || size > artifactlimits.MaxTotalBytes || base+size+footerSize != uint64(info.Size()) {
		return nil, errors.New("setup payload footer is invalid")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.NewSectionReader(file, int64(base), int64(size))); err != nil {
		return nil, err
	}
	if !bytes.Equal(hash.Sum(nil), footer[32:]) {
		return nil, errors.New("setup payload checksum mismatch")
	}
	failed = false
	return &Payload{file: file, TemplateBytes: int64(base), ArchiveBytes: int64(size)}, nil
}

func (payload *Payload) Close() error { return payload.file.Close() }

func (payload *Payload) CopyTemplate(output string) error {
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, io.NewSectionReader(payload.file, 0, payload.TemplateBytes))
	return errors.Join(copyErr, file.Close())
}

// Extract accepts only canonical, regular ZIP entries under one application
// root, enforces expansion budgets, and rechecks the completed portable tree.
func (payload *Payload) Extract(output string) (Extraction, error) {
	reader, err := zip.NewReader(io.NewSectionReader(payload.file, payload.TemplateBytes, payload.ArchiveBytes), payload.ArchiveBytes)
	if err != nil {
		return Extraction{}, err
	}
	if len(reader.File) == 0 || len(reader.File) > artifactlimits.MaxFiles {
		return Extraction{}, errors.New("setup file count outside limits")
	}
	root := ""
	seen := make(map[string]bool)
	budget := artifactlimits.Budget{}
	for _, entry := range reader.File {
		parts := strings.SplitN(entry.Name, "/", 2)
		if !entry.Mode().IsRegular() || safefs.ValidateArchiveEntry(entry.Name) != nil || len(parts) != 2 || seen[strings.ToLower(entry.Name)] {
			return Extraction{}, errors.New("unsafe or duplicate setup ZIP entry")
		}
		seen[strings.ToLower(entry.Name)] = true
		if root == "" {
			root = parts[0]
		} else if root != parts[0] {
			return Extraction{}, errors.New("setup ZIP contains multiple roots")
		}
		if err := budget.Add(entry.Name, entry.UncompressedSize64); err != nil {
			return Extraction{}, err
		}
		if err := artifactlimits.CheckCompression(entry.Name, entry.UncompressedSize64, entry.CompressedSize64); err != nil {
			return Extraction{}, err
		}
	}
	if err := safefs.EnsureDirectory(output, 0o755); err != nil {
		return Extraction{}, err
	}
	for _, entry := range reader.File {
		path := filepath.Join(output, filepath.FromSlash(entry.Name))
		if err := safefs.EnsureDirectory(filepath.Dir(path), 0o755); err != nil {
			return Extraction{}, err
		}
		input, err := entry.Open()
		if err != nil {
			return Extraction{}, err
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
		if err != nil {
			input.Close()
			return Extraction{}, err
		}
		size, copyErr := io.Copy(file, io.LimitReader(input, int64(entry.UncompressedSize64)+1))
		closeErr := errors.Join(input.Close(), file.Close())
		if copyErr != nil || closeErr != nil || uint64(size) != entry.UncompressedSize64 {
			return Extraction{}, fmt.Errorf("extract setup ZIP entry: %w", errors.Join(copyErr, closeErr, errors.New("entry size or checksum invalid")))
		}
	}
	directory := filepath.Join(output, root)
	inspection, err := inspector.Inspect(directory)
	if err != nil {
		return Extraction{}, err
	}
	return Extraction{Directory: directory, Inspection: inspection}, nil
}
