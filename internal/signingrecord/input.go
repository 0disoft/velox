package signingrecord

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	actutumarchive "github.com/0disoft/actutum/internal/archive"
)

const SigningInputName = "actutum-signing-input.zip"

type SigningInputResult struct {
	Path     string   `json:"path"`
	Artifact Artifact `json:"artifact"`
}

func PrepareSigningInput(unsignedDirectory, output string) (result SigningInputResult, err error) {
	if unsignedDirectory == "" || output == "" {
		return SigningInputResult{}, errors.New("unsigned directory and output path are required")
	}
	if filepath.Base(output) != SigningInputName {
		return SigningInputResult{}, fmt.Errorf("signing input output must be named %s", SigningInputName)
	}
	unsigned := NativeSet{Artifacts: make([]Artifact, 0, 2)}
	inputs := make([]actutumarchive.Input, 0, 2)
	for _, name := range []string{"actutum.exe", "actutum-host.exe"} {
		source := filepath.Join(unsignedDirectory, name)
		artifact, inspectErr := inspectArtifact(source, name)
		if inspectErr != nil {
			return SigningInputResult{}, fmt.Errorf("inspect unsigned %s: %w", name, inspectErr)
		}
		unsigned.Artifacts = append(unsigned.Artifacts, artifact)
		inputs = append(inputs, actutumarchive.Input{Source: source, Name: name})
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return SigningInputResult{}, fmt.Errorf("create signing input directory: %w", err)
	}
	created := false
	defer func() {
		if err != nil && created {
			_ = os.Remove(output)
		}
	}()
	archiveResult, err := actutumarchive.CreateFiles(output, inputs)
	if err != nil {
		return SigningInputResult{}, fmt.Errorf("create signing input: %w", err)
	}
	created = true
	artifact, err := inspectArtifact(output, SigningInputName)
	if err != nil {
		return SigningInputResult{}, fmt.Errorf("inspect generated signing input: %w", err)
	}
	if artifact.Bytes != archiveResult.Size || artifact.SHA256 != archiveResult.SHA256 || archiveResult.FileCount != 2 {
		return SigningInputResult{}, errors.New("generated signing input result is inconsistent")
	}
	if err := verifySigningInput(output, unsigned); err != nil {
		return SigningInputResult{}, fmt.Errorf("verify generated signing input: %w", err)
	}
	return SigningInputResult{Path: output, Artifact: artifact}, nil
}
