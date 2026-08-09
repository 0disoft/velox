package archive

import (
	"archive/zip"
	"compress/flate"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/0disoft/velox/internal/buildphase"
)

type Stream struct {
	destination string
	output      *os.File
	writer      *zip.Writer
	names       map[string]struct{}
	fileCount   int
	finished    bool
}

func NewStream(destination string) (*Stream, error) {
	if _, err := os.Lstat(destination); err == nil {
		return nil, errors.New("archive output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect archive output: %w", err)
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create archive: %w", err)
	}
	writer := zip.NewWriter(output)
	writer.RegisterCompressor(zip.Deflate, func(destination io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(destination, flate.BestSpeed)
	})
	return &Stream{
		destination: destination,
		output:      output,
		writer:      writer,
		names:       make(map[string]struct{}),
	}, nil
}

func (stream *Stream) CreateEntry(name string, mode os.FileMode) (io.Writer, error) {
	if stream == nil || stream.finished {
		return nil, errors.New("archive stream is closed")
	}
	if !safeEntryName(name) {
		return nil, fmt.Errorf("unsafe archive input %q", name)
	}
	key := strings.ToLower(name)
	if _, exists := stream.names[key]; exists {
		return nil, fmt.Errorf("duplicate archive entry %s", name)
	}
	header := &zip.FileHeader{Name: name, Method: compressionMethod(name), Modified: normalizedTime}
	header.SetMode(mode)
	entry, err := stream.writer.CreateHeader(header)
	if err != nil {
		return nil, fmt.Errorf("create archive entry %s: %w", name, err)
	}
	stream.names[key] = struct{}{}
	stream.fileCount++
	return entry, nil
}

func (stream *Stream) Close() (Result, error) {
	return stream.CloseObserved(nil)
}

func (stream *Stream) CloseObserved(observer buildphase.Observer) (Result, error) {
	if stream == nil || stream.finished {
		return Result{}, errors.New("archive stream is closed")
	}
	if stream.fileCount == 0 {
		stream.Abort()
		return Result{}, errors.New("archive requires at least one input")
	}
	finalizeStarted := time.Now()
	if err := stream.writer.Close(); err != nil {
		stream.Abort()
		return Result{}, fmt.Errorf("finalize archive: %w", err)
	}
	buildphase.Record(observer, "archive.finalize", finalizeStarted)
	syncStarted := time.Now()
	if err := stream.output.Sync(); err != nil {
		stream.Abort()
		return Result{}, fmt.Errorf("sync archive: %w", err)
	}
	buildphase.Record(observer, "archive.sync", syncStarted)
	verifyStarted := time.Now()
	if _, err := stream.output.Seek(0, io.SeekStart); err != nil {
		stream.Abort()
		return Result{}, fmt.Errorf("rewind archive for verification: %w", err)
	}
	hash := sha256.New()
	verifiedSize, err := io.Copy(hash, stream.output)
	if err != nil {
		stream.Abort()
		return Result{}, fmt.Errorf("verify archive bytes: %w", err)
	}
	info, err := stream.output.Stat()
	if err != nil {
		stream.Abort()
		return Result{}, fmt.Errorf("inspect archive: %w", err)
	}
	if verifiedSize != info.Size() {
		stream.Abort()
		return Result{}, fmt.Errorf("verify archive size: read %d bytes, expected %d", verifiedSize, info.Size())
	}
	buildphase.Record(observer, "archive.verify", verifyStarted)
	if err := stream.output.Close(); err != nil {
		stream.Abort()
		return Result{}, fmt.Errorf("close archive: %w", err)
	}
	stream.finished = true
	return Result{FileCount: stream.fileCount, Size: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func (stream *Stream) Abort() error {
	if stream == nil || stream.finished {
		return nil
	}
	stream.finished = true
	closeErr := stream.output.Close()
	removeErr := os.Remove(stream.destination)
	if closeErr != nil {
		return closeErr
	}
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return removeErr
	}
	return nil
}
