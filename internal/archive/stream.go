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

	"github.com/0disoft/velox/internal/artifactlimits"
	"github.com/0disoft/velox/internal/buildphase"
)

type Stream struct {
	destination string
	output      *os.File
	writer      *zip.Writer
	names       map[string]struct{}
	fileCount   int
	finished    bool
	budget      artifactlimits.Budget
	headers     []*zip.FileHeader
	failure     error
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
	var compressor *flate.Writer
	writer.RegisterCompressor(zip.Deflate, func(destination io.Writer) (io.WriteCloser, error) {
		if compressor == nil {
			var err error
			compressor, err = flate.NewWriter(destination, flate.BestSpeed)
			return compressor, err
		}
		// zip.Writer closes the previous entry before requesting the next one.
		compressor.Reset(destination)
		return compressor, nil
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
	if err := stream.budget.Add(name, 0); err != nil {
		return nil, err
	}
	header := &zip.FileHeader{Name: name, Method: compressionMethod(name), Modified: normalizedTime}
	header.SetMode(mode)
	entry, err := stream.writer.CreateHeader(header)
	if err != nil {
		return nil, fmt.Errorf("create archive entry %s: %w", name, err)
	}
	stream.names[key] = struct{}{}
	stream.fileCount++
	stream.headers = append(stream.headers, header)
	return &boundedEntry{stream: stream, writer: entry, name: name}, nil
}

type boundedEntry struct {
	stream  *Stream
	writer  io.Writer
	name    string
	written uint64
}

func (entry *boundedEntry) Write(value []byte) (int, error) {
	if entry.stream.finished {
		return 0, errors.New("archive stream is closed")
	}
	if entry.stream.failure != nil {
		return 0, entry.stream.failure
	}
	if err := entry.stream.budget.Grow(entry.name, entry.written, uint64(len(value))); err != nil {
		entry.stream.failure = err
		return 0, err
	}
	written, err := entry.writer.Write(value)
	entry.written += uint64(written)
	if err == nil && written != len(value) {
		err = io.ErrShortWrite
	}
	entry.stream.failure = err
	return written, err
}

func (stream *Stream) Close() (Result, error) {
	return stream.CloseObserved(nil)
}

func (stream *Stream) CloseObserved(observer buildphase.Observer) (Result, error) {
	if stream == nil || stream.finished {
		return Result{}, errors.New("archive stream is closed")
	}
	if stream.failure != nil {
		stream.Abort()
		return Result{}, stream.failure
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
	for _, header := range stream.headers {
		if err := artifactlimits.CheckCompression(header.Name, header.UncompressedSize64, header.CompressedSize64); err != nil {
			stream.Abort()
			return Result{}, err
		}
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
