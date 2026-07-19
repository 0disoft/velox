//go:build windows

package authenticode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

const powershellProbe = `$ErrorActionPreference = 'Stop'
$path = [Environment]::GetEnvironmentVariable('VELOX_AUTHENTICODE_FILE', 'Process')
if ([string]::IsNullOrWhiteSpace($path)) { throw 'VELOX_AUTHENTICODE_FILE is required' }
$module = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\Modules\Microsoft.PowerShell.Security\Microsoft.PowerShell.Security.psd1'
Import-Module -Name $module -Force -ErrorAction Stop
$signature = Microsoft.PowerShell.Security\Get-AuthenticodeSignature -LiteralPath $path
if ($null -eq $signature.SignerCertificate) { throw ('Authenticode status is ' + [string]$signature.Status) }
$bytes = [IO.File]::ReadAllBytes($path)
if ($bytes.Length -lt 256 -or $bytes[0] -ne 0x4d -or $bytes[1] -ne 0x5a) { throw 'file is not a PE image' }
$peOffset = [BitConverter]::ToInt32($bytes, 0x3c)
if ($peOffset -lt 64 -or $peOffset + 24 -gt $bytes.Length) { throw 'PE header offset is invalid' }
if ($bytes[$peOffset] -ne 0x50 -or $bytes[$peOffset + 1] -ne 0x45 -or $bytes[$peOffset + 2] -ne 0 -or $bytes[$peOffset + 3] -ne 0) { throw 'PE signature is invalid' }
$optionalOffset = $peOffset + 24
$magic = [BitConverter]::ToUInt16($bytes, $optionalOffset)
if ($magic -eq 0x20b) { $directoryOffset = $optionalOffset + 112 }
elseif ($magic -eq 0x10b) { $directoryOffset = $optionalOffset + 96 }
else { throw 'PE optional header is unsupported' }
$securityDirectory = $directoryOffset + (8 * 4)
if ($securityDirectory + 8 -gt $bytes.Length) { throw 'PE security directory is missing' }
$certificateOffset = [BitConverter]::ToUInt32($bytes, $securityDirectory)
$certificateSize = [BitConverter]::ToUInt32($bytes, $securityDirectory + 4)
if ($certificateOffset -eq 0 -or $certificateSize -lt 8 -or [uint64]$certificateOffset + [uint64]$certificateSize -gt [uint64]$bytes.Length) { throw 'PE certificate table is invalid' }
$certificateLength = [BitConverter]::ToUInt32($bytes, [int]$certificateOffset)
$certificateType = [BitConverter]::ToUInt16($bytes, [int]$certificateOffset + 6)
if ($certificateType -ne 2 -or $certificateLength -lt 9 -or $certificateLength -gt $certificateSize) { throw 'PE certificate entry is unsupported' }
$payload = New-Object byte[] ([int]$certificateLength - 8)
[Array]::Copy($bytes, [int]$certificateOffset + 8, $payload, 0, $payload.Length)
Add-Type -AssemblyName System.Security
$cms = New-Object System.Security.Cryptography.Pkcs.SignedCms
$cms.Decode($payload)
if ($cms.SignerInfos.Count -ne 1) { throw 'Authenticode payload must contain one primary signer' }
$timestamp = $signature.TimeStamperCertificate
$timestampSubject = ''
$timestampSerial = ''
$timestampThumbprint = ''
if ($null -ne $timestamp) {
  $timestampSubject = [string]$timestamp.Subject
  $timestampSerial = [string]$timestamp.SerialNumber
  $timestampThumbprint = [string]$timestamp.Thumbprint
}
[ordered]@{
  status = [string]$signature.Status
  subject = [string]$signature.SignerCertificate.Subject
  issuer = [string]$signature.SignerCertificate.Issuer
  serial = [string]$signature.SignerCertificate.SerialNumber
  thumbprint = [string]$signature.SignerCertificate.Thumbprint
  digestOid = [string]$cms.SignerInfos[0].DigestAlgorithm.Value
  timestampSubject = $timestampSubject
  timestampSerial = $timestampSerial
  timestampThumbprint = $timestampThumbprint
} | ConvertTo-Json -Compress
`

func probeAuthenticode(ctx context.Context, path string) (probeResult, error) {
	root, err := windows.GetWindowsDirectory()
	if err != nil {
		return probeResult{}, fmt.Errorf("resolve Windows directory: %w", err)
	}
	systemDirectory, err := windows.GetSystemDirectory()
	if err != nil {
		return probeResult{}, fmt.Errorf("resolve Windows system directory: %w", err)
	}
	powershell := filepath.Join(systemDirectory, "WindowsPowerShell", "v1.0", "powershell.exe")
	info, err := os.Lstat(powershell)
	if err != nil || !info.Mode().IsRegular() {
		return probeResult{}, errors.New("Windows PowerShell 5.1 is unavailable")
	}
	command := exec.CommandContext(ctx, powershell, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", powershellProbe)
	command.Env = probeEnvironment(root, path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return probeResult{}, errors.New("Authenticode verification timed out")
		}
		message := strings.TrimSpace(stderr.String())
		if len(message) > 1024 {
			message = message[:1024]
		}
		if message == "" {
			message = err.Error()
		}
		return probeResult{}, fmt.Errorf("Windows Authenticode probe failed: %s", message)
	}
	if stdout.Len() > 64*1024 {
		return probeResult{}, errors.New("Windows Authenticode probe output is too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.DisallowUnknownFields()
	var result probeResult
	if err := decoder.Decode(&result); err != nil {
		return probeResult{}, fmt.Errorf("decode Windows Authenticode probe: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return probeResult{}, errors.New("Windows Authenticode probe returned trailing data")
	}
	return result, nil
}

func probeEnvironment(root, path string) []string {
	environment := []string{
		"SystemRoot=" + root,
		"WINDIR=" + root,
		"PSModulePath=" + filepath.Join(root, "System32", "WindowsPowerShell", "v1.0", "Modules"),
		"VELOX_AUTHENTICODE_FILE=" + path,
	}
	for _, name := range []string{"TEMP", "TMP"} {
		if value := os.Getenv(name); value != "" {
			environment = append(environment, name+"="+value)
		}
	}
	return environment
}
