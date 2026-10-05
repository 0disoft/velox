// Package devdiagnostic formats bounded, metadata-only development errors.
package devdiagnostic

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/0disoft/velox/internal/assettree"
)

const MaxReports = 20
const maxRequestBytes = 1024
const maxCoordinate = 10000000

//go:embed listener.js
var listener string

func ListenerSource() string { return listener }

type Reporter struct {
	writer   io.Writer
	sources  map[string]bool
	attempts int
}

// New records only safe source names, not file contents. Call only in debug mode.
func New(root string, writer io.Writer) (*Reporter, error) {
	tree, err := assettree.ScanMetadata(root)
	if err != nil {
		return nil, err
	}
	if len(tree.Files) > 10000 {
		return nil, fmt.Errorf("debug asset count exceeds 10000")
	}
	r := &Reporter{writer: writer, sources: make(map[string]bool, len(tree.Files))}
	for _, asset := range tree.Files {
		if len(asset.RelativePath) <= 512 && !strings.ContainsFunc(asset.RelativePath, unicode.IsControl) {
			r.sources[asset.RelativePath] = true
		}
	}
	return r, nil
}

// Report runs on the existing UI thread. Invalid requests also consume budget.
func (r *Reporter) Report(raw json.RawMessage) {
	if r.attempts >= MaxReports || r.writer == nil {
		return
	}
	r.attempts++
	if len(raw) > maxRequestBytes {
		return
	}
	var event struct {
		Kind   string  `json:"kind"`
		Source string  `json:"source"`
		Line   *uint32 `json:"line"`
		Column *uint32 `json:"column"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&event) != nil || event.Line == nil || event.Column == nil ||
		*event.Line > maxCoordinate || *event.Column > maxCoordinate ||
		(event.Kind != "uncaught-error" && event.Kind != "unhandled-rejection") {
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return
	}
	source, line, column := "<unknown>", uint32(0), uint32(0)
	if event.Kind == "uncaught-error" && r.sources[event.Source] {
		source, line, column = event.Source, *event.Line, *event.Column
	}
	fmt.Fprintf(r.writer, "velox-debug: %s %s:%d:%d\n", event.Kind, source, line, column)
}
