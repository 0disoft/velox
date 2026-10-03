# beta.18 Local Distribution Preparation

- Date: 2026-10-03
- Status: Prepared locally; unsigned; not published or installed
- Source: `b79fc01def6b88da44574032cd14c80ec36e75bb`
- Output root: `dist/distribution/beta18/`
- Runtime: `0.5.10-beta.18`; both examples: `0.2.0`
- Owner: Project maintainer

## Contents

| Artifact Relative to Output Root | Bytes | SHA-256 |
| --- | ---: | --- |
| `runtime/velox-windows-x64.zip` | 6717002 | `0938b6efe0045258147e4423e1ccdc1af1797d49542bbed4ff394f571e912291` |
| `examples/file-notes/dev.velox.filenotes.zip` | 8417722 | `d96dec8937deebc6e253d9cf7f36e16d146bd0a93723a77b8c64df3db689a43e` |
| `examples/file-notes/dev.velox.filenotes-setup.exe` | 12512250 | `69a37dbbb5faffa8dea24a47196c87b521cb13941e5af6e284b827c1d74eab2d` |
| `examples/folder-browser/dev.velox.folderbrowser.zip` | 3130337 | `58d7e091e4f8bc04b3fde737785d32767b8b564d2a22eacdd9898330f531a860` |
| `examples/folder-browser/dev.velox.folderbrowser-setup.exe` | 7224865 | `3d3b4729f97637a0eb5413fda511b4091e1bef1c07e7e7324c60e68f339409b8` |

The runtime ZIP includes CLI, GUI host and unsigned GUI Setup template built
from the same clean source. This preparation uses a version-specific directory
and does not replace existing app or preview outputs. `README.md` and
`SHA256SUMS` in the output root are local distribution companions, not inputs
inside the five hashed artifacts.

## Verification

- Existing [Windows Consumer evidence run 37110431391](https://github.com/0disoft/velox/actions/runs/37110431391)
  completed successfully at the exact source commit above. No additional run was dispatched.
- All three source builds passed with `GOPROXY=off` and `GOSUMDB=off`; no
  dependency download, frontend build or consumer compilation was required.
- Local toolchain: Go 1.27.1. Embedded Go build metadata in all three binaries
  reports the exact source commit above and `vcs.modified=false`. The hosted
  source checks do not imply byte equality with these locally built binaries.
- Installer-enabled runtime assembly and both `build --installer` operations passed.
- CLI ZIP inspection passed for each example, with beta.18 metadata, expected
  app identity/version and unchanged permission sets.
- All 13 runtime-manifest artifact sizes and SHA-256 digests matched their files.
- Both Setup footers, bounds, embedded payload digests, exact template prefixes
  and exact external ZIP payload bytes matched. The Setup template is
  4,094,464 bytes.
- Final checksum-list and distribution-guide consistency are checked as part
  of preparation. This is direct local command evidence, not a Mustflow receipt.

## Delivery Boundary

No app installation, registry/shortcut mutation, app launch, tag, GitHub Release,
upload or public publication was performed. Final-package runtime interaction
and fresh install/removal are not newly claimed. Existing beta.18 manual preview
evidence and intermittent local early-close/packaging failures remain recorded
in [product readiness](product-readiness.md). A successful hosted run does not
prove those intermittent local causes were fixed.

The packages are unsigned Windows x64 previews requiring existing Evergreen
WebView2. SmartScreen or managed-device restrictions may apply. Existing
installations are refused rather than upgraded: close and uninstall first;
user documents and app profiles are preserved. Use the portable ZIP when an
installation is not desired. Public channel admission and signing remain
separate decisions under the existing ADRs.

This preparation changes only delivery records, not runtime APIs, schemas,
dependencies, DB, repository hygiene or CI workflows. The evidence-only docs
commit does not change the binary source commit or require another version bump.
