package releaseevidence

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/0disoft/velox/internal/buildinfo"
	"github.com/0disoft/velox/internal/releasebundle"
)

const (
	SBOMFile       = "velox-windows-x64.spdx.json"
	ProvenanceFile = "velox-windows-x64.intoto.jsonl"
	ChecksumsFile  = "checksums.sha256"
)

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

type Options struct {
	ReleaseDirectory string
	ReleaseArchive   string
	OutputRoot       string
	SourceRepository string
	SourceCommit     string
	InvocationID     string
	CreatedAt        time.Time
}

type Result struct {
	ArchiveSHA256    string `json:"archiveSha256"`
	SBOM             string `json:"sbom"`
	SBOMSHA256       string `json:"sbomSha256"`
	Provenance       string `json:"provenance"`
	ProvenanceSHA256 string `json:"provenanceSha256"`
	Checksums        string `json:"checksums"`
}

type spdxDocument struct {
	SPDXVersion       string             `json:"spdxVersion"`
	DataLicense       string             `json:"dataLicense"`
	SPDXID            string             `json:"SPDXID"`
	Name              string             `json:"name"`
	DocumentNamespace string             `json:"documentNamespace"`
	CreationInfo      spdxCreationInfo   `json:"creationInfo"`
	Packages          []spdxPackage      `json:"packages"`
	Files             []spdxFile         `json:"files"`
	Relationships     []spdxRelationship `json:"relationships"`
}

type spdxCreationInfo struct {
	Created  string   `json:"created"`
	Creators []string `json:"creators"`
}

type spdxPackage struct {
	Name                    string                  `json:"name"`
	SPDXID                  string                  `json:"SPDXID"`
	VersionInfo             string                  `json:"versionInfo"`
	DownloadLocation        string                  `json:"downloadLocation"`
	FilesAnalyzed           bool                    `json:"filesAnalyzed"`
	LicenseConcluded        string                  `json:"licenseConcluded"`
	LicenseDeclared         string                  `json:"licenseDeclared"`
	CopyrightText           string                  `json:"copyrightText"`
	PackageVerificationCode spdxPackageVerification `json:"packageVerificationCode"`
}

type spdxPackageVerification struct {
	Value string `json:"packageVerificationCodeValue"`
}

type spdxFile struct {
	FileName         string         `json:"fileName"`
	SPDXID           string         `json:"SPDXID"`
	Checksums        []spdxChecksum `json:"checksums"`
	LicenseConcluded string         `json:"licenseConcluded"`
	CopyrightText    string         `json:"copyrightText"`
}

type spdxChecksum struct {
	Algorithm     string `json:"algorithm"`
	ChecksumValue string `json:"checksumValue"`
}

type spdxRelationship struct {
	SPDXElementID      string `json:"spdxElementId"`
	RelationshipType   string `json:"relationshipType"`
	RelatedSPDXElement string `json:"relatedSpdxElement"`
}

type statement struct {
	Type          string            `json:"_type"`
	Subject       []subject         `json:"subject"`
	PredicateType string            `json:"predicateType"`
	Predicate     provenancePayload `json:"predicate"`
}

type subject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

type provenancePayload struct {
	BuildDefinition buildDefinition `json:"buildDefinition"`
	RunDetails      runDetails      `json:"runDetails"`
}

type buildDefinition struct {
	BuildType            string               `json:"buildType"`
	ExternalParameters   map[string]string    `json:"externalParameters"`
	InternalParameters   map[string]string    `json:"internalParameters"`
	ResolvedDependencies []resolvedDependency `json:"resolvedDependencies"`
}

type resolvedDependency struct {
	URI    string            `json:"uri"`
	Digest map[string]string `json:"digest"`
}

type runDetails struct {
	Builder  map[string]string `json:"builder"`
	Metadata map[string]string `json:"metadata"`
}

type fileEvidence struct {
	Path   string
	SHA1   string
	SHA256 string
}

func Build(options Options) (Result, error) {
	if options.ReleaseDirectory == "" || options.ReleaseArchive == "" || options.OutputRoot == "" {
		return Result{}, errors.New("release directory, archive, and output root are required")
	}
	if options.SourceRepository == "" || !commitPattern.MatchString(options.SourceCommit) || options.InvocationID == "" {
		return Result{}, errors.New("source repository, lowercase 40-character commit, and invocation ID are required")
	}
	if options.CreatedAt.IsZero() {
		return Result{}, errors.New("created time is required")
	}
	if err := verifyReleaseManifest(options.ReleaseDirectory); err != nil {
		return Result{}, err
	}
	archiveSHA, _, err := hashFile(options.ReleaseArchive)
	if err != nil {
		return Result{}, fmt.Errorf("hash release archive: %w", err)
	}
	files, err := inventory(options.ReleaseDirectory)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(options.OutputRoot, 0o755); err != nil {
		return Result{}, fmt.Errorf("create evidence output: %w", err)
	}

	sbomPath := filepath.Join(options.OutputRoot, SBOMFile)
	provenancePath := filepath.Join(options.OutputRoot, ProvenanceFile)
	checksumsPath := filepath.Join(options.OutputRoot, ChecksumsFile)
	sbom := buildSBOM(options, archiveSHA, files)
	if err := writeJSON(sbomPath, sbom, false); err != nil {
		return Result{}, err
	}
	provenance := buildProvenance(options, archiveSHA)
	if err := writeJSON(provenancePath, provenance, true); err != nil {
		return Result{}, err
	}
	sbomSHA, _, err := hashFile(sbomPath)
	if err != nil {
		return Result{}, err
	}
	provenanceSHA, _, err := hashFile(provenancePath)
	if err != nil {
		return Result{}, err
	}
	checksums := fmt.Sprintf("%s  %s\n%s  %s\n%s  %s\n", archiveSHA, filepath.Base(options.ReleaseArchive), provenanceSHA, ProvenanceFile, sbomSHA, SBOMFile)
	if err := os.WriteFile(checksumsPath, []byte(checksums), 0o644); err != nil {
		return Result{}, fmt.Errorf("write checksums: %w", err)
	}
	return Result{ArchiveSHA256: archiveSHA, SBOM: sbomPath, SBOMSHA256: sbomSHA, Provenance: provenancePath, ProvenanceSHA256: provenanceSHA, Checksums: checksumsPath}, nil
}

func verifyReleaseManifest(root string) error {
	data, err := os.ReadFile(filepath.Join(root, "release-manifest.json"))
	if err != nil {
		return fmt.Errorf("read release manifest: %w", err)
	}
	var manifest releasebundle.Manifest
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("decode release manifest: %w", err)
	}
	if manifest.SchemaVersion != releasebundle.SchemaVersion || manifest.ReleaseVersion != buildinfo.Version || manifest.Target != releasebundle.TargetWindowsX64 {
		return errors.New("release manifest identity differs from evidence builder")
	}
	for _, artifact := range manifest.Artifacts {
		path := filepath.Join(root, filepath.FromSlash(artifact.File))
		digest, size, err := hashFile(path)
		if err != nil {
			return fmt.Errorf("verify release artifact %s: %w", artifact.File, err)
		}
		if digest != artifact.SHA256 || size != artifact.Bytes {
			return fmt.Errorf("release artifact %s differs from manifest", artifact.File)
		}
	}
	return nil
}

func inventory(root string) ([]fileEvidence, error) {
	var files []fileEvidence
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("release evidence input must be regular: %s", path)
		}
		sha256Value, _, err := hashFile(path)
		if err != nil {
			return err
		}
		sha1Value, err := hashFileSHA1(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, fileEvidence{Path: filepath.ToSlash(relative), SHA1: sha1Value, SHA256: sha256Value})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inventory release directory: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func buildSBOM(options Options, archiveSHA string, files []fileEvidence) spdxDocument {
	verification := sha1.New() // SPDX 2.3 defines packageVerificationCode as SHA-1 over file SHA-1 values.
	spdxFiles := make([]spdxFile, 0, len(files))
	relationships := []spdxRelationship{{SPDXElementID: "SPDXRef-DOCUMENT", RelationshipType: "DESCRIBES", RelatedSPDXElement: "SPDXRef-Package-Velox"}}
	for index, file := range files {
		_, _ = io.WriteString(verification, file.SHA1)
		id := fmt.Sprintf("SPDXRef-File-%04d", index+1)
		spdxFiles = append(spdxFiles, spdxFile{FileName: "./" + file.Path, SPDXID: id, Checksums: []spdxChecksum{{Algorithm: "SHA256", ChecksumValue: file.SHA256}}, LicenseConcluded: "NOASSERTION", CopyrightText: "NOASSERTION"})
		relationships = append(relationships, spdxRelationship{SPDXElementID: "SPDXRef-Package-Velox", RelationshipType: "CONTAINS", RelatedSPDXElement: id})
	}
	return spdxDocument{
		SPDXVersion: "SPDX-2.3", DataLicense: "CC0-1.0", SPDXID: "SPDXRef-DOCUMENT",
		Name:              "velox-windows-x64-" + buildinfo.Version,
		DocumentNamespace: strings.TrimSuffix(options.SourceRepository, "/") + "/sbom/" + options.SourceCommit + "/" + archiveSHA,
		CreationInfo:      spdxCreationInfo{Created: options.CreatedAt.UTC().Format(time.RFC3339), Creators: []string{"Tool: velox-release-evidence/" + buildinfo.Version}},
		Packages:          []spdxPackage{{Name: "velox-windows-x64", SPDXID: "SPDXRef-Package-Velox", VersionInfo: buildinfo.Version, DownloadLocation: "NOASSERTION", FilesAnalyzed: true, LicenseConcluded: "NOASSERTION", LicenseDeclared: "NOASSERTION", CopyrightText: "NOASSERTION", PackageVerificationCode: spdxPackageVerification{Value: hex.EncodeToString(verification.Sum(nil))}}},
		Files:             spdxFiles, Relationships: relationships,
	}
}

func buildProvenance(options Options, archiveSHA string) statement {
	return statement{
		Type:          "https://in-toto.io/Statement/v1",
		Subject:       []subject{{Name: filepath.Base(options.ReleaseArchive), Digest: map[string]string{"sha256": archiveSHA}}},
		PredicateType: "https://slsa.dev/provenance/v1",
		Predicate: provenancePayload{
			BuildDefinition: buildDefinition{
				BuildType:            "https://github.com/0disoft/velox/.github/workflows/alpha-evidence.yml@v1",
				ExternalParameters:   map[string]string{"target": releasebundle.TargetWindowsX64},
				InternalParameters:   map[string]string{"releaseVersion": buildinfo.Version},
				ResolvedDependencies: []resolvedDependency{{URI: strings.TrimSuffix(options.SourceRepository, "/") + "@" + options.SourceCommit, Digest: map[string]string{"gitCommit": options.SourceCommit}}},
			},
			RunDetails: runDetails{Builder: map[string]string{"id": strings.TrimSuffix(options.SourceRepository, "/") + "/actions"}, Metadata: map[string]string{"invocationId": options.InvocationID}},
		},
	}
}

func hashFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), info.Size(), nil
}

func hashFileSHA1(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeJSON(path string, value any, jsonLines bool) error {
	var data []byte
	var err error
	if jsonLines {
		data, err = json.Marshal(value)
	} else {
		data, err = json.MarshalIndent(value, "", "  ")
	}
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}
