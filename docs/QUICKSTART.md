# Velox Release Quickstart

- Audience: Windows developers and clean-room coding agents
- Starting point: one immutable public Velox release
- Consumer toolchain: none

This guide starts from published release bytes. It does not require a Velox
source checkout, Go, C++, Rust, Zig, Node.js, Bun, a frontend package manager,
or a build-system cache.

The PowerShell examples below are consumer-facing local commands. They are not
repository-maintainer command authority and do not replace the repository's
Mustflow validation intents.

## 1. Record the release identity

Open the [Velox Releases](https://github.com/0disoft/velox/releases) page and
choose one immutable `vX.Y.Z-alpha.N` or `vX.Y.Z-beta.N` release. Do not use a
moving `latest` URL for evaluation evidence.

Download these assets from that exact release:

- `velox-windows-x64.zip`
- `checksums.sha256`

Record the release tag, release URL, and expected ZIP SHA-256 before executing
anything. A checksum downloaded from the same release is an integrity check,
not an independent publisher identity or authenticated attestation.

## 2. Verify the ZIP

In a new empty working directory containing the two downloaded files:

```powershell
$ChecksumMatches = @(Get-Content -LiteralPath .\checksums.sha256 |
  Where-Object { $_ -match '^[0-9A-Fa-f]{64}\s+velox-windows-x64[.]zip$' })
if ($ChecksumMatches.Count -ne 1) {
  throw "Expected exactly one Velox ZIP checksum."
}
$Expected = ($ChecksumMatches[0] -split '\s+')[0].ToLowerInvariant()
$Observed = (Get-FileHash -LiteralPath .\velox-windows-x64.zip -Algorithm SHA256).Hash.ToLowerInvariant()
if ($Expected -ne $Observed) {
  throw "Velox release checksum mismatch."
}
```

For clean-room evaluation, also require `$Observed` to equal the independently
supplied expected digest. Stop before extraction when either comparison fails.

## 3. Extract the release

```powershell
Expand-Archive -LiteralPath .\velox-windows-x64.zip -DestinationPath .\tool
$Velox = (Resolve-Path -LiteralPath .\tool\velox-windows-x64\velox.exe).Path
```

Keep `$Velox` as an explicit absolute executable path during the evaluation.
The `velox.exe` name collides with unrelated software and should not be assumed
to resolve safely through `PATH`.

## 4. Exercise the public CLI

Create all project and output files under this clean working directory:

```powershell
& $Velox version --json
& $Velox init .\work\hello --json
& $Velox validate --config .\work\hello\velox.json --json
& $Velox doctor --config .\work\hello\velox.json --out .\work\doctor --json
& $Velox build --config .\work\hello\velox.json --out .\work\dist --json
& $Velox inspect .\work\dist\dev.velox.hello.zip --json
& $Velox run --config .\work\hello\velox.json --out .\work\run --json
```

`run` stays attached to the desktop application. Close the application window
to let the command finish. A visible window alone is not proof of usable
content; verify that the generated page rendered before closing it.

## 5. Confirm the output boundary

The build should produce:

```text
work/dist/dev.velox.hello/
work/dist/dev.velox.hello.zip
```

The portable directory contains the unchanged generic host, runtime
configuration, static web assets, and `build-result.json`. The ZIP is unsigned,
and directory assets are not protected against a local writer. Windows
SmartScreen may warn, and managed Windows policy may block execution.

## Failure Boundaries

Stop and preserve the first stable diagnostic when:

- the release or artifact digest differs;
- `doctor` reports an unsupported Windows or WebView2 version;
- a command requests an undeclared compiler, runtime, package manager, network
  dependency, source checkout, or cache;
- inspection disagrees with the expected application identity;
- the application never reaches usable content.

Do not install another toolchain or substitute local source output to make the
trial pass. A localized failure is valid evaluation evidence.
