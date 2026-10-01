package setuppayload

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"

	"github.com/0disoft/velox/internal/buildinfo"
	"github.com/0disoft/velox/internal/releasebundle"
	"github.com/0disoft/velox/internal/safefs"
)

// VerifyTemplate binds the adjacent setup template to the current release's
// artifact inventory. This is compatibility/integrity checking, not signing.
func VerifyTemplate(path string) error {
	metadata, _, err := safefs.OpenVerifiedRegular(filepath.Join(filepath.Dir(path), "release-manifest.json"))
	if err != nil {
		return err
	}
	defer metadata.Close()
	var manifest releasebundle.Manifest
	decoder := json.NewDecoder(io.LimitReader(metadata, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("release manifest has trailing data")
	}
	contracts := releasebundle.Contracts{Manifest: 1, Runtime: 1, Host: 1, IPC: 1, BuildResult: 1}
	if manifest.SchemaVersion != releasebundle.SchemaVersion || manifest.ReleaseVersion != buildinfo.Version || manifest.Target != releasebundle.TargetWindowsX64 || manifest.Contracts != contracts {
		return errors.New("setup release is incompatible")
	}
	var expected *releasebundle.Artifact
	for _, artifact := range manifest.Artifacts {
		if artifact.File == "velox-setup.exe" {
			if expected != nil {
				return errors.New("duplicate setup artifact")
			}
			copy := artifact
			expected = &copy
		}
	}
	if expected == nil {
		return errors.New("release bundle does not contain a setup template")
	}
	file, info, err := safefs.OpenVerifiedRegular(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if expected.Bytes != info.Size() || info.Size() > maxTemplateSize {
		return errors.New("setup template size mismatch")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected.SHA256 {
		return errors.New("setup template checksum mismatch")
	}
	return nil
}
