# beta.18 Local Distribution Preparation

- Date: 2026-10-03
- Status: Prepared locally; Folder Browser install/removal verified; unsigned; not published
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

## Folder Browser Install/Removal Verification: 2026-10-03

The exact Folder Browser Setup listed above was installed with `--silent` after
confirming that its install directory, Start Menu shortcut and uninstall
registration were absent. No existing installation was replaced.

- Setup exited successfully. All eight installed ownership-record files and
  the Start Menu shortcut matched their recorded SHA-256 digests.
- The uninstall registration reported `Velox Folder Browser` version `0.2.0`
  and the expected per-user install location and uninstall command.
- CLI inspection of the installed app passed with runtime `0.5.10-beta.18`,
  app version `0.2.0`, and only `folder.read` / `folder.readText` permissions.
- Launch through the generated Start Menu shortcut opened the installed EXE.
  The maintainer confirmed the requested folder listing and README.md text
  preview check, then closed the app. This is manual interaction evidence,
  not automated UI verification.
- After confirming that the app and its profile's WebView2 processes had
  exited, the installed uninstaller ran with `--silent`. Its helper removed
  the install directory, Start Menu shortcut and uninstall registration.
- The 217 files in the app profile and seven files in the selected
  `examples/folder-browser` directory had identical file counts and SHA-256
  digests immediately before and after removal. The app process was absent.
- The normal `%TEMP%/velox-uninstall-*` helper directory remains, as documented
  in [Windows Installer](windows-installer.md). It is not a profile or document
  deletion, and complete temporary-file cleanup is not claimed.

These checks used direct local commands and maintainer confirmation, not a
Mustflow receipt. Only Folder Browser was installed; the final File Notes Setup
and non-silent Setup confirmation dialogs were not exercised. No build or CI
rerun was needed for this evidence-only update.

## Delivery Boundary

Folder Browser's local installation, shortcut launch, manual interaction and
removal are verified above. File Notes final-package installation is not
claimed. No tag, GitHub Release, upload or public publication was performed.
Existing beta.18 manual preview evidence and intermittent local early-close /
packaging failures remain recorded
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
