# Alpha.65 Source Candidate `11efd69`: 2026-10-05

## Identity

This is a local source-candidate preparation receipt, not the published
payload. The version string remains `0.5.10-alpha.65`, but this candidate's
bytes differ from the public alpha.65 prerelease; see
[the publication record](release.md#alpha65-published-preview-2026-10-05).
No new version, tag, release, hosted run, push or beta promotion was performed
in this step.

- Source: `11efd69a7dfeda04ec84e31b804b6af68cc29bad`, clean at build time.
- Version: `0.5.10-alpha.65` (string unchanged; not the public alpha.65 bytes).
- Toolchain: Go 1.27.1, Windows amd64, `-buildvcs=false`, `-trimpath`, `-s -w`;
  host and Setup also use `-H windowsgui`.
- Candidate root: `dist/candidates/source-11efd69/`.
- ZIP: `dist/candidates/source-11efd69/release/velox-windows-x64.zip`,
  6,863,300 bytes, SHA-256
  `7e57f3d18015923ee69873b8baa55516b788666ab78dcc3d714c8dd4596278dc`.

The candidate carries four local feature commits after the alpha.65 publication
receipt `bd0ad1e`: image/font development watch (`b131024`), opt-in metadata-only
JavaScript diagnostics (`49e7077`), bounded local draft storage (`2881775`)
and text-editor draft recovery (`11efd69`). The locally recorded `origin/main` is
`bd0ad1ebcca64b382fe6aa0279f15e51f746dd57`; the commits and this receipt are
local only.

| File | Bytes | SHA-256 |
| --- | --- | --- |
| `velox.exe` | 5,039,104 | `386b591217e41dd35f8ec30c2580c7b9323a903b84514570c73efd149c3a5b0e` |
| `velox-host.exe` | 4,861,440 | `2b6a0214b2c82eee8790273fb7426de7e405b9afb262ac007e74bea16c1f66d3` |
| `velox-setup.exe` | 4,099,584 | `3190fffa731039c52a614fa2cf2f8c7dedd37ec9b5d4100086f7628e8ba80786` |
| `velox-windows-x64.zip` | 6,863,300 | `7e57f3d18015923ee69873b8baa55516b788666ab78dcc3d714c8dd4596278dc` |

All three Authenticode statuses are `NotSigned`. The CLI, host and Setup were
built once. Each generated project's root `velox.d.ts` matches the shipped
`dist/candidates/source-11efd69/release/velox-windows-x64/types/velox.d.ts`,
both SHA-256
`246cb4e21da585f70490f15a1f4e066ce33ad7e04bdbbfa2079207c1fe43a77c`. No new
version, API, IPC, database, schema, dependency, CI or functional code change
is part of this preparation step.

## Verification

The release ZIP was extracted outside the checkout at
`%TEMP%\velox-source-11efd69-20261005` and only its prebuilt `velox.exe` was
used: `init`, `validate`, `build --installer` and `inspect` for both
templates. No compiler was used for consumer packaging. Both extracted
packaged EXEs byte-match the candidate host (`2b6a0214...`). The runtime
permission sets are unchanged: text-editor is exactly `file.open` and
`file.save`; folder-browser is exactly `folder.read` and `folder.readText`.
All four generation/validate/build/inspect actions for each template exited 0. The ZIP and Setup
were generated, not installed.

| Template | Artifact | Bytes | SHA-256 |
| --- | --- | --- | --- |
| text-editor | `dev.velox.text-editor.zip` | 2,424,428 | `60009ca22538012b4f808448dbdbcd56db767194eb673bcd67c583b1fd73c7bc` |
| text-editor | `dev.velox.text-editor-setup.exe` | 6,524,076 | `3cd13789f9e9f97d65181cfee1e543126f0796a2f7638d85839d574bd084d849` |
| folder-browser | `dev.velox.folder-browser.zip` | 2,421,247 | `a68672e8db87ae8d303212d4fd8f8bb789db19b3f7a8b020753e2219daddbd2c` |
| folder-browser | `dev.velox.folder-browser-setup.exe` | 6,520,895 | `610cae1d75100e61bda9d37702851657b90036e66f222269eb0f4de1c44d8d0e` |

An initial text-editor `inspect` controller guessed a stripped-hyphen app
filename and failed; the run was corrected to the CLI `result.archive` path
(`inspect` then exited 0) with no product change.

The native packaged text-editor save used an isolated profile, not the
maintainer's editor or live drafts. The trusted two-line Korean/ASCII test
draft was stored, then three automated phases each exited 0: seed (draft
persisted), restore-save (restore left the document dirty, then Save wrote the
file and cleared the draft) and saved-relaunch (no resurrection). The
maintainer (user) operated the real OS save dialog, selected the specified
`recovered-draft.txt` destination and confirmed the save completed. Automated
disk readback verified exactly 44 UTF-8 bytes (SHA-256
`5c08265d657b0f47b1286af6017b9753b9264d9cabb57e3197a9efd731ea5855`), the
draft was cleared, and the relaunch showed no resurrection. Cleanup found no
candidate-owned process remaining and a profile rename succeeded.

The Windows UI helper could not inspect the app (a `get_window` mismatch, then
a foreground query returned no PID), so the save operation is human-supplied,
not a helper assertion. CDP was injected by the test environment on a
temporary loopback port, not by default app flags. No actual OS
`beforeunload` prompt test is claimed; only clean close with exit 0 is
asserted. This is a local acquisition test, not a public download or hosted
consumer attestation.

Reused checks: scoped Go tests/vet and 23 Bun cases for unchanged inputs; the
prior draft-smoke CLI/host hashes matched exactly; the prior diagnostics host
hash matched; earlier watch evidence was reused for unchanged implementation
and is not claimed as exact current CLI bytes.

Not performed in this slice: installer execution, folder native
selection/read re-check (template and host unchanged, prior manual evidence),
hosted stress, performance benchmark, signing (artifacts `NotSigned`) and
publication. Absence of these is not a pass.

Receipts: `dist/candidates/source-11efd69/candidate-result.json`,
`consumer-result.json`, `native-save-result.json` and
`cleanup-result.json`.

## Use And Boundary

Extract the whole candidate ZIP and keep the release files together. The
existing local-only text-editor IndexedDB draft database comes from the
earlier local commits and is not new here; the public alpha.65 payload and
existing generated projects are unchanged.

Do not confuse this source candidate with a publicly downloaded alpha.65: the
version string is the same but the bytes differ. Publishing this candidate
requires an explicit version decision and authorization; there is no new tag,
release, push, hosted run or beta promotion in this step.
