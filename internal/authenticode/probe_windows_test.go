//go:build windows

package authenticode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProbeAuthenticodeReadsEmbeddedSystemSignature(t *testing.T) {
	var failures []string
	for _, path := range signedExecutableCandidates() {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), verificationTimeout)
		result, err := probeAuthenticode(ctx, path)
		cancel()
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if result.Status != "Valid" || result.Subject == "" || result.Issuer == "" || result.Serial == "" || result.Thumbprint == "" {
			t.Fatalf("signer result for %s = %#v", path, result)
		}
		if result.DigestOID != DigestOID {
			t.Fatalf("digest OID for %s = %q", path, result.DigestOID)
		}
		if result.TimestampSubject == "" || result.TimestampSerial == "" || result.TimestampThumbprint == "" {
			t.Fatalf("timestamp result for %s = %#v", path, result)
		}
		t.Logf("verified embedded Authenticode fixture %s with signer %s", path, result.Subject)
		return
	}
	t.Skipf("no embedded SHA-256 Authenticode fixture is installed: %s", strings.Join(failures, "; "))
}

func signedExecutableCandidates() []string {
	root := os.Getenv("SystemRoot")
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	candidates := []string{
		filepath.Join(programFiles, "PowerShell", "7", "pwsh.exe"),
		filepath.Join(programFiles, "Git", "cmd", "git.exe"),
		filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(root, "explorer.exe"),
		filepath.Join(root, "System32", "notepad.exe"),
	}
	patterns := []string{
		filepath.Join(programFilesX86, "Microsoft", "EdgeWebView", "Application", "*", "msedgewebview2.exe"),
		filepath.Join(programFilesX86, "Microsoft", "Edge", "Application", "*", "msedge.exe"),
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		candidates = append(candidates, matches...)
	}
	return candidates
}
