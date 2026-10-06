# Release

- Status: M4 unsigned distribution complete; M5 adoption evidence remains empty
- Owner: Project maintainer

## Current State

Velox distributes unsigned developer previews and has no package registry
entry, implemented signing workflow, or stable version policy. Maintainer tooling builds the Go CLI and
host, assembles the deterministic unsigned Windows x64 bundle, verifies
artifact entries against the release manifest, and emits checksums, a
file-level SPDX 2.3 SBOM, and one unsigned in-toto/SLSA provenance statement.
The alpha-evidence workflow builds the bundle twice and rejects differing ZIP
bytes.

ADR 0015 retains Velox as the maintainer-approved public identity. An existing
released Go CLI still distributes the exact `velox` command and `velox.exe`;
that collision is an accepted and disclosed release risk rather than a
replacement-name gate.

The repository also owns `velox.signing-record/v1` and a non-publishable
dry-run verifier. It binds unsigned inputs, the signing-input ZIP, signed-output
placeholders, the final manifest and ZIP, checksums, and SBOM without contacting
a provider or claiming Authenticode or artifact-attestation success.

The public repository now declares `MIT OR Apache-2.0`, identifies the
maintainer in CODEOWNERS, and includes security and privacy policies. The
SignPath application packet and exact proposed provider configuration live in
`docs/ops/signpath-onboarding.md` and `.signpath/`, but ADR 0011 defers provider
onboarding until a real adoption trigger exists.

A separate consumer job performs no source checkout and invokes no Go, Node,
Rust, C++, Bun, or package-manager command. It downloads the producer artifact,
verifies its checksum, initializes and validates a project, runs doctor, builds
twice, checks deterministic ZIP hashes, and inspects the result. Hosted runner
images can still contain preinstalled toolchains; the claim is that the
consumer job does not invoke them.

[Alpha evidence run 29714104653](https://github.com/0disoft/velox/actions/runs/29714104653)
completed the reproducible producer and checkout-free consumer jobs for tag
`v0.5.10-alpha.1` at commit
`9f10c545b6bde23d2c3dad5bbb12bffdac513712`. Manual publication
[run 29714173324](https://github.com/0disoft/velox/actions/runs/29714173324)
created the immutable prerelease. Public-download verification
[run 29715002921](https://github.com/0disoft/velox/actions/runs/29715002921)
downloaded the release without source checkout, matched ZIP SHA-256
`5df53090e1e67ce54c8639f061ffc7b03b7c3aa38f95a725c29342cfaff73b68`,
verified the sidecars, built twice, inspected, and reached startup-ready. This
is same-repository public-download evidence, not an independent external-user
attempt or authenticated attestation.

[Tag evidence run 29894943737](https://github.com/0disoft/velox/actions/runs/29894943737)
and [publication run 29895087658](https://github.com/0disoft/velox/actions/runs/29895087658)
produced the second preview `v0.5.10-alpha.2` from commit
`9bbb6bfcc1393058cb80d72c79df601caa970f2f`. Public-download verification
[run 29895490556](https://github.com/0disoft/velox/actions/runs/29895490556)
matched ZIP SHA-256
`abd07aab653db7d67adf822e6a944a6f85f54c9fb0752cce367724fb0ce62fb7`
and passed checksum, SPDX, provenance, deterministic-build, doctor, inspection,
and startup gates without checkout. It remains same-repository evidence with
`externalUserAttempt: false`.

The previous preview is
[`v0.5.10-alpha.40`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.40)
from commit `d206fe4ef1be9df198d86809742ef480549344b8`. Reproducible tag evidence
[run 34214224962](https://github.com/0disoft/velox/actions/runs/34214224962),
publication [run 34214445883](https://github.com/0disoft/velox/actions/runs/34214445883),
and public-download verification
[run 34215188131](https://github.com/0disoft/velox/actions/runs/34215188131)
passed. The publication artifact and public ZIP both have SHA-256
`771173b6eec2f74d92228e7ac5b52332160b0f9fb4d7baecf01976864a00f8c8`.
All four required assets are public. The candidate contains the callback
lifetime and initialization fixes recorded in the COM review. Beta remains
held under the product workflow checklist in `docs/ops/product-readiness.md`.

The now-archived public
[`0disoft/velox-consumer-smoke`](https://github.com/0disoft/velox-consumer-smoke)
repository consumed only the pinned public release. Hosted
[run 29736140250](https://github.com/0disoft/velox-consumer-smoke/actions/runs/29736140250)
passed the full public CLI path with no consumer toolchain command and zero
Actions cache upload bytes. ADR 0016 accepts this maintainer-controlled clean-
room result as the final technical M4 gate while preserving
`maintainerControlled: true` and `externalUserAttempt: false`.
It remains read-only as a one-shot receipt. Future release verification uses
the repository-owned public-preview workflow instead of advancing that pin.

## Alpha.67 Local Preparation: 2026-10-06

The source version is now `0.5.10-alpha.67`, a local candidate that is not
published. The scope groups manifest-watch notices, their JSON-mode stderr
path and the tray-app starter. Public alpha.66 remains unchanged. No tag,
push, hosted build or publication is included. Matching artifact results will
follow the version commit; see [alpha67-preparation.md](alpha67-preparation.md).

## Source Candidate `9dfcb8b`: 2026-10-06

The next preview's source scope is locally packaged: manifest-change notices
in development watch, stderr notices alongside JSON output and the tray-app
starter. The version remains `0.5.10-alpha.66`; these are local candidate
CLI/ZIP bytes, not the public alpha.66 release. The Go 1.27.1 CLI reuses the
unchanged public Go 1.26.0 host and Setup. Two packaging runs matched; an
outside-checkout prebuilt consumer generated, packaged and inspected all four
templates. Installers were generated, not executed. Sidecar checksums, SPDX
digests and unsigned provenance identity passed. No version/tag, push, hosted
job or publication occurred. Exact hashes, reused native evidence and skipped
checks are in [the source candidate record](source-candidate-9dfcb8b.md).

## Alpha.66 Published Preview: 2026-10-05

Unsigned prerelease `v0.5.10-alpha.66` ships the image/font `run --watch`
extension, opt-in metadata-only `run --debug` diagnostics and bounded local
IndexedDB draft recovery from source
`842fead0c889e9f161c2567a91c8d0fd4c2ca260`. Main and the annotated tag were
pushed together. [Tag CI 37317775399](https://github.com/0disoft/velox/actions/runs/37317775399)
(attempt 1) passed reproducible unsigned producer builds and a basic
checkout-free consumer smoke with Go 1.26.0 on `windows-2025`. Both consumer
app ZIP hashes were
`0ee5221c541312848255165c2dc8b0b1ef97bf157d4e17d4b33c7919f694fa33`.

`gh release create --prerelease --verify-tag --latest=false` reused that run's
four verified assets without a second producer, a tag move or a hosted public
verifier dispatch; release ID `403770595`, `draft: false`, `prerelease: true`,
published `2026-10-05T13:56:59Z`. Release notes reused the committed
`alpha-evidence.yml` publication template with the release channel,
compatibility floor and ZIP digest resolved. CommandCode DeepSeek 4.1
Flash/high drafted the documentation updates, checked against observed evidence.

| Public asset | Bytes | SHA-256 |
| --- | --- | --- |
| `velox-windows-x64.zip` | 5,903,850 | `3e1bc83cc8ee31e8feb4870a5b4fa26a4f992a5e22754d0625de891394cfacdc` |
| `checksums.sha256` | 279 | `f20db6a79ad78170d83360b4b067d1024ca20220eaaae40c76c15c381728dd72` |
| `velox-windows-x64.spdx.json` | 9,618 | `196d958fbd52c8c68a10d75a7da7d9c6896fbb15ac5dbdb8b2fd46a2f3c26938` |
| `velox-windows-x64.intoto.jsonl` | 744 | `951fabec910eadb00d92a94fed0fa358ea328a39868fe846aee558410d3fe1c2` |

All four public URLs were downloaded without authentication headers; their
bytes, sizes and hashes matched the CI assets and the GitHub metadata digests.
Three checksum entries, 16 manifest artifacts, 17 SPDX file digests, ZIP/CRC,
version/target and provenance source/run `37317775399/1` passed. The extracted
public CLI reported `0.5.10-alpha.66`. The executables are unsigned, and the
SPDX and provenance files are not authenticated attestations. This publication
verified artifact identity only: no native UI, startup, installer, `run --watch`,
draft recovery, multi-template packaging, hosted public verifier, performance,
stress or installer execution ran against these published bytes. The prior
implementation and local native evidence is retained but not upgraded to
alpha.66. No API, database, dependency or runner/job-scheduling change is
included. See [the alpha.66 publication record](alpha66-publication.md).

## Alpha.66 Tag CI: 2026-10-05

The owner approved a batched push of prepared source `842fead` and the
annotated tag `v0.5.10-alpha.66`. Remote main and the peeled tag both matched
`842fead0c889e9f161c2567a91c8d0fd4c2ca260`. The tag push triggered exactly one
alpha-evidence run, [37317775399](https://github.com/0disoft/velox/actions/runs/37317775399)
attempt 1, which passed the reproducible unsigned producer job and the
checkout-free consumer smoke with Go 1.26.0 on `windows-2025`. The publication
job was skipped in that run; publication followed separately afterward and is
recorded above, and at that CI step public alpha.65 was unchanged.
The release ZIP is 5,903,850 bytes, SHA-256
`3e1bc83cc8ee31e8feb4870a5b4fa26a4f992a5e22754d0625de891394cfacdc`; both
consumer app ZIPs hashed
`0ee5221c541312848255165c2dc8b0b1ef97bf157d4e17d4b33c7919f694fa33`. These
Go 1.26.0 CI bytes differ from the prior local Go 1.27.1 candidates and are not
upgraded to the earlier local native results. The receipt, exact hashes and
skipped native/public boundaries are in
[the alpha.66 tag evidence](alpha66-tag-evidence.md).

## Alpha.66 Local Preparation: 2026-10-05

This historical preparation preceded the tag CI above. At that point the
source version was `0.5.10-alpha.66`, not published, with no alpha.66 tag,
distribution bundle, digests or hosted evidence. The locally run
`go run ./cmd/velox version --json` reported the prepared version. Tag CI has
since produced the matching distribution artifacts; publication was still
pending at that step, and the public preview then remained
`v0.5.10-alpha.65`. Alpha.66 was published later from those verified assets;
see the published section above.

The prepared release scope is the feature slices implemented after the
alpha.65 publication: image and font asset detection in
`run --watch`, opt-in metadata-only `run --debug` JavaScript diagnostics, and
local IndexedDB draft recovery for newly generated `init --template text-editor`
projects. Existing generated projects are not upgraded. The
optional per-user Windows Setup from `build --installer` and optional
per-application EXE icon and version resource staging are already implemented
and are not new here. There is still no automatic updater, sealed assets,
Authenticode signing, arbitrary application backend or plugins, or non-Windows
target.

Version-dependent package tests, hygiene tests and scoped vet passed locally.
The publishing script parsed successfully and its notes contained no control
escapes; the workflow's triggers, jobs and commands are unchanged.
Preparation scope, validation boundaries, and the
historical source-candidate record are in
[the alpha.66 preparation doc](alpha66-preparation.md). The prior local
source candidate `11efd69` carries the same feature commits but uses the
`0.5.10-alpha.65` version string, so its bytes are not alpha.66 artifacts.

## Alpha.65 Published Preview: 2026-10-05

Unsigned prerelease `v0.5.10-alpha.65` ships the text-editor and folder-browser
`init --template` starters from source
`c8f618bd94e48cb7c01d61aa0e65e3bc7116875c`. Main and the annotated tag were
pushed together. [Tag CI 37298703202](https://github.com/0disoft/velox/actions/runs/37298703202)
passed reproducible unsigned producer builds and basic checkout-free consumer
packaging with Go 1.26.0 on `windows-2025`. Both consumer app ZIP hashes were
`29c9136005c8706d4cb237d4c2f3de824ef7230ebccb0cb7cff99400491bf26e`.

`gh release create` reused that run's four verified assets without a second
producer; release ID `403616613`, `draft: false`, `prerelease: true`,
`--latest=false`. CommandCode DeepSeek 4.1 Flash/high drafted release notes
and receipt paragraphs, checked against the source and observed evidence.

| Public asset | Bytes | SHA-256 |
| --- | --- | --- |
| `velox-windows-x64.zip` | 5,896,489 | `7f837fe69ff4ec9efcf528c97c63ea1d3dc010ba2a7d75d2eb824a1ff372c42b` |
| `checksums.sha256` | 279 | `c708ef8eb8793b600f7c9adc0a2dc057b87395c6919543957b7d90bd13ac960d` |
| `velox-windows-x64.spdx.json` | 9,618 | `3d141251d0a0f59e872044bdd3e55ccc9ab0e90032c1e13722459026a8b11658` |
| `velox-windows-x64.intoto.jsonl` | 744 | `7fc2f59913ec59ca09447848da8585ff0978e31862bbb4720c837df14fae8cb0` |

All four public URLs were downloaded without authentication headers; their
hashes matched the CI assets and GitHub digests. Three checksum entries,
16 manifest artifacts, 17 SPDX file digests, ZIP/CRC, version/target and
provenance source/run `37298703202/1` passed. Host/Setup PE headers are unsigned
Windows GUI (2). No publisher signature or authenticated attestation is claimed.

The extracted public CLI reported alpha.65. Each native template passed
`init`, exact permission/asset inventory, `validate`, `doctor`, portable plus
Setup packaging, and `inspect`. For the generated names `text-editor` and
`folder-browser`, web inventories were 8 files / 13,701 bytes and 7 files /
12,675 bytes; generated display names affect those sizes. Setup was not executed.
Receipt: `.cache/alpha65-public-template-check/result.json`.

The host remains 4,407,808 bytes and Setup 3,574,784 bytes, the same sizes as
alpha.64, but their hashes changed. Runtime source is unchanged except for the
release version. CLI size is 4,542,464 bytes, an increase of 47,104 bytes.
CLI SHA-256 `24424c8c199eff1435d1bc5e8447159df7d6adfb53682451d04ff2046d2dd8af`;
host SHA-256 `22779ec47413bb4582ccaf6ae23ad709694a283c138db8231475542398ca48b0`.

The exact public CLI/host passed native `run --watch` with debug off using
`scripts/dev-reload-smoke.ts`: two HTML/CSS/JS reload cycles, real
`beforeunload` cancellation preserving input, subsequent retry and normal
close/cleanup with exit 0. HTTPS origin and a private profile were retained,
with no test-side reload/cache override. Receipt:
`.cache/normal-reload-1791197916010/result.json`. Public files are in
`.cache/alpha65-public-assets/` and `.cache/alpha65-public-extracted/`.

Before publication, the maintainer confirmed a real Save as text save in
`My Editor` and Korean `sample.txt` preview in `My Browser`. Those local
source-generated apps used the unchanged alpha.64 host, not alpha.65 public
bytes. This is separate manual evidence, not external adoption or a hosted
native-template check. Prior alpha.64 records remain below. No installer
execution, hosted native template interactions, full stress/permission matrix,
performance comparison or beta promotion was performed. IPC/schema/DB/host
behavior/dependency/workflow contracts are unchanged.

## Alpha.64 Published Preview: 2026-10-05

The owner authorized publishing `v0.5.10-alpha.64` as an unsigned prerelease,
not beta. The remaining two commits and annotated tag were pushed together;
remote main and the peeled tag matched `10202571236801451fa697ade15f0e7a799a77ad`.
[Tag CI 37286184342](https://github.com/0disoft/velox/actions/runs/37286184342)
passed reproducible release builds and checkout-free consumer packaging with
Go 1.26.0 on `windows-2025`. The consumer's two app ZIP hashes matched
`c46d96df68dccb5c4becd8c34ccd2b7a3a021bda631b1199193ece11997f97f8`.

Verified artifacts from that run were published through `gh release create`
without a second producer or publication workflow. Release ID `403530744`,
`draft: false`, `prerelease: true`, and `--latest=false`. Release notes were
drafted by CommandCode DeepSeek 4.1 Flash/high and checked against the code
and exact CI receipts before publication.

| Public asset | Bytes | SHA-256 |
| --- | --- | --- |
| `velox-windows-x64.zip` | 5,876,352 | `010478c5eea256ae1892fec5c186f61ae133db0542327677cf14f28b21744c35` |
| `checksums.sha256` | 279 | `73d22226d01e5a437486c92123b5e5f41e78ff9671b1abfaadad2f274ad60319` |
| `velox-windows-x64.spdx.json` | 9,618 | `d476d707b0366198fb88ffb6ebc97743fc4c883923b4614931d936ee965cdca9` |
| `velox-windows-x64.intoto.jsonl` | 744 | `c61d9b67ce8868172fd5f45edb913757401df371faadc861772ab5c190425b74` |

All four exact public URLs were downloaded without authentication headers;
their hashes matched the verified CI assets and GitHub's reported digests.
Three checksum entries, 16 manifest artifacts, 17 SPDX file digests, ZIP/CRC,
version/target, provenance source and invocation `37286184342/1` passed.
The extracted public CLI reported alpha.64. Host/Setup PE headers are unsigned
Windows GUI (2); no publisher identity or authenticated attestation is claimed.

The exact downloaded binaries also passed native `run --watch` with debug off:
two HTML/CSS/JS auto-reload cycles retained the HTTPS origin, real
`beforeunload` cancellation preserved trusted test input, another edit retried
successfully, and normal close/cleanup returned 0. No test-side reload/cache
override or residual candidate browser process was observed.
CLI SHA-256 `4337138b571f9197b0986e78c16cd564f3ae24ce93e164624c7f2843e2ede0a3`;
host SHA-256 `ae5cd1bdd78b19743aebdfa80bb15e91e3f7b01b992e02aab6414e29eb0b151d`.
Receipt `.cache/normal-reload-1791190634658/result.json`; public files are under
`dist/releases/alpha64-37286184342/public/` and `public-extracted/`.

These Go 1.26.0 CI bytes differ from the Go 1.27.1 local candidate at `82d2485`.
The test used copied assets and a private profile, not an existing installed
app. No additional hosted stress, installer execution, complete native file
permission/recovery matrix, production performance comparison or manual
maintainer check was repeated. API/DB/CI/dependency contracts are unchanged
by this publication record; beta remains held.

## Alpha.64 Source Candidate: 2026-10-05

Historical preparation record; the subsequent publication is recorded above.

The owner approved one batched main push of four verified commits and local
alpha.64 candidate preparation. Remote main was confirmed at
`588c751279cde7b8ba1d12572665000607778e27`. Source version and current-version
test fixtures are aligned to `0.5.10-alpha.64`, including development watch.
At that preparation step, the public prerelease remained alpha.63. No alpha.64
tag, hosted release run, publication or beta promotion was included in that
authorization; the subsequent publication used a separate approval.

The clean source commit `82d248513c4fdac82ad3984a8bf0c90e209b7a77` now has
two byte-identical local candidate builds, sidecars and outside-checkout
consumer verification. Candidate ZIP SHA-256 is
`37b81f0aad32d7188b7f5e9ee11a6c99ea7bc73c0b66c9a60627b2e62dd1a2cf`.
Native source/packaged startup and exact-candidate development watch passed.
Paths, toolchain, checks and omissions are in [the candidate receipt](alpha64-candidate.md).

## Alpha.63 Published Preview: 2026-10-05

The previous unsigned prerelease was
[`v0.5.10-alpha.63`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.63),
source `fce9955bdb355fd1b1a377dec277a60727c4ad39`. Its annotated tag was pushed
and its peeled remote SHA matched this source. The owner explicitly approved
the unsigned alpha publication in the release task.

[Tag CI run 37224406021](https://github.com/0disoft/velox/actions/runs/37224406021)
passed reproducible producer builds and the checkout-free consumer job. It
used Go 1.26.0 on `windows-2025`. The two producer ZIP hashes matched; consumer
builds matched SHA-256
`dfc7d9787eb537221f412e281009980635030ead808f4a852b80222ecf16c762`.
The consumer invoked only `velox.exe`. Its 2,494.3955 ms acquisition/build
receipt is a single hosted observation, not a cold-build performance claim.

Publication reused this successful tag run's four assets with authenticated
`gh release create --prerelease --verify-tag --latest=false`, after local
payload verification. The workflow publication job was skipped on the tag
push; no second producer run was dispatched. No existing release was replaced.
Release ID `403152107` is public, non-draft and prerelease, with four assets:

| Asset | Bytes | SHA-256 |
| --- | --- | --- |
| `velox-windows-x64.zip` | 5,825,098 | `19205e691e79dcddaeeb414cbbeb4cb055e59344d85dabfd5f7bfe5ea99b27ca` |
| `checksums.sha256` | 279 | `379322658c7ab0f1f44305d4d5cbfa46cfa8682cf3ef6aadc35778771ab71e72` |
| `velox-windows-x64.spdx.json` | 9,618 | `b828f931e8f0daacb7d278a16d1ee5330be265f0dce9c7af8988a6fd2e27d3ae` |
| `velox-windows-x64.intoto.jsonl` | 744 | `fcbf5c5c017e82cae2bde5e9d3f3680eeae35da6b66c2a404aea7596f7443703` |

All four exact public release URLs were downloaded with `curl` without an
authentication header. Three checksum entries, all 16 release-manifest
artifacts, all 17 SPDX file digests, ZIP read/CRC integrity, version/target,
and provenance source/run `37224406021/1` matched. Public CLI `version --json`
returned `0.5.10-alpha.63`. The publication payload's three executables had
Authenticode `NotSigned`; host and setup PE subsystems were GUI (2).

Host SHA-256:
`9094c807a068e57bec3d055a4393a4c8ba70cdacf1e679309f75fda13b40df1d`;
CLI SHA-256:
`76d462fefdaf8a5eec6b63d1caede1104db961a69655f9d816c480d98945bff0`.
This Go 1.26.0 CI bundle differs from the earlier Go 1.27.1 local candidate;
their sizes are not a like-for-like performance comparison.

No public-preview-verification workflow, fresh native UI/lifecycle test,
Linux retest or full stress run was repeated in this publication step. The
previous alpha.62 public/native records and local candidate records remain
historical, not observations of these exact public bytes. Beta remains held.
Release notes disclose unsigned publisher identity, external mutable assets,
restart latency, the supported Windows floor, opt-in installer/branding, and
the absence of an updater or macOS/Linux application runtime.

## Alpha.63 Local Candidate Build: 2026-10-05

Source and remote `main` are `5c07245eb3794c88d54c61e65a69c8d35db06b12`; the
prior two commits were pushed and the remote SHA was verified. A fresh local
candidate was built from that commit with Go 1.27.1 (`-buildvcs=false`,
`-trimpath`, `-s -w`) for the CLI, the windowsgui host and setup, twice for
CLI, host and setup. The compiler cache was reused, so this is not a cold-build
performance claim.

Candidate ZIP `dist/candidates/alpha63-5c07245/velox-windows-x64.zip` is
6,782,242 bytes, SHA-256
`e637ff7eb36f665d7c929b6f799124b47cb91b78598bb7a69dd68ab9d6b62eec`.
Binaries: `velox.exe` 4,984,320, `velox-host.exe` 4,736,000, `velox-setup.exe`
4,099,584 bytes. The bundle carries 16 manifest artifacts including the three
`types/` files, with no `embed.go`. Sidecars are `checksums.sha256`, SPDX SBOM
and unsigned provenance; they do not authenticate the publisher. `TestBuiltHostStartup`
passed all five subcases in 45.079 s.

This candidate is local only: not tagged, published or signed. The current
public preview remains
[`v0.5.10-alpha.62`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.62)
with the URLs and SHA-256 records above unchanged, and beta remains held. The
detailed verification record is under Alpha.63 Local Candidate Build in
[Product Readiness](product-readiness.md#alpha63-local-candidate-build-2026-10-05).

## Alpha.63 Source Candidate: 2026-10-05

The working tree is aligned to source version `0.5.10-alpha.63`, replacing the
`0.5.10-beta.20` development version string. This is a source-only version and
channel alignment: the candidate has no release artifact, bundle ZIP, checksum
or SHA record yet, is not tagged, and is not published. The current public
preview remains
[`v0.5.10-alpha.62`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.62)
with the URLs and SHA-256 records above unchanged. Beta remains held under
`docs/ops/product-readiness.md`; this alignment does not promote beta and does
not claim any alpha.63 artifact bytes.

Cached `beta.20` CLI/host bundles and the prior verification records must not be
represented as `alpha.63` artifacts. Any published asset and release manifest
needs newly built, matching CLI/host binaries at the new version; no existing
beta.20 bundle is a substitute.

No API, database, CI, dependency or native-feature contract change is included.
Local verification is recorded under Version And Channel Alignment in
[Product Readiness](product-readiness.md#version-and-channel-alignment-2026-10-05).

This is the earlier source-only step; the built local candidate and its
checksums are recorded in the local candidate build section above.

## Alpha.49 Product Delivery: 2026-09-10

The previous preview is
[`v0.5.10-alpha.49`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.49)
from commit `b708cdd64fbc489729fb9ba519629fe72ee9242f`.
[Tag evidence 34453578289](https://github.com/0disoft/velox/actions/runs/34453578289)
and [publication 34453796476](https://github.com/0disoft/velox/actions/runs/34453796476)
passed reproducible builds and checkout-free consumption. Publication used
Go 1.26.8 and published four immutable unsigned assets. The release ZIP is
3,600,382 bytes with SHA-256
`236e71ce0fa19bae2b2bb56c44bd1daed1083931d275b64f8a11425d9d1c59fd`.

Changes include File Notes save-snapshot isolation and operation locking,
`velox run --debug` for source development, corrected Quickstart output paths,
and ADR 0019 product workflow gates. File Notes remains a repository example;
the binary release bundle does not newly embed those example sources.
IPC, database schemas and packaged security defaults are unchanged.

Public-download verification passed in
[run 34454305875](https://github.com/0disoft/velox/actions/runs/34454305875):
sidecar and producer-digest checks, version, init, validate, doctor, two
deterministic builds, inspect and startup all completed without source checkout.
Evidence artifact `10142809565` has 30-day retention. This remains
`same-repository-public-download` with `externalUserAttempt: false`.
The first [verification 34453983275](https://github.com/0disoft/velox/actions/runs/34453983275)
failed before execution because the expected digest came from the tag producer,
which used Go 1.26.7, rather than the publication producer, which used Go 1.26.8.
Both producers passed their own two-build reproducibility checks. The new
verification uses the publication producer's independently recorded digest;
no public asset or tag was replaced to hide the mismatch.

Local whole-Go and vet checks passed. File Notes model/application tests passed
11 cases; normal and debug native sample startup and deterministic builds passed.
The first final Windows startup smoke failed its 15-second readiness and
10-second browser-exit bounds. No residual test process was found afterward.
An unchanged rerun passed: first ready 724 ms, immediate ready 7.161 s, host exit
55/98 ms, browser exit approximately 6.4 s, profile release 6.446 s. The initial
intermittent failure remains unexplained; no timeout was relaxed.

This is an unsigned alpha, not beta promotion. Real native picker actions,
interactive reload, durable draft recovery after restart and broader runtime
coverage remain unverified under `docs/ops/product-readiness.md`.

## Alpha.51 File Consent Delivery: 2026-09-11

The historical preview is
[`v0.5.10-alpha.51`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.51)
from source `f18d7f3958c136b1f673b93255916771db3cde15`.
[Tag evidence 34584656621](https://github.com/0disoft/velox/actions/runs/34584656621)
and [publication 34584828937](https://github.com/0disoft/velox/actions/runs/34584828937)
passed independent two-build reproducibility and checkout-free consumption.
Four unsigned assets were published without replacing an existing release.
The ZIP is 3,601,844 bytes with SHA-256
`a2beb179266861be018fea9366eb6be201ede502515a97f0fcacb964ad0dc72a`.

[Public verification 34585168947](https://github.com/0disoft/velox/actions/runs/34585168947)
downloaded the published assets and passed checksum, sidecar, version, init,
validate, doctor, deterministic build, inspect and startup checks without a
source checkout. Its expected digest came from the successful publication
Actions artifact, whose provenance was checked against the tagged source.
The separate tag producer was not used as a substitute for those bytes.

The runtime now leaves trusted-origin FileReadWrite requests to browser
activation checks and consent even while serialized handles are restored.
It does not automatically grant permission or change the native IPC table.
The maintainer confirmed saving and restart recovery locally; native negative
permission and cancellation checks remain distinct. File Notes is still a
repository example, not a newly bundled application. There is no signing,
independent-adoption or beta claim. See `docs/ops/product-readiness.md` for
the remaining product checks and development reload results.

## Alpha.61 Desktop Delivery: 2026-09-16

The historical unsigned preview is
[`v0.5.10-alpha.61`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.61),
source `3fbe332e35cd262df47e3865e4b1f4e926c734c4`.
[Publication 35063918809](https://github.com/0disoft/velox/actions/runs/35063918809)
passed independent two-build reproducibility, checkout-free consumption, and
publication of the four verified unsigned assets. The ZIP is 3,777,283 bytes:
`c082c90cd15116617fd0a29b0f11cbc5a2080019bd6a5a05103f8fadbac8b4c5`.

[Public verification 35064135758](https://github.com/0disoft/velox/actions/runs/35064135758)
passed public downloads, checksums, SPDX/provenance, tag/version binding,
init, validate, doctor, deterministic build, inspect, and startup without
checking out source. Its expected digest was independently calculated from
the successful publication run's producer artifact, not from another build.

The release includes per-monitor DPI rendering, debug reload improvements,
the GUI-subsystem host, the embedded WebView2 loader, the default Velox icon,
clearer File Notes save errors, and native consent-based permission recovery.
The latter resets only a stored Deny for the current app origin to Default
after confirmation; it never grants Allow or retries saving automatically.

The maintainer confirmed Explorer-launched Save as, exit, relaunch, and Save
on the earlier CI-built alpha.61 example. This is separate from the later
publication bytes and from unperformed human Deny-reset confirmation/cancel
checks. See [file-permission recovery](file-permission-recovery.md) for exact
artifact identities and the preserved verification-helper timeout.
This release changes no public IPC or database contract. It remains unsigned,
does not establish independent adoption, and does not promote beta.

## Alpha.62 Early Close Delivery: 2026-09-16

The current unsigned preview is
[`v0.5.10-alpha.62`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.62),
source `02c9acb5035014d9e29a0eb5881a3cf5310f5d6d`.
[Consumer CI 35076577814](https://github.com/0disoft/velox/actions/runs/35076577814)
passed the new early-close regression, native startup/security, permission
recovery, quick lifecycle measurement, and consumer packaging checks.
[Publication 35079091056](https://github.com/0disoft/velox/actions/runs/35079091056)
passed independent two-build reproducibility and checkout-free consumption,
then published four verified unsigned assets. ZIP size: 3,778,397 bytes;
SHA-256: `10137ca603c5ba7f765d58f9e93fc78683f328aebad77659fd63e01367265871`.

[Public verification 35079337819](https://github.com/0disoft/velox/actions/runs/35079337819)
passed public download, checksum, SPDX/provenance, version, build and startup
checks. The expected digest came from this publication's producer artifact.
A separate local public-URL download matched it, and its actual EXE passed
all three early-close/relaunch/profile-release pairs plus icon, GUI subsystem,
ordinary lifecycle, missing-runtime and security tests in 43.64 seconds.
Host SHA-256: `651a9d87d16eee5687f4a1072226e3f9209a6ece438c0672e6c30e6680037679`.

Early user closure now exits 0 without a runtime-error diagnostic; genuine
initialization failure remains an error. The existing public alpha.61 binary
is not rewritten. The installed File Notes app and user profiles were not
replaced. Same-profile relaunch still took 6.97 seconds; extended hosted stress,
signing, and beta promotion remain separate. No JavaScript IPC or DB changes.

## Proposed Release Unit

During MVP, the CLI, generic host, JavaScript bridge, schemas, and
compatibility metadata release atomically as one versioned bundle.

Independent component releases are deferred until a real compatibility need
exists.

## Channels

Planned channels are alpha, beta, and stable. `0.5.10-alpha.1` remains the first
published preview and `0.5.10-alpha.66` is the current unsigned developer preview
at immutable tag `v0.5.10-alpha.66`. Public artifacts and executables use the
Velox identity fixed by ADR 0015. ADR 0019 defines the product workflow checks
required before beta technical readiness. AI evaluation is optional. Actual
beta or stable promotion, signing, and publication remain separate maintainer
decisions after those checks pass. The initial beta support scope is recorded
in the product spec and product-readiness checklist; it does not change this
alpha release or authorize beta publication.

Nightly distribution is not planned during the initial project stage.

## Required Release Contents

- Windows x64 Velox bundle.
- CLI and unchanged generic host.
- An optional `velox-setup.exe` template in the bundle and the per-application
  `<app-id>-setup.exe` produced by `velox build --installer`; both are
  unsigned, and the default portable output is unchanged.
- JavaScript bridge and schemas.
- Release manifest with contract versions and artifact digests.
- SHA-256 checksums.
- Software bill of materials.
- Third-party notices.
- Compatibility and known-limitation notes.
- Unsigned provenance metadata before the developer preview.
- Prominent unsigned, SmartScreen, and managed-device limitations.

The current local bundle includes the CLI, unchanged host, strict host
metadata, product and checkout-free-consumer JSON schemas, release manifest,
and third-party notices. The release builder uses an explicit schema allowlist
and fails when a required product schema is missing. Benchmark and other CI
evidence schemas remain maintainer contracts and are not copied into the
consumer archive.

The setup template is included only when `velox-release --setup` is passed, and
its release version, byte size, and SHA-256 are recorded in the release manifest
and re-verified before any installer is built. The Setup executable is an
optional additional artifact, not a replacement for the portable ZIP. Install
and removal behavior is in `docs/ops/windows-installer.md`.

Checksums, SPDX, and provenance are release assets, not contents of the
consumer ZIP. The provenance statement is deterministic metadata but is not a
signed attestation. An attacker who can replace both release and evidence can
still forge the complete unsigned set. ADR 0011 accepts that boundary for a
developer preview and requires it to be disclosed. ADR 0010 retains separate
authenticated provenance and Authenticode controls for a later signed channel.

## Release Gates

- The Velox identity and known command/search collisions are disclosed and
  accepted under ADR 0015.
- All configured correctness and Windows smoke checks pass.
- Unsigned reproducibility passes where applicable.
- Consumer build requires no compiler, Node.js, or Actions cache.
- Security baseline tests pass.
- Performance wording is regenerated from current benchmark evidence.
- Critical risks are mitigated, accepted explicitly, or stop the release.
- Directory asset tampering, branding, signing, and platform limitations are
  visible.
- Installer output, when produced, is opt-in, unsigned, and per-user, and adds
  no updater, elevation, or repair; changed or unowned files block removal.
- The preview is marked prerelease and prominently identifies both executables
  as unsigned.
- Publication requires explicit maintainer approval on an existing alpha tag
  and refuses to replace an existing release. The publication workflow uses
  exact-phrase confirmation; an approved manual publication can instead reuse
  a successful tag run's verified artifacts without rebuilding them.

## Compatibility Floor

The alpha contract supports Windows 10 version 1709 x64 and newer client
builds, or Windows Server 2016 x64 and newer server builds. Evergreen WebView2
Runtime `92.0.902.49` is the minimum because Velox requires
`ICoreWebView2_4` to cancel downloads as part of its security baseline. Doctor
checks both floors; ordinary Evergreen updates remain supported and are the
recommended runtime path.

The floor is derived from the
[Go Windows minimum](https://go.dev/wiki/MinimumRequirements), the
[WebView2 supported Windows list](https://learn.microsoft.com/en-us/microsoft-edge/webview2/),
and Microsoft's archived WebView2 SDK release notes that bind
`ICoreWebView2_4` SDK `1.0.902.49` to Runtime `92.0.902.49`.

## Signing Boundary

ADR 0011 owns the unsigned developer-preview boundary. No Authenticode provider
or signing credential is required for that channel. ADR 0010 and
`docs/ops/signing.md` own a future signed-channel boundary. SignPath Foundation
remains a conditional provider candidate; Microsoft Artifact Signing remains a
migration candidate where eligibility and publisher identity fit.

The provider signs the reproducibly built `velox.exe` and `velox-host.exe`.
The repository-owned `velox-signing-record prepare` command packages exactly
those two unsigned files into a deterministic, self-verified signing input
without contacting the provider.
The separate `authenticode` command then fails closed unless the returned
directory contains exactly those two names, both signatures are valid, both
use the approved exact publisher subject and SHA-256, both have timestamp
certificate identities, and both share one signer certificate.
The final bundle is then assembled from those exact signed inputs so
`velox-host.json` and `release-manifest.json` describe signed bytes. The generic
host remains byte-identical after release and during application packaging, so
its signature is preserved. Application-specific executable branding and
signing are not part of the initial release.

Signing credentials stay outside this repository. No private key or PFX enters
GitHub secret storage. Provider submission credentials, approval, and release
write permission belong to separate protected-environment gates.

## Developer-Preview Publication

ADR 0015 removes the replacement-name gate. This mechanism remains manual and
must not run until the candidate is rebuilt and every evidence gate below
passes.

Ordinary pull-request, tag, and evidence runs retain workflow artifacts and
have only `contents: read`. A manual dispatch can publish only when
`publish_preview` is true, the exact confirmation phrase is supplied, and the
selected ref is an existing `vX.Y.Z-alpha.N` tag. The isolated publication job
alone receives `contents: write`.

That job downloads the producer evidence after the checkout-free consumer job
passes, rejects missing or extra files, verifies every checksum, refuses an
existing release, and creates an immutable GitHub prerelease with the unsigned
warning. It also rejects a tag that is not exactly `v<releaseVersion>`. It does
not sign, attest, rebuild, or replace artifacts. The immutable release notes
also state the Windows and WebView2 compatibility floor, directory-asset
tampering boundary, unchanged-host branding limitations that the optional
per-user installer does not change, the optional unsigned Setup executable with
no updater or repair, and the accepted `velox` command and executable-name
collision.

Promotion to a future signed, beta, or stable channel reuses an already
verified immutable candidate. It does not relabel unsigned bytes as signed or
rebuild different bytes under the same version.

## Stop Conditions

- Reproducibility or checksum verification fails.
- Release artifact behavior differs from tested artifacts.
- Required WebView2 support cannot be stated accurately.
- Benchmark results fail the roadmap go-or-kill gate.
- Product or executable identity differs from ADR 0015.
- Security reporting and release ownership are not ready for public use.

## Post-Release Verification

For the first preview, a separate maintainer-controlled repository verified
download, checksum, version inspection, hello build, and application startup
from the public asset. Run `29736140250` completed that one-shot post-release
gate. The repository is now archived; this is not a recurring release design.

The repository-owned `Public preview verification` workflow covers the public
URL, independently supplied digest, checksum, SPDX, provenance, tag/version,
build, inspection, and startup boundaries without source checkout. Its result
is explicitly same-repository evidence. Neither it nor the clean-room consumer
repository substitutes for qualifying adoption evidence defined in
`docs/ops/external-user-attempt.md`.

For the first preview, the workflow initially exhausted its eight-minute job
limit because the ordinary generated starter did not emit the benchmark-only
ready marker. Commit `17a91f5c90dcbd58cf8aa20836994097e9c3262b`
made the verifier inject that marker only into its temporary fixture and bound
each downloaded CLI invocation to 120 seconds. The successful rerun is the
current post-release verification record.
