# Alpha.67 Local Preparation: 2026-10-06

## Identity

- Source version: `0.5.10-alpha.67` (local candidate; not published).
- Public preview: `0.5.10-alpha.66` (published unsigned prerelease).
- No alpha.67 tag, push, hosted run or publication is authorized in this step.
- Artifact source: `4d73cfbfd5e679283e7f44897ad4364544b0565b`, clean at build.
- All three executables were compiled once with Go 1.27.1 Windows amd64,
  `-buildvcs=false -trimpath -ldflags='-s -w'`; host and Setup also use
  `-H windowsgui`. All three are `NotSigned`.
- Candidate root: `dist/candidates/alpha67-local/`.
- ZIP: `release/velox-windows-x64.zip`, 6,868,107 bytes, SHA-256
  `15403133e1638a1621ab1d81e12b0643226b674010007b7ba538d8ae24b65723`.

| Binary | Bytes | SHA-256 |
| --- | --- | --- |
| CLI | 5,062,144 | `4b544a184de5b240d3ced5374124d0c36448fbfb084467caf3d3c1f828773b43` |
| Host | 4,861,440 | `a7364ee43c8d95c1c59d0879b187b346eef2f18ae282eb514b92c3dfaac4dfa0` |
| Setup | 4,099,584 | `5fd1683fa4cc3fe2a92e64c53c9bb59b71c2b855247c6cbd56ffd75461cfa154` |

## Scope

The next preview groups the CLI manifest-change notices for `run --watch`,
their stderr path alongside one JSON stdout envelope, and the new
`init --template tray-app` starter. Manifest edits require a restart; they
do not reload, restart or reconfigure the running host. Normal packaged
apps do not acquire a manifest watcher. The tray starter grants only
`notification.show` and reuses existing native behavior.

Implementation and manual evidence are retained in
[development-watch.md](development-watch.md) and
[tray-app-starter.md](tray-app-starter.md). The prior
[source candidate](source-candidate-9dfcb8b.md) retains its alpha.66 version
string and hashes; those artifacts are not alpha.67 bytes.

## Local Verification

Version-dependent buildplan, builder, CLI, inspector and runner tests passed,
along with releasebundle, releaseevidence and repository hygiene tests.
`git diff --check` passed.

- Two packaging runs from the same binaries produced identical ZIPs; this
  does not claim two independent recompilations.
- The ZIP was extracted outside the checkout. Its prebuilt CLI passed
  init/validate/build --installer/inspect for basic, text-editor, folder-browser
  and tray-app. All versions reported alpha.67, generated hosts matched the
  bundled host, and declaration files matched bundled types. Tray ZIP packaging
  repeated identically. Installers were generated, not executed.
- All 16 manifest artifacts, three sidecar checksums, 17 SPDX file SHA-256
  entries and the single provenance statement's source/ZIP identity passed.
  Unsigned sidecars are not authenticated attestations.
- PE subsystem checks passed: host/Setup GUI 2, CLI console 3. These local
  Go 1.27.1 sizes are not a same-toolchain comparison with public Go 1.26.0.
- Exact CLI/host native tray checks passed: visible Message input, accepted
  notification request, same document/text on second invocation, normal close
  and cleanup exit 0. A subsequent process check found no owned process.
- Mock light/dark UI checks at 620x480 and 360x540 passed, including rendered
  icon, fixed button, literal text retention, keyboard activation and no
  horizontal overflow. Native readiness screenshots were visually inspected.

Receipts are `dist/candidates/alpha67-local/candidate-result.json` and
`verification-result.json`; checksum/SPDX/provenance sidecars are in `evidence/`.

## Retained Fixture Failure

The first native test launched the GUI host with `windowsHide: true` and
timed out waiting for the Message input to be visible. Its process tree was
force-cleaned; receipt `.cache/tray-starter-1791292206598/result.json` remains
a failure. The same binary hashes passed visible `--manual-check` launch in
`.cache/tray-starter-1791292308385/result.json`. The smoke fixture now starts
the GUI host visibly even in automated mode; console helper launches remain
hidden. The default native smoke then passed in
`.cache/tray-starter-1791292356557/result.json` without an app-code change.
This identifies a fixture launch distinction, not the underlying cause of
the earlier maintainer blank-window report or a product root-cause fix.

## Boundaries

Existing public alpha.66 assets and historical receipts stay intact.
No API/IPC, DB/schema, dependency or CI workflow change is included.
No tag, push, hosted job, signing, installation, hosted stress, performance
comparison or publication occurred. Manifest-watch native tests were not
repeated against these exact alpha.67 bytes; unchanged implementation evidence
is reused. Earlier manual tray/balloon observations are not relabeled alpha.67
confirmation. Both requested DeepSeek/high documentation drafts ended without
final text at the token limit; this factual record was assembled from receipts.
