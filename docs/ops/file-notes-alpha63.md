# File Notes 0.4.0 With Public Alpha.63: 2026-10-05

## Artifact Identity

File Notes source at `970d12259e9419f4290aa021fe20e22f43468ac9` was packaged
with the independently verified public `v0.5.10-alpha.63` CLI. Application
version remains `0.4.0`; no source or manifest change was needed. The old
`dist/examples/file-notes` beta.20 output was preserved.

Public runtime source: `fce9955bdb355fd1b1a377dec277a60727c4ad39`.
Release ZIP SHA-256:
`19205e691e79dcddaeeb414cbbeb4cb055e59344d85dabfd5f7bfe5ea99b27ca`.
The packaging command invoked only the public `velox.exe`, with an absolute
fresh output path and `--installer`; it did not compile application code.
Local receipt inspection and the native smoke controller used Go separately.

Local output root:
`dist/distribution/file-notes-0.4.0-alpha63-970d122/`.

| Artifact | Bytes | SHA-256 |
| --- | --- | --- |
| `dev.velox.filenotes.zip` | 8,155,482 | `e4297f6d7a0f8bd5341a2ff992e6a7f375f486b38b41226b2ac29c61f141ceca` |
| `dev.velox.filenotes-setup.exe` | 11,730,330 | `8272091649420ad0f92def87a709d6476d434d0ed6e783e5dc35d028715ad8b3` |
| `dev.velox.filenotes/dev.velox.filenotes.exe` | 4,414,464 | `6b77efbba3545b737389a6443a3641a214df58d831fc622aabb25b409971c6a3` |

`checksums.sha256`, `verification-result.json` and `startup-result.json` are
beside the artifacts. These local File Notes files are not public release
assets. Their checksum set does not authenticate the publisher.

## Validation

- Public CLI validate and doctor passed. Doctor observed Windows client build
  `10.0.26200`, WebView2 `154.0.4258.53` and compatible alpha.63 host metadata.
- Public CLI build with `--installer` passed; ZIP and directory inspection
  matched release/app versions, 17 portable files and unchanged permissions:
  `file.open`, `file.save`, `window.title`.
- All 14 source web assets matched packaged bytes, including Noto Sans KR and
  its CSS reference. Aggregate asset SHA-256 remains
  `b7e054a5aa14edcae39fda9fb116236fa147435acedca8f36fbec7779b2bbc1a`.
- All 20 embedded icon/group-icon resource leaves matched the public host.
  The branded GUI PE retains the default Velox icon. Windows version metadata
  reports Velox File Notes, application/file version `0.4.0`, company Velox and
  description Local Markdown editor.
- Host and Setup Authenticode status was `NotSigned`; host PE subsystem is
  Windows GUI (2). Branding changes host bytes, so its digest is distinct from
  the public generic template.
- The installer payload was opened, hash-checked, safely extracted and
  inspected without executing Setup. Its embedded ZIP is 8,155,482 bytes;
  its 3,574,784-byte template exactly matches the public Setup template.
  Verification-only extracted files were moved to an owned `.cache` folder.
- The actual packaged File Notes executable was started once with its
  unchanged assets and a fresh `VELOX_DATA_DIR` under `.cache`. The benchmark
  readiness marker was `ready dom-2raf 36812` at 1,071.105 ms, then host exit 0,
  complete window-destroy/run-loop shutdown and browser exit were observed.
  Profile isolation also isolates the single-instance identity; the existing
  app/profile was not activated, closed or copied.

This readiness callback follows draft restoration and two animation frames.
It is a startup/shutdown smoke, not a new manual rendering, font appearance,
save/open or data-recovery confirmation, and not a performance comparison.
No installer execution, registry/Start Menu changes, install/removal test,
native permission interaction, stress matrix or second reproducibility build
was performed. Existing source/interaction evidence retains its original
version boundary. No API, DB, dependency, CI or release-channel change.

## Isolated Installer Follow-Up: 2026-10-05

The owner subsequently reported that the packaged File Notes app worked.
The installer follow-up used the public alpha.63 CLI with a disposable app ID,
`dev.velox.installer-smoke-d70655675121`, and name suffix `Installer Smoke`.
All 14 File Notes web assets matched the source; the original manifest,
application installation and user profile were not modified. This is not an
execution of the original `dev.velox.filenotes-setup.exe` bytes above.

- Two builds produced identical test Setup bytes, SHA-256
  `0b40474871722e9a210bcab46b2c00badc01fb5edac0e91b5a878e83b8fc3c8a`.
- Silent per-user install, installed-directory inspection, actual Start Menu
  shortcut launch and uninstall registration checks passed.
- The native window caption was `Velox File Notes - Installer Smoke`.
  Native close completed with host exit 0 and browser exit. This shortcut
  check does not claim a DOM readiness marker or manual rendered-content test.
- Actual removal cleared the installation directory, shortcut and HKCU
  uninstall registration. The disposable document was unchanged.
- All 167 profile files, including five IndexedDB files and a synthetic
  preservation marker, had identical SHA-256 values before and after removal.
  Actual saved-draft restoration was not exercised. The retained test profile
  was moved into the owned cache receipt directory after this comparison;
  the temporary removal helper was cleaned up.

Command: `node scripts/installer-smoke.mjs --cli dist/releases/alpha63-37224406021/public-extracted/velox.exe --example examples/file-notes`.
Receipt: `.cache/installer-smoke-84f0b364-64af-4c97-81a9-d70655675121/result.json`,
SHA-256 `5711ab5585073e58e50d3b8063cfeca097825ddcbfe3f211077da1e240a3c19d`.
The script now accepts explicit CLI/example paths and uses a unique identity.
Earlier benchmark-driven shortcut attempts timed out and were cleaned up;
the passing run used normal shortcut launch and native window close instead.
No performance or benchmark-readiness claim follows from those attempts.
Node syntax and source/copy asset-hash checks passed. Runtime, API, DB,
dependencies, CI and release versions are unchanged; no publication occurred.

## Use

Extract the whole portable ZIP before running `dev.velox.filenotes.exe`, and
keep its web directory and runtime files together. The pre-extracted app is
already available under the output root's `dev.velox.filenotes/` directory.
The alternative Setup is unsigned and per-user; it has not been installed in
this preparation step. Neither package is an automatic update or beta release.
