package signingrecord

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/0disoft/velox/internal/releasebundle"
)

const (
	SchemaVersion = "velox.signing-record/v1"
	Target        = "windows-x64"

	ModeDryRun  = "dry-run"
	ModeRelease = "release"

	ProviderSignPath  = "signpath-foundation"
	ProviderMicrosoft = "microsoft-artifact-signing"

	StatusNotPerformed = "not-performed"
	StatusVerified     = "verified"

	maxRecordBytes = 1 << 20
)

var (
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
	tagPattern    = regexp.MustCompile(`^v[0-9A-Za-z][0-9A-Za-z.+-]*$`)
	runIDPattern  = regexp.MustCompile(`^[1-9][0-9]*$`)
	digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Artifact struct {
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type NativeSet struct {
	Artifacts []Artifact `json:"artifacts"`
}

type Source struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Tag        string `json:"tag"`
	Workflow   string `json:"workflow"`
	RunID      string `json:"runId"`
}

type Provider struct {
	Name                  string `json:"name"`
	Project               string `json:"project"`
	ArtifactConfiguration string `json:"artifactConfiguration"`
	SigningPolicy         string `json:"signingPolicy"`
	RequestID             string `json:"requestId"`
}

type Certificate struct {
	Status             string `json:"status"`
	Subject            string `json:"subject,omitempty"`
	Issuer             string `json:"issuer,omitempty"`
	Serial             string `json:"serial,omitempty"`
	TimestampAuthority string `json:"timestampAuthority,omitempty"`
}

type Distribution struct {
	Archive   Artifact `json:"archive"`
	Manifest  Artifact `json:"manifest"`
	Checksums Artifact `json:"checksums"`
	SBOM      Artifact `json:"sbom"`
}

type Attestation struct {
	Kind    string   `json:"kind"`
	Subject Artifact `json:"subject"`
	Status  string   `json:"status"`
}

type Record struct {
	SchemaVersion  string        `json:"schemaVersion"`
	Mode           string        `json:"mode"`
	Publishable    bool          `json:"publishable"`
	ReleaseVersion string        `json:"releaseVersion"`
	Target         string        `json:"target"`
	Source         Source        `json:"source"`
	Unsigned       NativeSet     `json:"unsigned"`
	SigningInput   Artifact      `json:"signingInput"`
	Provider       Provider      `json:"provider"`
	Signed         NativeSet     `json:"signed"`
	Certificate    Certificate   `json:"certificate"`
	Distribution   Distribution  `json:"distribution"`
	Attestations   []Attestation `json:"attestations"`
}

type Files struct {
	UnsignedCLI     string
	UnsignedHost    string
	SigningInput    string
	SignedCLI       string
	SignedHost      string
	ReleaseArchive  string
	ReleaseManifest string
	Checksums       string
	SBOM            string
}

type DryRunOptions struct {
	ReleaseVersion string
	Source         Source
	Provider       Provider
	Files          Files
}

type WriteResult struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func BuildDryRun(options DryRunOptions) (Record, error) {
	unsigned, signed, distribution, signingInput, err := inspectFiles(options.Files)
	if err != nil {
		return Record{}, err
	}
	record := Record{
		SchemaVersion:  SchemaVersion,
		Mode:           ModeDryRun,
		Publishable:    false,
		ReleaseVersion: options.ReleaseVersion,
		Target:         Target,
		Source:         options.Source,
		Unsigned:       unsigned,
		SigningInput:   signingInput,
		Provider:       options.Provider,
		Signed:         signed,
		Certificate:    Certificate{Status: StatusNotPerformed},
		Distribution:   distribution,
		Attestations: []Attestation{
			{Kind: "build-provenance", Subject: distribution.Archive, Status: StatusNotPerformed},
			{Kind: "sbom", Subject: distribution.SBOM, Status: StatusNotPerformed},
		},
	}
	if err := Validate(record); err != nil {
		return Record{}, err
	}
	if err := verifyLineage(options.ReleaseVersion, options.Files, unsigned, signed, distribution); err != nil {
		return Record{}, err
	}
	return record, nil
}

func DecodeFile(path string) (Record, error) {
	data, err := readRegularFile(path, maxRecordBytes)
	if err != nil {
		return Record{}, fmt.Errorf("read signing record: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record Record
	if err := decoder.Decode(&record); err != nil {
		return Record{}, fmt.Errorf("decode signing record: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return Record{}, err
	}
	if err := Validate(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func VerifyFiles(record Record, files Files) error {
	if err := Validate(record); err != nil {
		return err
	}
	unsigned, signed, distribution, signingInput, err := inspectFiles(files)
	if err != nil {
		return err
	}
	checks := []struct {
		label string
		got   Artifact
		want  Artifact
	}{
		{"unsigned CLI", unsigned.Artifacts[0], record.Unsigned.Artifacts[0]},
		{"unsigned host", unsigned.Artifacts[1], record.Unsigned.Artifacts[1]},
		{"signing input", signingInput, record.SigningInput},
		{"signed CLI", signed.Artifacts[0], record.Signed.Artifacts[0]},
		{"signed host", signed.Artifacts[1], record.Signed.Artifacts[1]},
		{"release archive", distribution.Archive, record.Distribution.Archive},
		{"release manifest", distribution.Manifest, record.Distribution.Manifest},
		{"checksums", distribution.Checksums, record.Distribution.Checksums},
		{"SBOM", distribution.SBOM, record.Distribution.SBOM},
	}
	for _, check := range checks {
		if check.got != check.want {
			return fmt.Errorf("%s differs from signing record", check.label)
		}
	}
	return verifyLineage(record.ReleaseVersion, files, unsigned, signed, distribution)
}

func Validate(record Record) error {
	if record.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported signing record schema %q", record.SchemaVersion)
	}
	if record.Target != Target {
		return fmt.Errorf("unsupported signing target %q", record.Target)
	}
	if err := validateText("release version", record.ReleaseVersion); err != nil {
		return err
	}
	if err := validateSource(record.Source); err != nil {
		return err
	}
	if err := validateProvider(record.Provider); err != nil {
		return err
	}
	if err := validateNativeSet("unsigned", record.Unsigned); err != nil {
		return err
	}
	if err := validateArtifact("signing input", record.SigningInput, SigningInputName); err != nil {
		return err
	}
	if err := validateNativeSet("signed", record.Signed); err != nil {
		return err
	}
	for index := range record.Unsigned.Artifacts {
		if record.Unsigned.Artifacts[index].SHA256 == record.Signed.Artifacts[index].SHA256 {
			return fmt.Errorf("signed artifact %s has the unsigned digest", record.Signed.Artifacts[index].File)
		}
	}
	if err := validateDistribution(record.Distribution); err != nil {
		return err
	}
	if len(record.Attestations) != 2 {
		return errors.New("signing record must contain build-provenance and SBOM attestations")
	}
	if err := validateAttestation(record.Attestations[0], "build-provenance", record.Distribution.Archive); err != nil {
		return err
	}
	if err := validateAttestation(record.Attestations[1], "sbom", record.Distribution.SBOM); err != nil {
		return err
	}
	switch record.Mode {
	case ModeDryRun:
		if record.Publishable {
			return errors.New("dry-run signing record cannot be publishable")
		}
		if record.Certificate.Status != StatusNotPerformed || record.Certificate.Subject != "" || record.Certificate.Issuer != "" || record.Certificate.Serial != "" || record.Certificate.TimestampAuthority != "" {
			return errors.New("dry-run signing record cannot contain verified certificate evidence")
		}
		for _, attestation := range record.Attestations {
			if attestation.Status != StatusNotPerformed {
				return errors.New("dry-run attestation status must be not-performed")
			}
		}
	case ModeRelease:
		if !record.Publishable {
			return errors.New("release signing record must be publishable")
		}
		if err := validateCertificate(record.Certificate); err != nil {
			return err
		}
		for _, attestation := range record.Attestations {
			if attestation.Status != StatusVerified {
				return errors.New("release attestation status must be verified")
			}
		}
	default:
		return fmt.Errorf("unsupported signing record mode %q", record.Mode)
	}
	return nil
}

func Write(path string, record Record) (WriteResult, error) {
	if err := Validate(record); err != nil {
		return WriteResult{}, err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return WriteResult{}, fmt.Errorf("encode signing record: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return WriteResult{}, fmt.Errorf("create signing record directory: %w", err)
	}
	if _, err := os.Lstat(path); err == nil {
		return WriteResult{}, errors.New("signing record output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return WriteResult{}, fmt.Errorf("inspect signing record output: %w", err)
	}
	temporary := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if _, err := os.Lstat(temporary); err == nil {
		return WriteResult{}, errors.New("signing record temporary output already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return WriteResult{}, fmt.Errorf("inspect signing record temporary output: %w", err)
	}
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return WriteResult{}, fmt.Errorf("write signing record: %w", err)
	}
	success := false
	defer func() {
		if !success {
			_ = os.Remove(temporary)
		}
	}()
	if err := os.Rename(temporary, path); err != nil {
		return WriteResult{}, fmt.Errorf("promote signing record: %w", err)
	}
	success = true
	digest := sha256.Sum256(data)
	return WriteResult{Path: path, SHA256: hex.EncodeToString(digest[:])}, nil
}

func inspectFiles(files Files) (NativeSet, NativeSet, Distribution, Artifact, error) {
	unsignedCLI, err := inspectArtifact(files.UnsignedCLI, "velox.exe")
	if err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect unsigned CLI: %w", err)
	}
	unsignedHost, err := inspectArtifact(files.UnsignedHost, "velox-host.exe")
	if err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect unsigned host: %w", err)
	}
	signingInput, err := inspectArtifact(files.SigningInput, SigningInputName)
	if err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect signing input: %w", err)
	}
	if err := verifyExactSignedDirectory(files.SignedCLI, files.SignedHost); err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect signed output directory: %w", err)
	}
	signedCLI, err := inspectArtifact(files.SignedCLI, "velox.exe")
	if err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect signed CLI: %w", err)
	}
	signedHost, err := inspectArtifact(files.SignedHost, "velox-host.exe")
	if err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect signed host: %w", err)
	}
	archive, err := inspectArtifact(files.ReleaseArchive, "velox-windows-x64.zip")
	if err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect release archive: %w", err)
	}
	manifest, err := inspectArtifact(files.ReleaseManifest, "release-manifest.json")
	if err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect release manifest: %w", err)
	}
	checksums, err := inspectArtifact(files.Checksums, "checksums.sha256")
	if err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect checksums: %w", err)
	}
	sbom, err := inspectArtifact(files.SBOM, "velox-windows-x64.spdx.json")
	if err != nil {
		return NativeSet{}, NativeSet{}, Distribution{}, Artifact{}, fmt.Errorf("inspect SBOM: %w", err)
	}
	return NativeSet{Artifacts: []Artifact{unsignedCLI, unsignedHost}}, NativeSet{Artifacts: []Artifact{signedCLI, signedHost}}, Distribution{Archive: archive, Manifest: manifest, Checksums: checksums, SBOM: sbom}, signingInput, nil
}

func verifyExactSignedDirectory(cliPath, hostPath string) error {
	cliDirectory := filepath.Clean(filepath.Dir(cliPath))
	hostDirectory := filepath.Clean(filepath.Dir(hostPath))
	if cliDirectory != hostDirectory {
		return errors.New("signed executables must share one directory")
	}
	if filepath.Base(cliPath) != "velox.exe" || filepath.Base(hostPath) != "velox-host.exe" {
		return errors.New("signed executables must use the expected file names")
	}
	entries, err := os.ReadDir(cliDirectory)
	if err != nil {
		return err
	}
	if len(entries) != 2 {
		return fmt.Errorf("signed output directory must contain exactly two entries, found %d", len(entries))
	}
	expected := map[string]bool{"velox.exe": false, "velox-host.exe": false}
	for _, entry := range entries {
		if _, ok := expected[entry.Name()]; !ok {
			return fmt.Errorf("signed output directory contains unexpected entry %s", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect signed output %s: %w", entry.Name(), err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("signed output %s must be a regular file", entry.Name())
		}
		expected[entry.Name()] = true
	}
	for name, found := range expected {
		if !found {
			return fmt.Errorf("signed output directory is missing %s", name)
		}
	}
	return nil
}

func inspectArtifact(path, name string) (Artifact, error) {
	if path == "" {
		return Artifact{}, errors.New("path is required")
	}
	linkInfo, err := os.Lstat(path)
	if err != nil {
		return Artifact{}, err
	}
	if !linkInfo.Mode().IsRegular() || linkInfo.Mode()&os.ModeSymlink != 0 {
		return Artifact{}, errors.New("evidence input must be a non-empty regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Artifact{}, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(linkInfo, info) || info.Size() < 1 {
		return Artifact{}, errors.New("evidence input must be a non-empty regular file")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return Artifact{}, err
	}
	return Artifact{File: name, Bytes: info.Size(), SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func verifyLineage(releaseVersion string, files Files, unsigned, signed NativeSet, distribution Distribution) error {
	if err := verifySigningInput(files.SigningInput, unsigned); err != nil {
		return fmt.Errorf("verify signing input: %w", err)
	}
	manifestArtifacts, err := verifyReleaseManifest(files.ReleaseManifest, releaseVersion, signed)
	if err != nil {
		return fmt.Errorf("verify final release manifest: %w", err)
	}
	if err := verifyReleaseArchive(files.ReleaseArchive, manifestArtifacts, distribution.Manifest); err != nil {
		return fmt.Errorf("verify final release archive: %w", err)
	}
	if err := verifyChecksums(files.Checksums, distribution.Archive, distribution.SBOM); err != nil {
		return fmt.Errorf("verify final checksums: %w", err)
	}
	if err := verifySBOM(files.SBOM, distribution.Archive); err != nil {
		return fmt.Errorf("verify final SBOM: %w", err)
	}
	return nil
}

func verifySigningInput(path string, unsigned NativeSet) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) != 2 {
		return errors.New("signing input must contain exactly two entries")
	}
	unsignedByName := map[string]Artifact{
		unsigned.Artifacts[0].File: unsigned.Artifacts[0],
		unsigned.Artifacts[1].File: unsigned.Artifacts[1],
	}
	for index, name := range []string{"velox-host.exe", "velox.exe"} {
		entry := reader.File[index]
		if entry.Name != name {
			return fmt.Errorf("signing input entry %d must be %s", index, name)
		}
		artifact, err := inspectZIPEntry(entry, name)
		if err != nil {
			return err
		}
		if artifact != unsignedByName[name] {
			return fmt.Errorf("signing input %s differs from unsigned artifact", name)
		}
	}
	return nil
}

func verifyReleaseManifest(recordPath, releaseVersion string, signed NativeSet) (map[string]Artifact, error) {
	data, err := readRegularFile(recordPath, maxRecordBytes)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest releasebundle.Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, err
	}
	if manifest.SchemaVersion != releasebundle.SchemaVersion || manifest.ReleaseVersion != releaseVersion || manifest.Target != Target {
		return nil, errors.New("release manifest identity differs from signing record")
	}
	wantedSigned := map[string]Artifact{
		"velox.exe":      signed.Artifacts[0],
		"velox-host.exe": signed.Artifacts[1],
	}
	artifacts := make(map[string]Artifact, len(manifest.Artifacts))
	for _, source := range manifest.Artifacts {
		artifact := Artifact{File: source.File, Bytes: source.Bytes, SHA256: source.SHA256}
		if err := validateManifestArtifact(artifact); err != nil {
			return nil, err
		}
		if _, exists := artifacts[artifact.File]; exists {
			return nil, fmt.Errorf("release manifest repeats %s", artifact.File)
		}
		artifacts[artifact.File] = artifact
		if want, exists := wantedSigned[artifact.File]; exists && artifact != want {
			return nil, fmt.Errorf("release manifest %s differs from signed artifact", artifact.File)
		}
	}
	for _, name := range []string{"velox.exe", "velox-host.exe"} {
		if _, exists := artifacts[name]; !exists {
			return nil, fmt.Errorf("release manifest is missing %s", name)
		}
	}
	return artifacts, nil
}

func verifyReleaseArchive(archivePath string, artifacts map[string]Artifact, manifest Artifact) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	prefix := "velox-windows-x64/"
	if len(reader.File) != len(artifacts)+1 {
		return fmt.Errorf("release archive contains %d files, want %d", len(reader.File), len(artifacts)+1)
	}
	seenEntries := make(map[string]bool, len(reader.File))
	seenArtifacts := make(map[string]bool, len(artifacts))
	manifestSeen := false
	for _, entry := range reader.File {
		if seenEntries[entry.Name] {
			return fmt.Errorf("release archive repeats %s", entry.Name)
		}
		seenEntries[entry.Name] = true
		if !strings.HasPrefix(entry.Name, prefix) {
			return fmt.Errorf("release archive entry %s escapes the expected root", entry.Name)
		}
		relative := strings.TrimPrefix(entry.Name, prefix)
		if relative == "release-manifest.json" {
			if manifestSeen {
				return errors.New("release archive repeats release-manifest.json")
			}
			manifestSeen = true
			artifact, err := inspectZIPEntry(entry, manifest.File)
			if err != nil {
				return err
			}
			if artifact != manifest {
				return errors.New("release archive manifest differs from recorded artifact")
			}
			continue
		}
		want, exists := artifacts[relative]
		if !exists {
			return fmt.Errorf("release archive entry %s is not declared by the manifest", entry.Name)
		}
		seenArtifacts[relative] = true
		artifact, err := inspectZIPEntry(entry, relative)
		if err != nil {
			return err
		}
		if artifact != want {
			return fmt.Errorf("release archive %s differs from recorded artifact", entry.Name)
		}
	}
	if !manifestSeen {
		return errors.New("release archive is missing release-manifest.json")
	}
	for name := range artifacts {
		if !seenArtifacts[name] {
			return fmt.Errorf("release archive is missing %s", name)
		}
	}
	return nil
}

func validateManifestArtifact(artifact Artifact) error {
	if artifact.File == "" || strings.Contains(artifact.File, "\\") || path.IsAbs(artifact.File) || path.Clean(artifact.File) != artifact.File || strings.HasPrefix(artifact.File, "../") {
		return fmt.Errorf("release manifest artifact path %q is unsafe", artifact.File)
	}
	if artifact.Bytes < 1 || !digestPattern.MatchString(artifact.SHA256) {
		return fmt.Errorf("release manifest artifact %s has invalid evidence", artifact.File)
	}
	return nil
}

func inspectZIPEntry(entry *zip.File, name string) (Artifact, error) {
	if entry.FileInfo().IsDir() || !entry.Mode().IsRegular() || entry.UncompressedSize64 < 1 || entry.UncompressedSize64 > uint64(^uint64(0)>>1) {
		return Artifact{}, fmt.Errorf("ZIP entry %s must be a non-empty regular file", entry.Name)
	}
	file, err := entry.Open()
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, int64(entry.UncompressedSize64)+1))
	if err != nil {
		return Artifact{}, err
	}
	if written != int64(entry.UncompressedSize64) {
		return Artifact{}, fmt.Errorf("ZIP entry %s size differs from header", entry.Name)
	}
	return Artifact{File: name, Bytes: written, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func verifyChecksums(path string, archive, sbom Artifact) error {
	data, err := readRegularFile(path, maxRecordBytes)
	if err != nil {
		return err
	}
	wanted := map[string]string{archive.File: archive.SHA256, sbom.File: sbom.SHA256}
	seen := make(map[string]bool, len(wanted))
	for lineNumber, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		parts := strings.Split(line, "  ")
		if len(parts) != 2 || !digestPattern.MatchString(parts[0]) || parts[1] == "" {
			return fmt.Errorf("invalid checksum line %d", lineNumber+1)
		}
		want, exists := wanted[parts[1]]
		if !exists {
			continue
		}
		if seen[parts[1]] || parts[0] != want {
			return fmt.Errorf("checksum for %s differs from recorded artifact", parts[1])
		}
		seen[parts[1]] = true
	}
	for _, name := range []string{archive.File, sbom.File} {
		if !seen[name] {
			return fmt.Errorf("checksums are missing %s", name)
		}
	}
	return nil
}

func verifySBOM(path string, archive Artifact) error {
	data, err := readRegularFile(path, maxRecordBytes)
	if err != nil {
		return err
	}
	var document struct {
		SPDXVersion       string `json:"spdxVersion"`
		DocumentNamespace string `json:"documentNamespace"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return err
	}
	if document.SPDXVersion != "SPDX-2.3" || !strings.HasSuffix(document.DocumentNamespace, "/"+archive.SHA256) {
		return errors.New("SBOM does not identify the final release archive")
	}
	return nil
}

func validateSource(source Source) error {
	parsed, err := url.ParseRequestURI(source.Repository)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("source repository must be an HTTPS URL without credentials, query, or fragment")
	}
	if !commitPattern.MatchString(source.Commit) {
		return errors.New("source commit must be a lowercase 40-character digest")
	}
	if !tagPattern.MatchString(source.Tag) || len(source.Tag) > 128 {
		return errors.New("source tag is invalid")
	}
	if err := validateText("source workflow", source.Workflow); err != nil {
		return err
	}
	if !runIDPattern.MatchString(source.RunID) || len(source.RunID) > 32 {
		return errors.New("source run ID must be a positive decimal identifier")
	}
	return nil
}

func validateProvider(provider Provider) error {
	if provider.Name != ProviderSignPath && provider.Name != ProviderMicrosoft {
		return fmt.Errorf("unsupported signing provider %q", provider.Name)
	}
	for _, field := range [][2]string{
		{"provider project", provider.Project},
		{"artifact configuration", provider.ArtifactConfiguration},
		{"signing policy", provider.SigningPolicy},
		{"request ID", provider.RequestID},
	} {
		if err := validateText(field[0], field[1]); err != nil {
			return err
		}
	}
	return nil
}

func validateNativeSet(label string, set NativeSet) error {
	if len(set.Artifacts) != 2 {
		return fmt.Errorf("%s artifact set must contain exactly two files", label)
	}
	if err := validateArtifact(label+" CLI", set.Artifacts[0], "velox.exe"); err != nil {
		return err
	}
	return validateArtifact(label+" host", set.Artifacts[1], "velox-host.exe")
}

func validateDistribution(distribution Distribution) error {
	checks := []struct {
		label string
		value Artifact
		name  string
	}{
		{"release archive", distribution.Archive, "velox-windows-x64.zip"},
		{"release manifest", distribution.Manifest, "release-manifest.json"},
		{"checksums", distribution.Checksums, "checksums.sha256"},
		{"SBOM", distribution.SBOM, "velox-windows-x64.spdx.json"},
	}
	for _, check := range checks {
		if err := validateArtifact(check.label, check.value, check.name); err != nil {
			return err
		}
	}
	return nil
}

func validateAttestation(attestation Attestation, kind string, subject Artifact) error {
	if attestation.Kind != kind {
		return fmt.Errorf("attestation kind %q must be %q", attestation.Kind, kind)
	}
	if attestation.Subject != subject {
		return fmt.Errorf("%s attestation subject differs from distribution", kind)
	}
	if attestation.Status != StatusNotPerformed && attestation.Status != StatusVerified {
		return fmt.Errorf("unsupported attestation status %q", attestation.Status)
	}
	return nil
}

func validateCertificate(certificate Certificate) error {
	if certificate.Status != StatusVerified {
		return errors.New("release certificate status must be verified")
	}
	for _, field := range [][2]string{
		{"certificate subject", certificate.Subject},
		{"certificate issuer", certificate.Issuer},
		{"certificate serial", certificate.Serial},
		{"timestamp authority", certificate.TimestampAuthority},
	} {
		if err := validateText(field[0], field[1]); err != nil {
			return err
		}
	}
	return nil
}

func validateArtifact(label string, artifact Artifact, expectedName string) error {
	if artifact.File != expectedName {
		return fmt.Errorf("%s file must be %s", label, expectedName)
	}
	if artifact.Bytes < 1 {
		return fmt.Errorf("%s byte size must be positive", label)
	}
	if !digestPattern.MatchString(artifact.SHA256) {
		return fmt.Errorf("%s SHA-256 is invalid", label)
	}
	return nil
}

func validateText(label, value string) error {
	if value == "" || len(value) > 512 || strings.TrimSpace(value) != value {
		return fmt.Errorf("%s must contain 1 to 512 trimmed characters", label)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf("%s contains a control character", label)
		}
	}
	return nil
}

func readRegularFile(path string, limit int64) ([]byte, error) {
	linkInfo, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !linkInfo.Mode().IsRegular() || linkInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("input must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || !os.SameFile(linkInfo, info) {
		return nil, errors.New("input changed while opening")
	}
	reader := io.LimitReader(file, limit+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("signing record exceeds size limit")
	}
	return data, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing signing record data: %w", err)
	}
	return errors.New("signing record contains multiple JSON values")
}
