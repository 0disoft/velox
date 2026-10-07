# Alpha.68 Local Preparation: 2026-10-07

## Identity

- Source version: `0.5.10-alpha.68` (local candidate; not published).
- Public preview: `0.5.10-alpha.67` (published unsigned prerelease).
- This step permits local tests and a candidate build only.
- No push, tag, hosted job, signing or publication is included.
- Candidate root: `dist/candidates/alpha68-local/`.
- Artifact source: `f6c40818e01c59049fea68aa93de31e58798d7dc`, clean at build.
- Toolchain: Go 1.27.1 Windows amd64. Binaries were compiled once with
  `-buildvcs=false -trimpath -ldflags='-s -w'`; host and Setup also use
  `-H windowsgui`. All three are `NotSigned`.
- ZIP: `release/velox-windows-x64.zip`, 6,877,478 bytes, SHA-256
  `863af52303eecb8c52f75df439c413f9785c76c2dc3e953ffc067a0dc05ec90b`.

| Binary | Bytes | SHA-256 |
| --- | --- | --- |
| CLI | 5,083,648 | `1fa76db4dd71f73b19e801b7985c8772c1c082e2c97ce7e9db10f42801d275f7` |
| Host | 4,861,440 | `a7364ee43c8d95c1c59d0879b187b346eef2f18ae282eb514b92c3dfaac4dfa0` |
| Setup | 4,099,584 | `1609909ee9c040088f27e1f21304697127be9873ebde2305c4e119ea2c05c0ec` |

## Scope

- `templates` lists the four existing starters.
- Expanded CLI/init help and shell-quoted next-step run/build hints.
- Web-only text-editor literal find bar with match-case and wrap navigation.
- Empty-query Escape/X close fix, including IME handling.

Existing generated apps are unchanged. No host/IPC API, permission,
DB/storage schema, dependency or CI workflow change is included.
Product specification and command-contract records retain the distinction
between these source additions and published alpha.67.

## Local Verification

Buildplan, builder, CLI, inspector, runner, releasebundle, releaseevidence and
hygiene Go tests, scoped `go vet` and `git diff --check` passed for the version
change.

- Two packaging runs from the same binaries produced identical ZIPs. This
  does not claim two independent recompilations.
- The ZIP was extracted outside the checkout. Only its prebuilt CLI ran the
  consumer commands: version/templates and init/validate/build --installer/
  inspect for basic, text-editor, folder-browser and tray-app. All four builds
  reported alpha.68; every portable ZIP matched a second build. Generated
  hosts and type declarations matched bundled inputs. Installers were
  generated, not executed. The machine still had its development tools
  installed; no compiler or package manager was invoked by these consumers.
- All 16 manifest artifacts, three sidecar checksums, 17 SPDX file SHA-256
  entries and the single provenance statement's source/ZIP identity passed.
  These unsigned sidecars are not authenticated attestations.
- PE subsystem checks passed: host/Setup GUI 2, CLI console 3.
- Against the recorded alpha.67 local Go 1.27.1 candidate, CLI grew 21 KiB,
  host hash/size stayed identical, and Setup size stayed identical. This is
  not a startup or memory-performance measurement.

Receipt: `dist/candidates/alpha68-local/candidate-result.json`. Sidecars are
in `evidence/`; the local verification script is `.cache/alpha68-candidate.ps1`.
These generated files are ignored and are not committed.

## Verification Boundaries

Local Go 1.27.1 bytes are not a same-toolchain comparison with the public
alpha.67 Go 1.26.0 bundle. Public alpha.67 assets remain unchanged.

Historical manual find-close confirmation and earlier native tray, watch,
picker and draft receipts are retained, not relabeled as alpha.68 public-byte
evidence. Installer execution, new manual native interaction, hosted stress,
performance comparison and beta promotion are outside this local step.
