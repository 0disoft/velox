# Alpha.64 Local Candidate: 2026-10-05

## Identity

- Source: `82d248513c4fdac82ad3984a8bf0c90e209b7a77`, clean at build time.
- Version: `0.5.10-alpha.64`; public preview remains alpha.63.
- Toolchain: Go 1.27.1, Windows amd64, `-buildvcs=false`, `-trimpath`, `-s -w`;
  host and Setup also use `-H windowsgui`.
- Candidate root: `dist/candidates/alpha64-82d2485/`.
- ZIP: `velox-windows-x64.zip`, 6,838,865 bytes, SHA-256
  `37b81f0aad32d7188b7f5e9ee11a6c99ea7bc73c0b66c9a60627b2e62dd1a2cf`.

| File | Bytes | SHA-256 |
| --- | --- | --- |
| `velox.exe` | 4,984,832 | `7f6f7468d09cb6b3f09399c484a4cad24bc54cf236163007908d039bb62f89e8` |
| `velox-host.exe` | 4,853,760 | `6806eae9c2381766f6f9c7cda65a319cf022ffa19ef254ef04587f8adcdf8784` |
| `velox-setup.exe` | 4,099,584 | `bbe8a804a18b3e45b66b571bc0ea16893ff2b967653da2e2771316de8a0e2fff` |

All three Authenticode statuses were `NotSigned`. Two independent executable
and bundle builds matched byte hashes; the Go compiler cache was reused, so
this is not cold-build performance evidence. The candidate includes generated
checksums, file-level SPDX SBOM and unsigned provenance bound to the source.

## Verification

- Version-dependent build-plan, builder, CLI, inspector, runner, host metadata,
  release-bundle and hygiene tests passed. Unchanged watch/native unit tests
  and vet results from the watch implementation were reused.
- The local ZIP was extracted outside the source checkout. The consumer
  invoked only `velox.exe` for version, init, validate, doctor, two installer-
  enabled builds, inspect and run. PowerShell was the test controller, not an
  application build tool. The generated declaration matched the shipped file.
- Consumer ZIP and Setup hashes matched across both builds. The app ZIP was
  2,406,809 bytes, SHA-256
  `2f8fde519cbb4e314319a5d566db71c79bbdd89c12770a3b9067b759d6d19e34`.
  Setup was generated, not executed.
- Source run and extracted packaged app launch both reached `dom-2raf`, exited
  0, and their private browser processes exited. Temporary run config was
  removed. This is a local acquisition test, not a public download or hosted
  consumer attestation.
- The exact candidate CLI/host also passed `run --watch` without `--debug`:
  two successive HTML/CSS/JS edits auto-reloaded on the same HTTPS origin,
  real `beforeunload` cancellation preserved trusted test input, a subsequent
  edit retried successfully, and normal close returned 0. No test-side reload
  or cache override was used. No candidate-owned browser process remained.

Receipts: `candidate-result.json` and `consumer-result.json` in the candidate
root; watch receipt `.cache/normal-reload-1791189267262/result.json`.
The watch test used a copied File Notes fixture and a private profile, and the
consumer used disposable files. Existing installed apps and user data were
not changed. The candidate is not a new File Notes distribution package.

## Use And Boundary

Extract the whole candidate ZIP and keep the release files together. Its CLI
supports `velox run --watch --config path/to/velox.json`; normal run/build
defaults, permissions and application data formats are unchanged. Watch
performs full-page reload, not state-preserving HMR. Close an existing app
with the same identity/profile before a development run.

The four earlier verified commits were pushed together to remote main
`588c751279cde7b8ba1d12572665000607778e27`. Candidate source and its receipt
remain local. No tag, hosted Actions dispatch, release publication, installer
execution, broad stress repeat or beta promotion was performed. Signing and
production startup/idle-CPU benchmarking were not performed in this slice.
