package authenticode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	SchemaVersion = "actutum.authenticode-verification/v1"
	Target        = "windows-x64"
	DigestOID     = "2.16.840.1.101.3.4.2.1"
	DigestName    = "sha256"

	verificationTimeout = 30 * time.Second
)

var hexPattern = regexp.MustCompile(`^[0-9a-f]+$`)

type Artifact struct {
	File                string `json:"file"`
	Status              string `json:"status"`
	Subject             string `json:"subject"`
	Issuer              string `json:"issuer"`
	Serial              string `json:"serial"`
	Thumbprint          string `json:"thumbprint"`
	DigestAlgorithm     string `json:"digestAlgorithm"`
	DigestOID           string `json:"digestOid"`
	TimestampAuthority  string `json:"timestampAuthority"`
	TimestampSerial     string `json:"timestampSerial"`
	TimestampThumbprint string `json:"timestampThumbprint"`
}

type Result struct {
	SchemaVersion   string     `json:"schemaVersion"`
	Target          string     `json:"target"`
	ExpectedSubject string     `json:"expectedSubject"`
	Artifacts       []Artifact `json:"artifacts"`
}

type probeResult struct {
	Status              string `json:"status"`
	Subject             string `json:"subject"`
	Issuer              string `json:"issuer"`
	Serial              string `json:"serial"`
	Thumbprint          string `json:"thumbprint"`
	DigestOID           string `json:"digestOid"`
	TimestampSubject    string `json:"timestampSubject"`
	TimestampSerial     string `json:"timestampSerial"`
	TimestampThumbprint string `json:"timestampThumbprint"`
}

type probeFunc func(context.Context, string) (probeResult, error)

func VerifyDirectory(directory, expectedSubject string) (Result, error) {
	return verifyDirectory(directory, expectedSubject, probeAuthenticode)
}

func verifyDirectory(directory, expectedSubject string, probe probeFunc) (Result, error) {
	if err := validateText("expected subject", expectedSubject); err != nil {
		return Result{}, err
	}
	paths, err := exactArtifactPaths(directory)
	if err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), verificationTimeout)
	defer cancel()

	result := Result{SchemaVersion: SchemaVersion, Target: Target, ExpectedSubject: expectedSubject}
	for _, path := range paths {
		probed, err := probe(ctx, path)
		if err != nil {
			return Result{}, fmt.Errorf("verify %s: %w", filepath.Base(path), err)
		}
		artifact, err := validateProbe(filepath.Base(path), expectedSubject, probed)
		if err != nil {
			return Result{}, fmt.Errorf("verify %s: %w", filepath.Base(path), err)
		}
		result.Artifacts = append(result.Artifacts, artifact)
	}
	if err := requireSharedSigner(result.Artifacts); err != nil {
		return Result{}, err
	}
	return result, nil
}

func exactArtifactPaths(directory string) ([]string, error) {
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, fmt.Errorf("inspect signed directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, errors.New("signed directory must be a real directory")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read signed directory: %w", err)
	}
	want := map[string]bool{"actutum-host.exe": true, "actutum.exe": true}
	if len(entries) != len(want) {
		return nil, errors.New("signed directory must contain exactly actutum-host.exe and actutum.exe")
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !want[entry.Name()] {
			return nil, fmt.Errorf("unexpected signed artifact %q", entry.Name())
		}
		entryPath := filepath.Join(directory, entry.Name())
		entryInfo, err := os.Lstat(entryPath)
		if err != nil {
			return nil, fmt.Errorf("inspect signed artifact %s: %w", entry.Name(), err)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 || !entryInfo.Mode().IsRegular() || entryInfo.Size() == 0 {
			return nil, fmt.Errorf("signed artifact %s must be a non-empty regular file", entry.Name())
		}
		paths = append(paths, entryPath)
	}
	sort.Strings(paths)
	return paths, nil
}

func validateProbe(file, expectedSubject string, result probeResult) (Artifact, error) {
	if result.Status != "Valid" {
		return Artifact{}, fmt.Errorf("Authenticode status is %q", result.Status)
	}
	for label, value := range map[string]string{
		"subject": result.Subject, "issuer": result.Issuer,
		"timestamp authority": result.TimestampSubject,
	} {
		if err := validateText(label, value); err != nil {
			return Artifact{}, err
		}
	}
	if result.Subject != expectedSubject {
		return Artifact{}, fmt.Errorf("signer subject %q does not match expected subject %q", result.Subject, expectedSubject)
	}
	if result.DigestOID != DigestOID {
		return Artifact{}, fmt.Errorf("signature digest OID %q is not SHA-256", result.DigestOID)
	}
	for label, value := range map[string]string{
		"serial": result.Serial, "thumbprint": result.Thumbprint,
		"timestamp serial": result.TimestampSerial, "timestamp thumbprint": result.TimestampThumbprint,
	} {
		if err := validateHex(label, value); err != nil {
			return Artifact{}, err
		}
	}
	return Artifact{
		File: file, Status: "verified", Subject: result.Subject, Issuer: result.Issuer,
		Serial: strings.ToLower(result.Serial), Thumbprint: strings.ToLower(result.Thumbprint),
		DigestAlgorithm: DigestName, DigestOID: DigestOID,
		TimestampAuthority:  result.TimestampSubject,
		TimestampSerial:     strings.ToLower(result.TimestampSerial),
		TimestampThumbprint: strings.ToLower(result.TimestampThumbprint),
	}, nil
}

func requireSharedSigner(artifacts []Artifact) error {
	if len(artifacts) != 2 {
		return errors.New("Authenticode verification requires exactly two artifacts")
	}
	first, second := artifacts[0], artifacts[1]
	if first.Subject != second.Subject || first.Issuer != second.Issuer || first.Serial != second.Serial || first.Thumbprint != second.Thumbprint {
		return errors.New("signed artifacts do not share one signer certificate")
	}
	return nil
}

func validateText(label, value string) error {
	if value == "" || len(value) > 4096 || strings.TrimSpace(value) != value {
		return fmt.Errorf("%s is invalid", label)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s contains control characters", label)
		}
	}
	return nil
}

func validateHex(label, value string) error {
	value = strings.ToLower(value)
	if len(value) < 2 || len(value) > 256 || len(value)%2 != 0 || !hexPattern.MatchString(value) {
		return fmt.Errorf("%s is not canonical hexadecimal", label)
	}
	return nil
}
