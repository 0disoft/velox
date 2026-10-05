# Product Readiness

- Status: Alpha active; beta not approved
- Decision: ADR 0019
- Owner: Project maintainer
- Public preview: `0.5.10-alpha.66` (published unsigned prerelease)
- Source version: `0.5.10-alpha.66` (published unsigned prerelease)

## Required Beta Checks

| Check | Required evidence | Current boundary |
| --- | --- | --- |
| Public consumer path | Exact release URL and ZIP digest; source-free init, validate, doctor, build twice, inspect, launch | alpha.66 tag CI 37317775399 passed basic checkout-free packaging; unauthenticated public download bytes, sizes and hashes matched the CI and GitHub digests, and the extracted public CLI reported `0.5.10-alpha.66`. Packaged-app launch, native UI and OS interaction were not run against these published bytes; the native watch/UI evidence remains the historical alpha.65 result and is not upgraded to alpha.66. Prior alpha.62/63/64 receipts remain historical; see release.md |
| File Notes behavior | Real open, edit, save, save-as cancellation, permission denial, close/reopen and draft recovery using disposable files | Native cancellation/retry, denied-write protection, post-denial draft restoration and manual Save as with exact disk readback passed; same-file retry interaction was not separately observed |
| Development loop | Source run, edit/reload, default debug-off, preserved profile and app ID | Historical alpha.65 evidence: the exact public alpha.65 CLI/host passed two HTML/CSS/JS automatic reloads, canceled-input preservation, later retry and normal close using `scripts/dev-reload-smoke.ts` in watch mode without debug/cache override. Private profile and HTTPS origin were retained; receipt and hashes are in release.md. That native result is not upgraded to the alpha.66 published bytes. Historical alpha.62/64 reload and earlier failures remain below |
| Windows lifecycle | Bounded shutdown, immediate relaunch, initialization cancellation, no residual process/profile lock; bind runtime and source/artifact versions | Public alpha.62 passed three pre-ready close/relaunch/profile-release pairs and hosted 50-pair/100-launch stress run 35081507786. Multi-second relaunch delay remains; the stress run does not repeat initialization cancellation or cover every Windows/WebView2 version; see [lifecycle record](alpha61-lifecycle.md) |
| Security and data integrity | No unresolved critical issue; permission, origin, overwrite and recovery checks pass | Preserve existing security gates and unsigned-alpha warnings |

A visible window or readiness callback does not prove working file pickers,
correct rendered content, durable edits, or reload behavior. Use disposable
files and a private profile for validation; never overwrite a user's files.
Record failures and denied operations, not only successful paths. A failed or
unverified required check keeps beta held. Do not replace a real interaction
with a mock and label the gate complete.

## Alpha.66 Tag CI: 2026-10-05

The prepared source `842fead` and annotated tag `v0.5.10-alpha.66` were
pushed in sequence, and one alpha-evidence run
[37317775399](https://github.com/0disoft/velox/actions/runs/37317775399)
passed the reproducible producer and checkout-free consumer jobs with Go
1.26.0 on `windows-2025`. That CI run skipped publication; alpha.66 was
published afterward from those verified assets, recorded in
[the alpha.66 publication record](alpha66-publication.md). No native UI,
startup, installer, watch or draft interaction ran against these CI bytes,
and no public download or adoption result is claimed here. Tag receipts are in
[the alpha.66 tag evidence](alpha66-tag-evidence.md).

## Source-Only Local Candidate: 2026-10-05

Local source `11efd69a7dfeda04ec84e31b804b6af68cc29bad` was clean at build time and kept
the `0.5.10-alpha.65` string, but this source candidate's bytes differ from the
public alpha.65 prerelease. Build, both-template generation/packaging/
inspection, and a packaged text-editor native Save passed: a manual OS save to
the specified `recovered-draft.txt` with automated exact 44-byte readback,
draft clear, clean relaunch without resurrection and no residual candidate
process. Both extracted packaged EXEs byte-match the candidate host and the
generated project's root `velox.d.ts` matches the shipped types. Scoped Go/Bun
tests and prior host-bounded native evidence were reused rather than repeated
broadly. The public alpha.65 evidence above and the beta-held status are
unchanged; no push, release, tag, hosted run or publication is claimed. See
[the source candidate receipt](source-candidate-11efd69.md).

## Alpha.65 Publication: 2026-10-05

Unsigned prerelease `v0.5.10-alpha.65` ships both native starter templates
from source `c8f618b` after tag CI `37298703202` passed. The same four assets
were reused with no second producer; all public downloads matched the
recorded digests. Public-CLI generation, portable/Setup packaging and
inspection passed for both templates. Exact public-byte native watch
reload/cancel/retry/normal close passed separately.

The maintainer confirmed a real Save as text save and Korean folder-file
preview in source-generated `My Editor`/`My Browser` apps using the alpha.64
host before publication. That manual evidence does not cover alpha.65 public
native dialogs or external adoption. No installer execution, hosted native
template interaction, full stress/permission matrix or beta promotion is
claimed. Full hashes and receipts are in
[the release record](release.md#alpha65-published-preview-2026-10-05).

## Alpha.64 Publication: 2026-10-05

Unsigned prerelease `v0.5.10-alpha.64` was published from source `1020257`
after tag CI `37286184342` passed. The same run's artifacts were reused, with
no second producer. All four public downloads matched recorded hashes and
metadata; exact public-byte watch auto-reload/cancel/retry/normal close passed.
No beta promotion or broader stress/permission matrix is claimed. Full source,
toolchain, hash and skipped-check boundaries are in [the release record](release.md#alpha64-published-preview-2026-10-05).

## Source Development Watch: 2026-10-05

The local alpha.64 candidate at source `82d2485` passed two matching builds,
outside-checkout consumer packaging/startup, and exact-candidate native watch
reload/cancellation/retry/normal close. It remains unsigned and unpublished;
see [the alpha.64 candidate receipt](alpha64-candidate.md).

At that source-candidate step, the version was `0.5.10-alpha.64`; the public preview remained
`0.5.10-alpha.63`. The four completed package/installer/watch commits were
batched to remote main at `588c751279cde7b8ba1d12572665000607778e27` before
candidate preparation. Alpha.64 publication was subsequently authorized
separately, as recorded above.

Source-only `run --watch` adds debounced development reload, independent of
`--debug`, with no packaged/default watch loop. Detector, CLI/runner and UI
dispatch tests passed. A native private-profile run observed two consecutive
HTML/CSS/JS updates, real `beforeunload` cancellation preserving input, a
successful subsequent edit and normal close. The same-toolchain host grew
115 KiB (about 2.5%); no production performance claim is made. Published
alpha.63 artifacts are unchanged. Scope and evidence are in
[the watch record](development-watch.md).

## File Notes Public-Runtime Package: 2026-10-05

Follow-up: an isolated File Notes identity built with public alpha.63 passed
actual install, Start Menu shortcut launch, native close and removal. The
document and all 167 profile files (including five IndexedDB files) were
unchanged by removal. This does not execute the original File Notes Setup
bytes or establish actual draft restoration; see the package receipt below.

File Notes 0.4.0 was packaged locally with the verified public alpha.63 CLI,
including portable ZIP and opt-in Setup, in a separate output folder. The
old beta.20 output, application identity/version, 14 web assets and three
permissions were preserved. Packaged icons/font bytes and installer payload
identity passed inspection; an isolated actual-app readiness/shutdown smoke
passed with host exit 0 and browser exit. Setup was not executed, and no new
manual save/open/rendering evidence or beta promotion is claimed. Hashes,
paths and exact scope are recorded in [the package receipt](file-notes-alpha63.md).

## Alpha.63 Publication: 2026-10-05

The owner approved publication of `v0.5.10-alpha.63` at
`fce9955bdb355fd1b1a377dec277a60727c4ad39` as an unsigned prerelease, not beta.
Tag CI `37224406021` passed reproducible release builds and checkout-free
consumer builds. Its verified artifacts were published manually without a
second producer run. All four assets were downloaded from public URLs without
authentication and matched checksums, manifest and SBOM file digests and
provenance source/run; the public CLI reported alpha.63. ZIP SHA-256:
`19205e691e79dcddaeeb414cbbeb4cb055e59344d85dabfd5f7bfe5ea99b27ca`.
The payload is unsigned. Native startup/UI/lifecycle and full stress checks
were not repeated on these exact public bytes; earlier evidence is preserved
with its original source/toolchain boundary. See
[the publication record](release.md#alpha63-published-preview-2026-10-05).

## External Linux Common-Code Verification: 2026-10-05

An external Debian 13 amd64 / Go 1.26.7 receipt for source archive commit
`5daf7ae3aa58a9c1e19ae11e048083765961a141` was reviewed locally. The 690 source
hashes match the original archive; result checksums, recorded command exits
and detailed test counts also match. No source changes were made by the
external verifier. Root-module tests and vet exited 0, and actual consumer
summary CLI success/negative cases behaved as expected with synthetic inputs.

The detailed log contains 39 passed test packages, 323 passed top-level items,
two existing conditional skips and no failed items. Windows native features
and the separate WebView2 module's tests were not executed. This evidence
closes the common-code Linux execution gap for this source snapshot without
adding Linux application support or changing beta/release status. The external
receipt was inspected, not rerun or promoted to a hosted CI attestation.
Full scope, hashes and preserved logs are in
[the Linux receipt record](linux-common-go-20261005.md).

## Version And Channel Alignment: 2026-10-05

Source version is aligned to `0.5.10-alpha.63`, replacing the `0.5.10-beta.20`
development string. This is a source-only candidate: it is not tagged or
published, and no alpha.63 release artifact, bundle or checksum exists yet. The
status line stays "Alpha active; beta not approved"; the public preview then
remained `v0.5.10-alpha.62` and its recorded verification stayed valid.

Cached `beta.20` CLI/host bundles and the prior verification records are not
alpha.63 artifacts. Matching binaries must be rebuilt at the new version before
any release or publication. No API, database, CI, dependency or native-feature
contract change is included.

Build-plan, builder, CLI, inspector, runner, host-metadata and release-bundle
tests passed. Release-evidence tests initially failed because their source
fixture lacked the three newly required type files. The fixture was updated,
SBOM coverage for those files added, and the release-evidence tests passed.
Scoped vet passed. `go run ./cmd/velox version --json` reported
`0.5.10-alpha.63`; candidate/public-version hygiene checks and diff checks passed.
No matching release bundle, native UI test, benchmark, workflow dispatch, tag,
push or publication was performed in this source-alignment step.

## Alpha.63 Local Candidate Build: 2026-10-05

A local alpha.63 candidate was built from source and remote `main`
`5c07245eb3794c88d54c61e65a69c8d35db06b12`; the prior two commits were pushed
and the remote SHA was verified. The build used Go 1.27.1 with `-buildvcs=false`,
`-trimpath` and `-s -w`; host and setup also use `-H windowsgui`. CLI, host and
setup were each built twice. The compiler cache was reused, so no cold-build
performance claim is made.

Local machine-readable results are `dist/candidates/alpha63-5c07245/candidate-result.json`
and `consumer-result.json`. The generated bundle is not design authority and no
runtime contract changed.

- ZIP `dist/candidates/alpha63-5c07245/velox-windows-x64.zip`: 6,782,242 bytes,
  SHA-256 `e637ff7eb36f665d7c929b6f799124b47cb91b78598bb7a69dd68ab9d6b62eec`.
- `velox.exe` 4,984,320; `velox-host.exe` 4,736,000; `velox-setup.exe`
  4,099,584 bytes, read from the hashes JSON.
- Release manifest: 16 artifact items including the three `types/` files; no
  `embed.go`.
- Sidecars: `checksums.sha256`, SPDX SBOM and provenance. The evidence is
  unsigned and unauthenticated. Authenticode reports `NotSigned` for all
  three release executables.

The provenance generator retains its workflow-style GitHub Actions builder ID;
the invocation ID is local. This unsigned JSON is not a hosted-run receipt.

`TestBuiltHostStartup` passed all five subcases (early-close, icons, GUI,
lifecycle, security policy) in 45.079 s. First readiness 0.622 s, immediate
7.199 s; profile cleanup and browser exit were observed. The multi-second
relaunch delay remains and is not a performance advantage.

A local controller extracted the candidate ZIP outside the checkout and used
only the CLI consumer commands (version, init, validate, doctor, build twice
with `--installer`, inspect, run) plus the extracted app EXE. Source and packaged
launches both reached native ready phase `dom-2raf` and exited 0 with browser
exit. App ZIP 2,356,596 bytes, SHA-256
`5b481c3921362302cf184164536a7e82acaa62069309c868762b0f0561ea11ff`. Installer
`dev.velox.project-setup.exe` 6,456,244 bytes, SHA-256
`ebdaf919cfc5f1d92bcfbf9a229b1100828aa0bf04d5e6022d1b11b87346a360`, never
executed (no install or uninstall). The disposable fixture JavaScript check saw
the heading plus two animation frames, which is not visual-render proof and not
a manual save-UI check. The declaration matched the types, `permissions` was
empty, and the temporary run config was removed.

The first consumer controller attempt timed out after 60 s because it read the
readiness pipe only after the process exit; the controller was revised to read
readiness before waiting for exit, and the unchanged artifact then passed. The
failure and fix are preserved; no runtime fix was applied.

Not run: hosted CI, public download, or manual native functional tests. No tag,
release, publication, signing, new dependency, API, DB or CI workflow change is
included. Status stays "Alpha active; beta not approved" and the current public
preview remains
[`v0.5.10-alpha.62`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.62).
The generated Setup payload was opened and safely extracted into the disposable
consumer directory with the existing payload API. Its portable contents passed
inspection as alpha.63, its embedded archive was 2,356,596 bytes, and its
4,099,584-byte template matched the release template. The installer itself was
not executed. This documentation follow-up is local and unpublished.

## Quickstart Refresh And Batched Push: 2026-10-05

The seven verified commits after `35f6956` through `ee75d63` were pushed to
origin/main once. A separate remote-ref lookup confirmed
`ee75d63a0d178dd8337da5fbb8ed8d177da04f35`; no release was published or
additional workflow dispatched. This quickstart follow-up is a local commit.

The quickstart retains its immutable public-release download and seven-command
CLI path, removes obsolete Mustflow guidance, and adds gated current-source
permission, plain-JavaScript save, optional type and host-configuration examples.
Sections 7/8 do not claim those additions are present in published alpha.62.
The save example reports cancellation/failure, preserves text, disables duplicate
clicks, reuses only a document-scoped target and retries selection after errors.

The focused CLI sequence and public-byte hygiene checks passed. Strict checkJs
and a local sandbox passed missing/old bridge, cancelled save, successful save,
connected-target conflict, Save-as retry and duplicate-click suppression cases.
These are mocked interactions, not another native dialog test. Runtime code,
API behavior, DB, dependencies and CI configuration are unchanged; host rebuild,
performance tests and native UI verification were not repeated. The first seven
commits were pushed; this documentation follow-up is not yet remote.

## TypeScript Delivery: 2026-10-05

Source-built release archives now include the explicit `types/velox.d.ts`,
`types/README.md` and `types/example.ts` inventory with byte counts and SHA-256
digests. The CLI embeds only the declaration and init writes it at project root
alongside the four prior files, with a triple-slash reference in `web/app.js`.
Default permissions remain empty and the web asset boundary remains three
files. Init refuses a preexisting declaration before creating any new file.
Missing required release type files fail with staging cleanup. No dependency,
IPC method, DB, CI, version, push or public release change was added.

Initializer, release-bundle and CLI package tests passed; the release command
and embed package compiled. Focused missing-type cases and native-method
coverage passed, as did scoped vet. Strict no-emit TypeScript 5.9.3 and 6.0.3
checks included the save/error example; 6.0.3 checkJs passed on generated app.js
with its root declaration reference. A first checkJs invocation used an invalid
empty `--types` CLI argument; the corrected `--typeRoots` invocation passed.
The example was type-checked, not executed against native UI.

The rebuilt CLI was 4,984,320 bytes versus the same-toolchain cached CLI's
4,978,688 bytes (+5,632 / 5.5 KiB). The host dependency tree excludes the types
package; the unchanged 4,736,000-byte host retained SHA-256
`6097731876cc1e82be5e5fead8de05b96fefcc0d433f6a534fc7a3ed780e6600`.
No startup/performance benchmark or native UI test was repeated.

The local matching bundle built successfully (4,774,917 bytes, SHA-256
`8557d81b61423967986ef8635acba93683f39fca735d5dba1a9b9c8cf0718400`). All
three archived type files matched source and release-manifest hashes; the Go
embed source was absent from the ZIP. CLI init/validate/build/inspect passed,
and the generated declaration matched source SHA-256
`246cb4e21da585f70490f15a1f4e066ce33ad7e04bdbbfa2079207c1fe43a77c`.
The application ZIP contained no declarations, only the normal three web assets
and host/runtime/report files. This is local delivery evidence, not publication.

## TypeScript Bridge Declarations: 2026-10-05

The repository-local `types/velox.d.ts` describes all 26 IPC methods and the
three existing text-save helpers. It includes method-specific overloads,
response types, connected-save and clipboard cancellation unions, known error
codes, and an optional readonly `window.velox` for non-host browser previews.
Native permission, numeric/byte limits and document-token checks remain
authoritative. Definitions add no executable code and no runtime dependency.

Strict no-emit compilation passed with locally available TypeScript 5.9.3 and
6.0.3, including expected failures for unsupported names, missing/wrong/extra
parameters, inconsistent progress states, mismatched union-method arguments,
absent browser bridges and readonly assignments. The focused Go hygiene test
passed, comparing all 26 declaration keys against the native dispatcher's AST.
`git diff --check` passed. Runtime code is unchanged, so no host rebuild, native
UI test, performance benchmark or broad Go suite was repeated. The declaration
is not yet included in release bundles or init output, and no package-manager
publication, version, API behavior, DB, CI, push or release change was made.

## Maintainer Workflow Confirmations: 2026-10-04

When asked to verify actual Windows taskbar Normal percentages, Error,
Paused, Indeterminate and X clear in the isolated Taskbar Progress preview,
the maintainer replied "잘됨," generally confirming the requested interaction
worked. Individual percentages/colors were not separately reported. Actual
Explorer restart recovery remains unverified.

Earlier, the maintainer replied "잘됨" after being asked to check actual
Windows app light/dark switching for File Notes, generally confirming that
workflow. Native high contrast and a new file-save test were not reported.

When asked to press the tray bell, observe the actual notification, use tray
Hide window, then click the balloon to restore the window and close the test
window, the maintainer replied "잘됨. 다음 할일", generally confirming that
requested workflow. Per-kind icon/color, exact UTF-16 byte behavior, quiet
time and Explorer restart recovery were not separately reported. This is a
general workflow confirmation, not per-detail evidence.

When asked to type a disposable note, use tray Hide window, press
`Ctrl+Alt+Shift+V`, minimize and press the key again, close the window and
report any shortcut warning, the maintainer replied "같은 창과 글자가 잘 복원됨",
generally confirming that the same window and its text were restored. The
minimized-key restore, close-time unregistration and the conflict warning were
not separately reported, and the other manual conditions remain unverified.
This is a general workflow confirmation, not per-detail evidence.

These confirmations follow the automated-validation snapshots below.
Prior automated tests remain valid unchanged. No new build, tests, version
bump, API, DB, runner change or release is part of this documentation follow-up.

## Activation Shortcut Runtime: 2026-10-04

Optional `window.activationShortcut` (ADR 0039) adds one manifest/runtime-config string with no IPC permission, method or backend, no DB/profile change, no new dependency, timer, polling, worker, OS-setting change or version bump; nothing was pushed or published.

Executed `go test ./internal/activationkey ./internal/manifest ./internal/runtimeconfig ./internal/webview2 ./internal/builder ./cmd/velox-host ./tests/hygiene -count=1` and scoped `go vet`; all passed. Focused `-v TestActivationShortcut` runs after adding canceled-`WM_CLOSE` and reentrant-release cases passed without a skip, covering off/invalid input, rollback, and real Windows registration/collision/release with `Ctrl+Alt+Shift+9`, with no unowned `UnregisterHotKey`.

Injected `WM_HOTKEY` messages on real HWNDs verify id/modifier/key filtering, disabled-modal and closing suppression, hidden/minimized restore preserving maximized placement, canceled-close retention, exactly-once cleanup under reentry and no tray requirement. These prove native registration and injected-message handling, not physical-key delivery.

Matching Go 1.27.1 `-buildvcs=false -trimpath -ldflags '-s -w -H windowsgui'` builds measured 4,724,224 bytes before and 4,736,000 bytes after: +11,776 bytes / 11.5 KiB. New host SHA-256: `6097731876cc1e82be5e5fead8de05b96fefcc0d433f6a534fc7a3ed780e6600`. Size is not a startup-latency benchmark.

The matching CLI/host bundle built successfully. A private tray-less smoke app retained `Ctrl+Alt+Shift+9` in its packaged runtime configuration with no permissions; startup and exit-after-ready returned zero. The user-facing example is a separate change. Physical-key foreground activation and the real conflict-warning interaction remain unverified. Existing File Notes and the notification confirmation record are unchanged.

## Activation Shortcut Example: 2026-10-04

The isolated activation-shortcut 0.1.0 example sets
`window.activationShortcut: "Ctrl+Alt+Shift+V"` with `window.tray: true` and no
native permissions. Three Bun tests passed (0 failures, 15 assertions), and
headless Edge passed eight same-origin cases at 620 x 480 and 320 x 480 in
light/dark and both forced-color modes: the exact `Note` textbox label, local
Unicode UTF-16 counting, focus and live-theme note preservation, the 2048
`maxlength` blocking an extra input, and no IPC, page errors or horizontal
clipping. Desktop-light and mobile-dark screenshots are readable; an actual
native `Ctrl+Alt+Shift+V` key press was later exercised in the requested manual
workflow and generally confirmed, under Maintainer Workflow Confirmations:
2026-10-04.

The matching cached new beta.20 runtime (commit `1542c8a`) and CLI built the ZIP
without a runtime Go test, vet or rebuild rerun. ZIP inspection passed with no
permissions and runtime `activationShortcut: Ctrl+Alt+Shift+V`, `tray: true`:
2,357,597 bytes, SHA-256
`26a21442030dd98f3093a7dd5a0b8f2d3b461d433d0a722ea6b7a49dc8fd7517` (host
`6097731876cc1e82be5e5fead8de05b96fefcc0d433f6a534fc7a3ed780e6600`). Packaged
startup and exit-after-ready returned zero with a private profile, and a visible
preview was opened for the user. The same-window and text restore was generally
confirmed for the type, tray-hide and `Ctrl+Alt+Shift+V` workflow; the
minimized-key restore and close-time unregistration were not separately reported,
and the conflict warning and other manual conditions remain unverified and are
not release blockers. No permission, IPC method, DB, dependency, timer, storage,
network, CI, version, push, public release, File Notes or Activation Shortcut
Runtime record change was added.

## Tray Notification Runtime: 2026-10-04

`notification.show` independently permits transient balloons and requires an
explicitly enabled `window.tray`. IPC, manifest, runtime-config, WebView2, builder and
host tests passed, as did the repository hygiene suite; build-report has no
standalone tests and is covered by builder checks. Scoped `go vet` passed.
A second native pass adding a missing-tray case and title/short-message cases
kept the ipc and WebView2 suites green.

Tests inject the `Shell_NotifyIcon` call and use disposable real HWNDs. They
cover the strict 255-UTF-16 message limit and two-field validation under an
independent permission, `NIM_MODIFY` with `NIF_INFO | NIF_REALTIME` and
termination, the fixed manifest app name even when the caption changes,
missing-tray failure without auto-installing an icon, unavailable/modal/
shutdown rejection, shell failure, click restoring a hidden or minimized
window, and cleanup with no replay after `TaskbarCreated`. They do not submit
a real shell balloon, so actual visibility, click handling, quiet time and
disabled-notification settings remain manual unverified.

Matched Go 1.27.1 `-buildvcs=false -trimpath -ldflags '-s -w -H windowsgui'`
host builds compare 4,717,568 bytes before and 4,724,224 bytes after:
+6,656 bytes / 6.5 KiB. Final host SHA-256
`8005b4dc55a12c2c81ee3558a19004e3411208d9b1ea39f6f6b6cded12e9d171`. The
matching beta.20 CLI/host bundle was built and a private smoke app with
`tray: true` and `notification.show` packaged; runtime permission readback
matched, and a startup plus exit-after-ready run returned zero with a packaged
invalid message rejected as `INVALID_PARAMS`. The first smoke timed out and
was killed because its test HTML lacked the explicit `__veloxReady` signal;
only the test page was fixed before it passed. This is size evidence, not a
startup-latency benchmark. No dependency, timer, worker, DB or profile-format,
file-permission, CI/runner, version or File Notes change was made, and nothing
was pushed or published.

Validation: `go test ./internal/ipc ./internal/manifest ./internal/runtimeconfig ./internal/buildreport ./internal/webview2 ./internal/builder ./cmd/velox-host ./tests/hygiene -count=1`, then scoped `go vet`.

## Tray Notification Example: 2026-10-04

The isolated tray-notification 0.1.0 example requests only `notification.show`
with `window.tray: true`. Five Bun tests passed (0 failures, 66 assertions):
submit-only native calls, kinds and the 255-UTF-16 boundary, local input
counting, duplicate blocking, input preservation with focus return, redacted
errors and missing bridge. Headless Edge with a controlled mock bridge passed
eight same-origin cases at 620 x 480 and 320 x 480 in light/dark and both
forced-color modes: keyboard submit, exact `Kind`/`Message` names (corrected
from an initial nested-label timeout), the Lucide bell as a current-color mask,
no clipping, no page errors, and no IPC on live theme switching; inspected
desktop-light and mobile-dark screenshots are readable. This is mock-bridge UI
evidence, not native shell balloons.

The matching beta.20 CLI/host bundle was reused unchanged (host SHA-256
`8005b4dc55a12c2c81ee3558a19004e3411208d9b1ea39f6f6b6cded12e9d171`); no runtime
Go test, vet or rebuild reran. ZIP inspection passed with only
`notification.show` granted: 2,354,764 bytes, SHA-256
`6de96c0a6d7863b96a15849a1f5f4eb006a34f388d3ff9d5dd92667a3ed19b3e`. Packaged
startup and exit-after-ready returned zero with a private profile. Actual
balloon display and click-to-restore now carry the general maintainer workflow
confirmation under Maintainer Workflow Confirmations: 2026-10-04. Per-kind
icon/color, suppression settings, quiet time and exact UTF-16 byte behavior
remain unverified, outside that requested manual workflow. The preview used a
private profile. The first preview
helper started the GUI hidden; it was stopped and relaunched with normal GUI
options. No dependency, timer,
worker, network, storage or file permission was added, and no File Notes,
runtime API, DB, CI, version, push or release change occurred.

## Taskbar Progress Example: 2026-10-04

The isolated taskbar-progress example requests only window.progress. Its
three Bun interaction tests passed, covering all state payloads, local-only
slider input until change, clear, pending-call suppression, failure unlock
and missing bridge. Headless Edge with a controlled mock bridge passed eight
same-origin HTTP cases at 620 x 480 and 320 x 480 in light/dark and both
forced-color modes. Screenshots show readable controls and the existing
Lucide clear icon with no horizontal clipping or page errors. Live CSS theme
switching preserves selected state and percentage without another IPC call.
This is browser UI evidence, not native taskbar presentation.

The beta.20 CLI build and ZIP inspection passed, including runtime-config
permission readback. ZIP: 2,348,032 bytes, SHA-256
`7333500a2c63b98afcc1e7e28f935c9f482d902aaf0e6a7f104e543d13eb9276`.
Packaged host SHA-256:
`86e8e5c5fcb9bcd5598b432119ecf8ce90032c71e2cb71034c7a4bfd8fc809ea`.
Packaged startup and exit-after-ready returned zero with a disposable profile.
Visible Windows taskbar state/color/percentage/clear and actual Explorer
restart recovery remain manual observations. File Notes is untouched. No
network, file permission, draft storage, frontend package or timer is added.
No version change, push, workflow dispatch or publication occurred.

## Taskbar Progress Runtime: 2026-10-04

Independent window.progress adds window.setProgress under ADR 0037. IPC,
manifest, runtime-config, webview2, builder, host and hygiene tests passed;
buildreport has no tests. The vendored go-webview2 root tests and scoped
go vet passed. Native tests received a real Explorer TaskbarButtonCreated
message and exercised all five states through ITaskbarList3 on a disposable
window. This proves native call success, not visible taskbar rendering.
Controlled clients cover pre-ready caching without COM calls, lazy creation,
value-before-state, deduplication, creation/application failures and explicit
retry, button recreation, queued reentrant reset, destruction and exactly-once
release during an active native call. BeforeShow tests verify hidden-window
installation and failed-setup window/context cleanup. Explorer itself was not
restarted, and no latency or memory benchmark was run.

Matched stripped windowsgui/trimpath/buildvcs=false host builds grew from
4,696,576 to 4,717,568 bytes: +20,992 bytes / 20.5 KiB, about 0.45%.
No new dependency, timer, worker, DB/state format, file permission or runner
change is included. File Notes is unchanged; runtime remains beta.20.
This is local validation only, with no push or publication.

## File Notes System Theme: 2026-10-04

File Notes opts in with window.followSystemTheme true; CSS provides light and
dark palettes plus forced-colors, and the four icon buttons reuse the existing
Lucide SVGs as masks. Thirty-three Bun app/model/find tests passed. Same-origin
local HTTP headless Edge checks at 1080 x 760 and 390 x 844 passed eight CSS
cases (light/dark x none/active forced-colors), live light/dark changes kept
the buffer, and local find with discard cancellation worked with no page
errors or horizontal clipping. Screenshot inspection caught blank masked
icons in the initial file:// harness; the same-origin rerun visibly shows all
four icons.

CLI validate, build and ZIP inspect passed on beta.20; runtime-config readback
kept followSystemTheme true, and a disposable-profile packaged start exited
after ready with code 0. The archive is 8,458,736 bytes, SHA-256
`eaa25ecfaf67599324a213f807a9c3ad8770051f969dcde1e90c3afed8c32cf6`.
The shared host is unchanged at
`8577ceb149427e25ef68e84ca5f150c874ca950da28a4c0a8b92c7727f53dac0`;
the branded packaged host SHA-256 is
`2d3062de52b74e6593bdd0a1b2df08122fc1b59e5f369eae2a50f5018f9eafdf`.
Assets add 3,197 bytes over the preceding File Notes sources. No save,
recovery, JS, API, DB, permission, runner, dependency or version change is
included, and no push or publication occurred. The native title bar runtime
checks from ADR 0036 are reused; browser emulation does not verify current
live OS theme or high contrast, and real file saving was not repeated because
the JS and save logic are unchanged.

## Always-On-Top Native Test Follow-up: 2026-10-04

The earlier failure was reproduced at the first default-off menu toggle,
not at initial installation. An initial isolated ten-repeat run passed,
but subsequent isolated and full scoped runs failed on the never-shown
STATIC window. Direct topmost changes on that hidden fixture also failed
readback, despite SetWindowPos returning success. This is not evidence that
all hidden windows behave the same way, or a reproduction in a Velox app.

The system-menu test now shows its disposable window with SW_SHOWNA before
installation, without activating it. The final ten-repeat run passed, retaining
topmost/menu readback, geometry, visibility, no-activation, cleanup and
restart-default assertions. A separate hidden-window startup test covers
default-off and default-on installation without showing or activating either
window. The six-package scoped command below now passes with count=1;
go vet for internal/webview2 and git diff --check also pass.
No production runtime or dependency change is included. Hidden-window menu
toggle readback is no longer used as a proxy for a displayed system menu;
the historical failure remains recorded below.

## System Theme Example: 2026-10-04

The isolated scratchpad opts into window.followSystemTheme with no native
permissions. Two final builds produced the same 2,334,142-byte ZIP, SHA-256
`1d0492ab4ddf819c0bcbfe4c0d5e2d79aae032ae00a1aebc6d28a013fdd761b6`.
ZIP inspection passed; packaged host SHA-256 is
`8577ceb149427e25ef68e84ca5f150c874ca950da28a4c0a8b92c7727f53dac0`.
Packaged startup with private profiles and exit-after-ready returned zero
across two probes; the final startup after the CSS-only palette change passed.

Edge emulated CSS checks at 480 x 360 and 320 x 200 covered light and dark,
plus one 320 x 200 forced-colors active pass. They confirmed text input,
in-bounds textbox, no page errors, live prefers-color-scheme switching without
reload and Canvas/CanvasText forced-color mapping; screenshots were inspected.
These browser checks do not prove live Windows theme switching or the native
title bar. The example's manual live Windows theme switch remains pending.
File Notes and its data are untouched. No dependency, IPC permission, DB,
runner, workflow or version change is included; runtime version stays beta.20.

The maintainer reported "잘됨" after using the example. This is general
confirmation; individual theme-switch and high-contrast observations were
not separately reported.

## System Theme Runtime: 2026-10-04

Optional window.followSystemTheme defaults false. The core change is committed
at de7ff35. Theme-specific `go test ./internal/webview2 -run Theme -v -count=1`
passed all four tests, including a native DWM attribute-20 readback that checks
the applied ABI value. Manifest, runtime-config, builder, host and hygiene
packages passed, and `go vet` passed for the changed packages. Theme tests
cover default/unsupported no-call behavior, mocked light, dark, high-contrast
and deduplicated paths, synchronous re-entry, retained-state retry after
read/apply errors and owner cleanup. The registry is queried read-only; no
global setting is mutated. The Windows 11 build 22000 / attribute 20 floor is
documented, and unsupported builds keep the default title bar.

The full scoped command `go test ./internal/manifest ./internal/runtimeconfig
./internal/webview2 ./internal/builder ./cmd/velox-host ./tests/hygiene`
failed only on the existing `TestAlwaysOnTopNativeToggleAndCleanup` at
`always_on_top_windows_test.go:45` (`topmost != true` on a hidden
disposable STATIC HWND). No topmost code changed; that failure's cause and its
relationship to this change were not isolated. This is not a full-suite pass,
and the failure is not classified as known-flaky here.

Matched Go 1.27.1 stripped windowsgui/trimpath/buildvcs=false builds increased
the host from 4,682,240 to 4,696,576 bytes: +14,336 bytes / 14 KiB, about 0.31%.
This is artifact-size evidence, not a latency or memory benchmark. No new IPC
method or permission, dependency, timer, worker, DB, runner, workflow or
state-format change is added. Version remains beta.20; no push or release
publication was performed.

## Fixed Window Example: 2026-10-04

The isolated scratchpad uses resizable false, rememberState true and no native
permissions. Packaged runtime-config readback preserved both flags. Two builds
produced the same 2,329,784-byte ZIP, SHA-256
`75ef85623a920427377c9c7e092b560fcc61f38a0c0b818913d39c451d20df4e`.
Inspection passed; packaged host SHA-256 is
`937181aeeedb9631e51b95ddb858906a38df18330fecc8c6a2a3a9f9cf0b3d08`.
Packaged startup exited-after-ready with zero status using a private profile.
Edge typing/layout checks at 464 x 320 and 320 x 200 passed textbox bounds,
no page errors and no horizontal clipping; screenshots were inspected.
These browser checks do not verify native resize/maximize restrictions.
The native runtime checks are recorded below; physical drag/double-click,
minimize/restore, keyboard snap and cross-monitor DPI remain manual checks.
No existing user profile, installer or public release is replaced.

The maintainer reported "잘됨" for the Fixed Window example on 2026-10-04.
Record this as general confirmation that the example worked; do not infer
separate drag, snap or DPI observations.

## Fixed Window Runtime: 2026-10-04

Optional window.resizable defaults true; omission remains omitted in packaged
config. False removes sizing/maximize frame bits, blocks SC_SIZE/SC_MAXIMIZE
and rejects the existing window.maximize IPC operation with the existing native
failure result. Minimize/move/restore/close and DPI suggested-rectangle handling
are retained. Saved-state restoration for fixed apps uses current configured
outer dimensions at target DPI and saved position, ignoring saved maximization.
Resizable apps retain full-placement behavior. No state-format migration occurs.

Scoped manifest/runtime-config/WebView2/builder/host/hygiene tests and go vet
passed. Three-repeat fixed-window tests passed. Native checks cover default
style preservation, fixed style bits, system commands, maximize rejection,
unchanged geometry/visibility/focus, invalid handles, destruction and saved
position/current size with saved maximization ignored. Pure fitting checks
cover 96/120/144 DPI, negative monitor origins and small work-area bounds.
Actual frame dragging, keyboard snapping and physical DPI transitions remain
manual checks; no broader platform compatibility claim is added.

Matched Go 1.27.1 stripped windowsgui/trimpath/buildvcs=false builds increased
the host from 4,677,632 to 4,682,240 bytes: +4,608 bytes / 4.5 KiB, about 0.10%.
This measures artifact size, not startup latency or memory. No new IPC method
or permission, dependency, timer, worker, DB, runner or workflow is added.
Version remains beta.20; installer/public release publication is not included.

## Always On Top Example: 2026-10-04

The isolated scratchpad opts into window.alwaysOnTop with no native permissions.
Two final builds produced the same 2,326,201-byte ZIP, SHA-256
`738a7acd91c347cd9a339d9085c91ee990cff3d9a7db0e4acf6c42d1637cd447`.
ZIP inspection and runtime-config readback passed; unchanged packaged host
SHA-256 is `c94a6bf8705550ee0b36acd870ab37aada7dab80ce90c6aa7b831513e65d63e2`.
Packaged startup with a private profile and exit-after-ready completed with
status zero. The first startup probe timed out because the static example
omitted the benchmark readiness marker; adding the existing two-animation-frame
marker fixed the probe. Preserve this failure as test-fixture evidence.

Edge checks at 480 x 360 and 320 x 200 passed text input, textbox bounds,
no page errors and no horizontal clipping; screenshots were inspected.
These browser checks do not prove native z-order or menu interaction.
The maintainer reported "잘됨" after the manual test window on 2026-10-04.
Record that the example worked without inventing separate observations for
every menu, overlap, minimize/restore or owned-dialog step.
The scratchpad does not save its text; existing File Notes data is untouched.
No installer, public release, version bump or remote push is included.

## Always On Top Runtime: 2026-10-04

Optional window.alwaysOnTop defaults false and propagates through the manifest,
runtime config and host. Every app receives a checkable system-menu toggle.
It reads the actual WS_EX_TOPMOST state, does not activate/show/resize a window,
and does not persist the user's toggle. Restart reuses the manifest value.
File Notes does not opt in. No new IPC method/permission, dependency, timer,
worker, DB/state-format, runner or workflow change is included.

Scoped manifest/runtime-config/WebView2/builder/host/hygiene tests and go vet
passed. Disposable hidden HWND checks cover default-off-first and default-on
installation, command/checkmark round-trip, reserved low command bits, native
state refresh, unchanged geometry/visibility/focus, invalid handles, destruction
and fresh-window defaults. Early tests failed on the first lazy SetWindowPos
lookup from a native callback; pre-resolving it before installation passed two
ten-repeat native runs and the scoped suite. An earlier non-inlining hypothesis
was disproved and that workaround was removed.

Matched Go 1.27.1 stripped windowsgui/trimpath/buildvcs=false builds increased
the host from 4,671,488 to 4,677,632 bytes: +6,144 bytes / 6 KiB, about 0.13%.
This is size evidence, not a startup-latency or memory benchmark. System-menu
interaction, visible overlap and owned dialogs remain manual checks. Version
remains beta.20; no push or release publication was performed for this change.

## Minimum Window Size Examples: 2026-10-04

File Notes requests an outer minimum of 720 x 520 logical units; the isolated
attention example requests 360 x 360. Packaged runtime configs preserved both
pairs without changing permissions. Two File Notes builds produced the same
8,447,049-byte ZIP, SHA-256
`1519a3bd0738f20c1f70853eb4776d35faedc59ba44540b346cd2e01ca3ca187`.
ZIP inspection passed; packaged branded host SHA-256 is
`581ca0999074c673b2ed6e307f25207cc5f423b0752e9f2d7d2eb61c3eb04bb1`.
Attention's 2,326,784-byte ZIP SHA-256 is
`6afab3841a974ad266e2185d6b8af41799c76959953c7b853f7e850f5c964487`.

Both packaged apps started and exited-after-ready with zero status using
separate private profiles. Thirty-nine File Notes/attention Bun tests passed.
Mock-native Edge checks at File Notes 704 x 480 / 344 x 320 and attention
344 x 320 / 680 x 400 passed controls, search/request interactions, font/icon
loading and no horizontal clipping. Screenshots were inspected. Narrow File
Notes layouts intentionally use vertical scrolling; smaller work areas may
cap the requested minimum below its configured value. These browser checks
do not prove physical monitor/DPI dragging or user-driven native resizing.

The maintainer reported "잘됨" for the prior attention test executable on
2026-10-04. Record this as confirmation that the example worked, without
inventing individual flash counts or per-step cancellation observations.
No repeat human attention test is required for this minimum-size-only change.
Actual packaged visual shrinking and physical monitor changes remain manual.
Existing File Notes/recovery data and versions are preserved; no installer,
release, DB, dependency, runner or workflow change is included.

## Minimum Window Size Runtime: 2026-10-04

Optional window.minWidth/minHeight are validated and propagated through the
manifest, runtime config and host. Zero axes retain normal Windows tracking;
both zero install no handler. Minimums are outer 96-DPI logical dimensions,
bounded by initial dimensions and 16384. Current DPI scaling and monitor work
area caps apply after saved-state restoration, on normal resize/restore and
to suggested normal DPI-change rectangles. Maximum geometry remains OS-owned.

Scoped windowlimits/manifest/runtimeconfig/WebView2/builder/host tests, repository
hygiene checks and go vet passed. Pure checks cover 96/120/144 DPI, one-axis
limits, negative-coordinate and small work areas. Disposable hidden HWND tests
cover native tracking, normal programmatic resize, suggested DPI rectangles,
unchanged unset/max geometry, no visibility/activation and cleanup.
An existing attention test's comparison against arbitrary foreground desktop
state was replaced with checking that the test HWND never becomes foreground;
other user window switches no longer make the assertion flaky.

Matched Go 1.27.1 stripped windowsgui/trimpath/buildvcs=false builds increased
the host from 4,661,248 to 4,671,488 bytes: +10,240 bytes / 10 KiB, about 0.22%.
This is size evidence, not a latency benchmark. The added owner record exists
only for opting-in windows and is removed on destruction. No dependency,
worker, timer, public IPC method/permission, DB or state-format migration is
added. Physical monitor dragging and visually shrinking packaged windows
remain unverified. Version remains beta.20; no release publication occurred.

## Window Attention Runtime: 2026-10-04

Independent `window.attention` enables bounded request and explicit cancel
without window activation or visibility changes. Scoped IPC/WebView2/manifest/
runtime-config/builder tests and go vet passed. Build-report propagation is
covered by builder tests, not a separate build-report suite. Native tests use
a disposable hidden window to check FLASHWINFO layout, taskbar-only bounded
flags, accepted calls without interpreting BOOL as success, unchanged
foreground/visibility and invalid/destroyed handles. They do not establish
visible taskbar flashing or foreground no-op behavior by human observation.

Matched stripped Windows GUI, trimpath, buildvcs=false host builds compare
the previous title host (4,655,616 bytes) with attention (4,661,248 bytes):
+5,632 bytes / 5.5 KiB, approximately 0.12%. No startup performance claim,
dependency, host timer, DB, repository-hygiene or runner change is made.
API/spec/ADR/configuration/validation sources are updated; runtime version
remains beta.20. No push, remote CI dispatch or publication is part of this
local verification.

The repository hygiene suite initially rejected File Notes' already-approved
window.title grant because its exact permission list still expected two file
permissions. The assertion now requires exactly file.open, file.save and
window.title, and the suite passed. File Notes gains no attention permission.

## Window Attention Example: 2026-10-04

The independent Window Attention 0.1.0 example requests only window.attention.
It has numeric count/delay controls and Lucide request/cancel buttons, no
network or persistence, and one frontend timeout only after an explicit click.
Six Bun tests passed: explicit/snapshot-bound requests, duplicate suppression,
delayed and in-flight cancellation, late-response isolation, strict inputs,
redacted failures, missing bridge and pagehide cleanup.

Mock-native Edge checks at 680 x 400 and 360 x 400 passed loaded icons,
no horizontal overflow, stable 44-pixel buttons, Enter request and Space
cancel, scheduling and control recovery. Screenshots were visually checked.
These checks do not observe real taskbar flashing.

Two builds produced the same 2,324,106-byte ZIP, SHA-256
`b0bf293ec7960be1eeaa2e554c692e71a48f038a09a8eb8b6403857d53bd0fc5`.
ZIP inspection confirmed the sole permission and packaged host SHA-256
`b1c33356c917450e1b4a35c314c75395b9431679d3fd16a79c284752a32a4885`.
Direct packaged launch with a private profile and exit-after-ready returned
zero. Actual background-window blinking, explicit visual stop and foreground
no-op are still manual checks. No release/version change, push or hosted CI
dispatch occurred; existing File Notes and recovery data were not replaced.

## Dynamic Window Title Runtime: 2026-10-04

`window.title` independently enables `window.setTitle({title})`, without
changing app identity or native approval text. IPC, WebView2, manifest,
runtime-config and builder tests passed; build-report has no standalone tests
and its permission propagation is exercised by builder checks. Scoped
`go vet` passed. A disposable hidden Win32 window verified Korean/ASCII/empty
caption readback and destroyed-window rejection. An initial failure on a
destroyed handle was fixed with an explicit IsWindow check and retested.

Matched Go 1.27.1, buildvcs=false, trimpath and stripped windowsgui builds
compare pre-change HEAD a753ea3 (4,651,008 bytes) with the title runtime
(4,655,616 bytes): +4,608 bytes / 4.5 KiB, approximately 0.10%.
This is size evidence, not a startup-latency benchmark. No dependency, worker,
DB or runner change was made. Packaged File Notes interaction is not yet
manual evidence; no version bump, remote CI dispatch or publication occurred.

## File Notes Title Candidate: 2026-10-04

File Notes now opts into `window.title`. Its native caption follows restored,
opened, new, saved and renamed documents, with a dirty marker. Identical
captions are not sent again on each edit. Caption failures do not block file
actions; 33 Bun tests passed, including these transitions, cancellation,
restoration, denied/unavailable title calls and existing save/find behavior.

Two local builds produced the same 8,443,807-byte ZIP with SHA-256
`6e90d293594ca45e0453ee1f7a26ccc0d99f31a53e81443e3d4ecfe99dde1eab`.
ZIP and directory inspection both passed with the three explicit permissions
`file.open`, `file.save`, `window.title`. The packaged host SHA-256 is
`87e87950ad3b0d5b19bfe3bcb6ae200ced9aa174b75ed8ed8b25bf974e41ddf8`.
Local runtime and example versions remain beta.20 and 0.4.0; this candidate is
not a new published release.

Direct packaged launch and CLI source run with private profiles and automatic
exit-after-ready returned zero. Two earlier Start-Process Hidden checks timed
out and their test processes were stopped; do not count hidden-window readiness
as passed or infer its cause from the successful ordinary launches. An initial
inspect used the wrong output path: relative build output resolves from the
manifest directory; rerunning against the actual candidate passed.
Manual packaged filename/dirty-marker interactions remain unverified.
Existing app files and the user's recovery profile were not replaced.

## Clipboard Example Packaging: 2026-10-03

The independent Clipboard 0.1.0 example (`dev.velox.clipboard`) opts into only
`clipboard.write` and `clipboard.read`. Copy writes only on a button action;
Paste waits for the native result, preserves prior output on refusal/error,
and displays Unicode through a read-only textarea rather than HTML. Both
buttons are disabled while work is pending. No clipboard content is saved,
logged, sent over a network or automatically read/written. The example neither
replaces File Notes nor uses its profile/recovery data.

Four application tests passed. Mock-native Playwright/Edge checks at 880 x 620
and 360 x 620 passed role/name/label snapshots, Enter/Space button activation,
focus return, cancellation preservation, Unicode/literal markup display, icon
loading and long text/error overflow checks. These are browser interaction
checks, not actual Windows clipboard approval or Notepad interoperability.

The clean runtime build is from `218f30d7ac22ccaadeba3471a527626566da24b9`,
with input host SHA-256
`23fe1278886d3ac03a4a2e3044a202d7472ad811ce667417c17031b7dbb49f1e`.
The 4,651,008-byte host has the same size as the pre-commit native smoke host;
its build metadata and digest changed. The earlier smoke is source-equivalent
runtime evidence, not an exact-byte run of this later clean binary.
The example built twice with identical 2,321,323-byte ZIPs, SHA-256
`4af5accdcabba695d6598a2f16be0e6aaa41cb02c98d08f97e98944252d1b07b`.
Directory and ZIP inspection passed with nine portable files, 7,813 asset
bytes, app version 0.1.0, runtime beta.20 and exactly the two declared permissions.
Output is under `dist/manual/clipboard-beta20/example/`.

The first packaging attempt used unsupported `--manifest` and returned
`USAGE_INVALID` before packaging. After checking CLI help, using `--config` and
an absolute owned output path produced the successful packages above. The failed
attempt is not counted as a passing build. All checks were direct local runs,
not Mustflow receipts. Actual native confirmation and paste remain manual.
No hosted CI, installer, public publication, dependency, DB, repository hygiene
rule, runner or CI workflow change was made for this example.

After the requested disposable-text Copy, Paste refusal/approval and Notepad
check in the prepared Velox Clipboard window, the maintainer reported success.
This is general manual confirmation for Clipboard 0.1.0 on beta.20, separate
from mock-native Edge checks. Individual action traces, exact clipboard byte
readback, non-text formats and real clipboard-contention behavior were not
independently recorded; those narrower claims are not inferred from the report.

## Confirmed Clipboard Text Reads: 2026-10-03

Local beta.20 adds only `clipboard.readText({})` under independent, default-off
`clipboard.read`, with per-request native Yes/No approval (default No).
Cancellation opens no clipboard and returns no text. Reads are deferred outside
WebView callbacks, limited to one pending confirmation, and bound to the active
document. Navigation and shutdown prevent stale text disclosure. The existing
write permission does not grant reading. See [ADR 0030](../adr/0030-confirmed-clipboard-text-read.md).

Tests passed for permission and parameter denial, deferred/single completion,
approval/cancellation ordering, queue rejection, stale document/shutdown,
Unicode/UTF-8 limits, bounded copying, borrowed-handle cleanup and native-error
redaction. The Windows memory-copy test uses owned allocations and does not
read or replace the maintainer's clipboard. Related manifest/runtime/build
permission propagation, builder, CLI, inspector, runner, releasebundle,
WebView2 and hygiene tests passed, as did targeted Go vet.

The pre-commit GUI host built with Go 1.27.1, trimpath and `-s -w -H windowsgui`
is 4,651,008 bytes: +23,040 bytes (22.5 KiB, about 0.5%) versus the beta.19
host built with the same toolchain (4,627,968 bytes). This is a local binary-size
comparison, not a zero-cost or unchanged-startup claim. Native built-host
startup/security/lifecycle smoke passed in 43.18 seconds, including early close,
icons and the GUI subsystem. Immediate readiness remained about seven seconds;
the existing relaunch limitation is unchanged. Native clipboard prompt/paste
interaction is separate manual evidence; the subsequent maintainer confirmation
is recorded in the Clipboard Example Packaging section above.

IPC v1 gains an additive optional method and permission; existing apps retain
their grants. Permission schemas, product scope, configuration and IPC docs
are synchronized. DB, dependencies, repository hygiene, runner selection and
CI workflows are unchanged. No clipboard listener, timer, retry, stored approval,
public release or installed-app replacement was added. Additional hosted CI
and installer generation were intentionally omitted for this local capability.

## File Notes Find in Document: 2026-10-03

File Notes 0.4.0 adds Ctrl+F and an accessible search icon, literal
case-sensitive non-overlapping matching, a result counter, next/previous
navigation with wraparound, Enter/Shift+Enter and Escape-to-editor focus return.
Search does not change document text, dirty state, saved baseline or recovery
drafts. IME composition, key-code 229, repeated Enter, pending file work and
the open discard dialog cannot navigate results. Regex, replacement and case
folding are not included.

Search retains a count and one selected position rather than a position array.
A temporary accessibility-hidden mirror measures wrapped text through browser
layout to reveal the selected result and is removed when search closes. Dense
2 MiB scanning took approximately 434 ms in one local measurement; this is not
a latency guarantee, and large wrapped documents can also require layout work.

All 30 application/find/model/storage tests passed. Playwright/Edge checks at
1080 x 760 and 360 x 740 verified icon loading, no horizontal overflow, visible
selection scrolling, Escape focus return and mirror cleanup. Native file calls
were mocked; these checks do not prove actual WebView2 keyboard routing or
native dialog interaction. The narrow native-access hygiene test passed.
After the requested Ctrl+F, Korean-query, Enter/Shift+Enter and Escape check
with the prepared EXE, the maintainer reported that search works well in the
actual Velox window. This is general manual success confirmation for File
Notes 0.4.0 on beta.19, separate from the mocked Edge checks. Per-action
results, real IME composition and large-document latency were not independently
recorded; those narrower claims are not inferred from this confirmation.

The example was packaged and inspected with the existing beta.19 CLI/host under
`dist/manual/file-notes-find/package/`. The ZIP contains 17 portable files,
is 8,435,363 bytes, and has SHA-256
`f2a4e8e372f42ebfc208eda036e40ed2ac2d22dd0890185c6754d42695d041cc`.
Permissions remain exactly `file.open` and `file.save`. The input runtime host remains
4,627,968 bytes with SHA-256
`5a3d9595c3a0854f4fa69df4295352b880c62764c63213b02dda22acf6bab62d`.

Only the example version changed from 0.3.0 to 0.4.0. Runtime source/version,
IPC, permissions, dependencies, DB, CI workflow and repository hygiene rules
are unchanged. Bundled Lucide/Feather icon licenses accompany the new assets.
These are direct local checks, not Mustflow receipts. Runtime rebuild,
installer generation and additional hosted CI were intentionally omitted for
this frontend-only change. No installed app, open document or user profile
was replaced, and no public release was published.

## File Notes Keyboard Actions: 2026-10-03

File Notes 0.3.0 adds app-local Ctrl+S, Ctrl+Shift+S, Ctrl+O and Ctrl+N through
the existing button click handlers. The connected save, cancellation, error,
permission and unsaved-change confirmation paths are unchanged. IME composition
(including key-code 229), repeated keys, pending file work, and the open discard
dialog cannot dispatch another action. Unsupported modifier combinations and
unrelated keys are not intercepted. Native button labels and tab order remain
unchanged; `aria-keyshortcuts` exposes the four bindings.

All 23 File Notes application/model/storage tests passed. Playwright/Edge
actual keyboard events exercised initial Save as, connected Save, forced Save
as, Open cancellation and New, plus discard cancellation and acceptance.
Native calls were mocked; these checks do not establish WebView2 accelerator
routing or actual native dialog interaction. The narrow File Notes native-access
hygiene test, local link checks and whitespace checks passed.

The example was packaged and inspected using the existing beta.19 CLI/host,
with only `file.open` and `file.save`. Two builds produced the same portable ZIP
digest `facc575a82b2e5b7da1a9b83fafda8d9db2473772747aa028a7b9fe00306ec8c`
(8,429,399 bytes). Output is isolated under `dist/manual/file-notes-keyboard/`.
The input host remains 4,627,968 bytes with SHA-256
`5a3d9595c3a0854f4fa69df4295352b880c62764c63213b02dda22acf6bab62d`.
An initial host-preservation command used PowerShell's reserved `$Host` variable
and did not perform that comparison. After correcting the variable, a repeat
build verified the host digest and identical ZIP bytes; the initial measurement
error is not counted as a successful check.

No runtime source, IPC, permission, dependency, DB, CI workflow or host version
change was needed. Only the example version changed from 0.2.0 to 0.3.0.
These are direct local checks, not Mustflow receipts. No existing output,
installation, open document or user profile was replaced. Additional hosted CI,
runtime rebuild, installer generation, push and public
publication were not performed for this frontend-only change.

After the requested actual Velox-window shortcut check with the prepared EXE,
the maintainer reported success. This is manual success confirmation for File
Notes 0.3.0 on beta.19, separate from the mocked Edge checks. Individual shortcut
results, saved-file byte readback and real IME coverage were not independently
recorded; those narrower claims are not inferred from the general confirmation.

## Opt-In Clipboard Text Writes: 2026-10-03

Beta.19 adds only `clipboard.writeText({text})` under the independent
`clipboard.write` permission and ADR 0029. The 32 KiB UTF-8 text and existing
64 KiB serialized IPC budgets remain bounded. No read API, listener, automatic
retry, worker, timer, database or dependency is added. Clipboard permission
allows trusted scripts to replace contents without an attested user gesture;
Folder Browser 0.3.0 invokes it only from its explicit copy button.

Related clipboard, IPC, manifest/runtime, host, CLI, builder, plan, inspector,
runner and hygiene tests passed locally. Targeted `go vet` passed after fixing
the initial native-pointer conversion warning with `RtlMoveMemory`. Real
Windows movable-allocation Unicode readback passed without accessing the
system clipboard. Folder Browser's ten Bun tests passed, including click-only
copy, pending-operation suppression, empty text and failure preservation.
Playwright/Edge layout checks and screenshots at 900x650 and 360x740 verified
the copy icon, long filename wrapping and no horizontal overflow; those used
mock IPC and do not prove native clipboard interaction.

Host size comparison used Go 1.27.1, `-trimpath` and
`-ldflags="-s -w -H windowsgui"`, with CGO disabled. The exact beta.18 runtime ZIP
from the distribution record supplied the `b79fc01` baseline host, 4,611,584
bytes. The clean `a85249a` beta.19 host is 4,627,968 bytes: +16,384 bytes
(0.355%). Both report matching toolchain/target settings and clean VCS metadata.
This comparison includes version/revision metadata changes and does not claim
zero startup or invocation latency impact.

Local release assembly, Folder Browser build and package inspection passed
with runtime beta.19, app version 0.3.0 and exactly the three declared
permissions. Outputs are isolated under `dist/manual/clipboard-beta19/`;
the beta.18 distribution and File Notes installs were not replaced. These are
direct local command results, not Mustflow receipts. No hosted CI, push or
publication was performed. The isolated Folder Browser was launched with the
private `.cache/clipboard-beta19-manual-profile` profile. After the requested
copy-button / paste-in-an-editor check, the maintainer reported that copying
worked. This is manual success evidence for the prepared beta.19 package, not
an automated clipboard readback. The selected file, pasted bytes and Unicode
coverage were not independently recorded. The maintainer's clipboard was not
read or replaced by automated tests. App closure was not separately confirmed.

## beta.18 Local Distribution Preparation: 2026-10-03

The exact beta.18 source `b79fc01` passed hosted Windows Consumer evidence
run [37110431391](https://github.com/0disoft/velox/actions/runs/37110431391).
An installer-enabled runtime bundle and both File Notes / Folder Browser
portable ZIPs and Setup EXEs are prepared locally with matching beta.18 runtime
metadata. Runtime-manifest checksums, ZIP inspection and exact Setup template /
payload verification passed. The exact Folder Browser Setup was then installed,
launched through its Start Menu shortcut, manually checked for folder listing
and text preview, and removed. Installed-file and shortcut hashes matched;
the install tree, shortcut and uninstall registration were removed while 217
profile files and seven selected-folder files retained identical hashes.
The normal temporary uninstall helper remains. File Notes Setup interaction was
not checked. No additional CI or publication was performed; this does not fix
or erase prior intermittent local failures.
See [local distribution record](beta18-local-distribution.md) for exact files,
sizes, digests, delivery limits and user instructions.

## Immediate Folder Text Reads: 2026-10-03

Beta.18 adds `folder.openText` with separate opt-in `folder.readText`, requiring
`folder.read` as well. Existing listing-only apps remain denied. Reads use an
identity-checked directory handle and a validated immediate basename, with
2 MiB UTF-8 limits and no write grant. Real Windows tests covered unchanged
file bytes, Unicode/BOM/invalid/oversized text, offline/directory/hard-link/
reparse rejection, and pinned-handle reads after directory rename/replacement.
Related file, IPC, manifest/runtime, build/CLI/inspect/runner/WebView and hygiene
tests passed, as did related Go vet. The source host measured 4,611,584 bytes,
15,360 bytes (15 KiB, about 0.33%) above beta.17; no dependency, background work,
DB, repository hygiene or CI workflow change was added.

The built-host GUI subsystem, default icons, lifecycle and security-policy
subtests passed. The initial early-user-close subtest failed because one browser
process exceeded its 10-second exit bound; that process was no longer present
when inspected. A single isolated rerun passed all three early-close attempts.
This intermittent result is retained, not claimed as a fixed lifecycle defect
or a uniformly passing initial smoke. No hosted CI, push or public release is
claimed for beta.18. Actual Folder Browser text preview is separate manual evidence.

Folder Browser 0.2.0 now declares both permissions and reads only on a file-name
click. Its eight Bun application tests passed, including literal rendering,
cancellation, stale-row suppression, busy-state serialization, unsupported-file
handling and connection-expiry cleanup. Validate/doctor, two identical ZIP
builds, inspection and packaged/source startup passed with private profiles.
The ZIP measured 3,130,333 bytes, SHA-256
`e83c54e8f040009905283d1bac2292545cf8886c17b61742c9203ae53f987c33`.
The maintainer reported success after the requested folder selection and text
file click/preview check with the beta.18 example built from `dec8669`. This is
user-reported manual evidence, not independent byte-for-byte content comparison,
layout screenshot review or confirmation of every cancellation/error case.
File Notes is unchanged.
The first separate `dist/examples/folder-browser-beta18` build returned the
generic `PACKAGING_FAILED` diagnostic without a specific cause. One unchanged
retry succeeded with the same ZIP digest above. The initial failure's cause
remains undetermined; it is not erased by the successful retry.

## Bounded Folder Listing Local Checks: 2026-10-03

Beta.17 adds `folder.list` and Folder Browser 0.1.0 under only `folder.read`.
It checks the selected directory's identity, enumerates its handle, considers
128 immediate entries plus one look-ahead, and bounds serialized results to
32 KiB. Entry names/kinds, truncation and examined exclusions are returned;
no child path, content, recursive scan, total count or persisted grant is added.
Reparse/offline/encrypted entries are skipped without following them.

`go test ./...` and `go vet ./...` passed, as did 27 related Bun tests.
The built-host GUI subsystem, lifecycle and security-policy smoke passed.
First ready was 693 ms and immediate same-profile relaunch 7.37 seconds in that
single local sample; this is not a latency improvement or broad benchmark claim.
Folder Browser passed validate/doctor, two identical ZIP builds, inspect and
packaged/source startup with private profiles. Its ZIP measured 3,124,209 bytes.
The locally built source host measured 4,596,224 bytes: +45,056 bytes (44 KiB,
about 0.99%) versus beta.15, including selection and listing. No new dependency,
background polling, watcher, worker, database or CI workflow was added.

The maintainer reported success after the requested Folder Browser selection,
refresh, reselection cancellation and explicit release sequence. This is
user-reported manual evidence for beta.17 at `fcc4e63`, not an independent
screenshot review or confirmation of every exclusion and truncation case.
The tested ZIP SHA-256 is
`0d249c33275b9285f87e53e4c616c1b92cad65bd3523f0521b3502054135eff8`.
In-app browser screenshot inspection was unavailable because local file URLs
are blocked; no alternate browser path was used to bypass that policy.
No new hosted CI, push, installation, signing or public release is claimed.

## Local Folder Selection: 2026-10-03

Beta.16 adds opt-in `folder.read`, native `folder.select` and explicit
`folder.release` under ADR 0028. Related file/folder, IPC, manifest/runtime,
build/inspect/runner, WebView and hygiene suites passed; related Go vet passed.
Real Windows tests covered dialog options, folder identity, DOS short names,
offline/file/remote/link rejection. Engine/IPC tests covered cancellation,
replacement, generation/revocation, busy/exhausted tokens, permission/parameter
denial and shutdown. These do not claim actual manual folder selection.

The source host measured 4,574,208 bytes, 23,040 bytes (about 0.51%) above the
beta.15 source host's 4,551,168 bytes. No dependency, persisted grant, watcher,
timer, idle worker, DB or workflow was added. Listing is a separate follow-up;
no public release or new hosted check is claimed by this local evidence.

## File Notes Native File Migration: 2026-10-03

File Notes 0.2.0 uses `file.openText`, `saveTextAs` and `saveTextTo` under only
`file.open` and `file.save`. New and successful Open release the prior target;
canceled Open/Save as preserve it. Open remains read-only, so the first Save
after Open or draft restoration requires a fresh native selection. Schema-v1
draft text and saved baselines remain readable; legacy browser handles are
ignored and new draft writes exclude handles, paths and native target tokens.
External-change conflicts, expired targets and save failures preserve editor
contents without automatic retry or overwrite. Nineteen File Notes model,
application and draft-storage tests plus four bridge tests passed. Six related
Go package suites (buildplan, builder, CLI, inspector, runner and hygiene) passed.
The beta.14 example smoke passed validation, doctor, two byte-identical builds,
ZIP inspection and packaged/source startup with isolated profiles (exit 0).
No runtime file API, dependency, schema, DB version or CI workflow was changed.
The maintainer reported success after the requested beta.15 Save as, edit,
connected Save and restart/draft-recovery check. This is user-reported manual
evidence, not independent byte-for-byte readback or separate conflict/cancel
confirmation. Source: `d471d87`; app ID: `dev.velox.filenotes`; packaged host
SHA-256: `3ee5803c74e7f0303a916d991d9b1da11a65d669948f713e30c7fd685f4f82a7`.
External-change interaction remains separately unverified for File Notes.

## Native Save Short-Path Follow-Up: 2026-10-03

[Consumer evidence run 37039373327](https://github.com/0disoft/velox/actions/runs/37039373327)
failed the Windows native-save tests at source `c3ab0a5`: existing-file saving
and post-save snapshots returned `UNSUPPORTED_FILE`. The same failure was
reproduced locally by saving through a real DOS short-name directory alias.
The handle's normalized long path differed from the input spelling.

Beta.15 retains the final-path equality check, expanding the input with
[GetLongPathNameW](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getlongpathnamew)
only when direct case-insensitive equality fails. Link, drive, attribute and
hard-link rejection remain unchanged. The regression now passes new saving,
snapshot, connected saving and replacement through the alias, with long-path
readback. File-open, conflict and rejection tests passed, as did `go test ./...`
and `go vet ./...`. No permission, IPC, DB or workflow change was made.
[Consumer evidence run 37040338144](https://github.com/0disoft/velox/actions/runs/37040338144)
passed at exact source `d471d87182bfa0f9b93ab82938a798155672dcca`, including
native file tests, release build, startup/security/permission recovery, lifecycle
and checkout-free consumer checks. The original failed run remains recorded.

## Document-Scoped Save Local Checks: 2026-10-03

Local beta.13 adds ADR 0027's explicit connected Save as/Save. The full Go suite,
`go vet ./...` and four Bun bridge tests passed. Real disk tests verified repeated
save/readback and rejected external edits, same-size edits with restored write
time, replacement and deletion without damaging external contents or recreating
the deleted file. Engine and IPC tests cover cancellation retaining the previous
target, revocation of queued writes, document-generation mismatch, navigation
and shutdown cleanup, changed post-write readback, permission denial and strict
parameter routing. Existing picker-per-call saving and read-only opening remain
unchanged. No watcher, worker, timer or new dependency was added.

The Text Writer 0.2.0 example now provides New, Save and Save as with session-only
target reuse and error-buffer preservation. The maintainer reported that the
beta.13 connected-save example worked after the requested manual check. This
is user-reported success, not independent disk readback or separate confirmation
of every cancellation and external-change conflict path. The tested source is
`134e63a`, with packaged host SHA-256
`6e41f6828ed9b1196a0430cde282cf19ce24490ecc4037e86ec020aa09ecd347`.
Cancellation and external-change conflict UI remain separately unverified.
File Notes migration, draft/file reconnection after restart, hosted CI, push,
publication and beta-channel approval are not claimed by these local checks.

## Native Text Save Manual Check: 2026-10-03

The maintainer confirmed saving, overwriting and opening the saved text in
Notepad worked in the beta.11 Text Writer. These are user-reported observations,
not independent byte-for-byte readback. The supplied screenshot also showed an
empty file-type selector: the native dialog had no registered file-type list.
Save-as cancellation and overwrite-prompt cancellation were not explicitly
confirmed in this report.

The tested package is associated with source
`61448ab720b0788d8326b07f7c4ec8b5614f9030`, app ID `dev.velox.filesaver`.
The currently packaged host SHA-256 was rechecked when recording the report:
`9d143a6217dfedf27f54e2510d84279e0bdb60e3a990c0c32f5a1f8bdea020a7`.
This does not certify later rebuilt bytes or claim File Notes migration.

The beta.12 follow-up registers Text documents, Markdown and All files filters,
selects the initial filter from the suggested name, and enables automatic
default-extension handling. This changes filename selection, not UTF-8 encoding
or document conversion. Filter selection remains manually unverified.

## Native Text Save Local Checks: 2026-10-03

Local beta.11 implements ADR 0026's `file.save` and the frozen `saveText` helper.
The full Go suite, `go vet ./...`, three Bun helper tests and the rebuilt host's
`TestBuiltHostStartup` passed. Disk tests cover new-file and existing-file
readback, empty/Unicode/exact 2 MiB text, restricted DACL preservation and
rejections without damaging the original. The helper tests cover 2 MiB staging,
Unicode/escaping boundaries, unchanged 64 KiB message limits and upload cleanup.
The Text Writer package's directory and ZIP both passed public CLI inspection
with only `file.save`. Initial inspection used the wrong relative output
location and failed; rebuilding to an absolute output path resolved that error.

The measured source-host build is 4,521,472 bytes, 48,640 bytes (about 1.09%)
above the recorded beta.10 host. No idle CPU or latency improvement is claimed;
staging and disk I/O are bounded, on-demand costs. Existing same-profile
immediate relaunch latency remains (7.05 seconds in the passing startup check).
At that stage, native Save as, cancel and overwrite-confirmation interaction
was manually unverified; the follow-up report above records later observations.
File Notes has not migrated; save-to-last-path, hosted CI,
publication, signing and beta-channel approval are not claimed.

## Native Text File Open Manual Check: 2026-10-03

The maintainer confirmed the requested real file-selection and cancellation
checks worked in the read-only Text Viewer example. This is user-reported
manual evidence, not an automated dialog observation or an independently
recorded byte-for-byte comparison of the selected file.

The local beta.10 example package was built from source commit
`b12753134e263dd38eb216626b8b566919971724`, app ID `dev.velox.filereader`.
Its host executable SHA-256, rechecked when recording this confirmation, is
`b6c0db6d8942975bd958771821051079d7867ac2452037fac55ad941b728aef3`.
The example requests only `file.open`; it does not write files or retain a
native file-access grant. See [ADR 0025](../adr/0025-selected-local-text-file-opening.md).

Native saving and File Notes migration remain separate work. This confirmation
does not imply a hosted CI pass, publication, signing, or beta-channel approval.

## External HTTPS Manual Check: 2026-10-02

The maintainer confirmed the allowed fixture behaved as requested: cancelling
HTTPS confirmation opened no browser, approving the next request opened the
Velox GitHub repository, and HTTP/File requests returned `INVALID_PARAMS`
without confirmation. These are user-reported observations. The launcher
separately recorded exit code 0 without a timeout.

Fixture source was `8ae352d1941bc97320320edc92e233ec1ed8cf7a`; the existing
host executable SHA-256 was
`bc35864487532ab49e3bb2f8547bbe6f1b8004b969b130556e0534f29af52752`.
The source identity describes the fixture checkout, not proof that this
prebuilt host was produced from that commit. Both fixtures used disposable
profiles. The denied fixture reached its eight-minute limit without a manual
response; `PERMISSION_DENIED` remains manually unverified, not passed.
No user profile, release artifact, or browser was terminated by the harness.
See [ADR 0023](../adr/0023-confirmed-external-https-links.md).

## System Tray Manual Check: 2026-10-02

The maintainer reported all requested interactions passed in the isolated
`Velox File Notes - Tray Check` app: Hide window, reopen by clicking the tray
icon, cancel Quit with unsaved text while retaining the window, then Quit and
confirm the icon disappears. The maintainer also reported the test window was
closed. This is user-reported manual evidence, not an automated observation.

The tested beta.8 build came from source commit
`87396cced4f917ff4b932772cc1a4e345eff3818`, app ID
`dev.velox.filenotes.traycheck-1003865f`, with executable SHA-256
`dda781ebb5dc3cf74dec8ef722f5239c859cf86455d33a947ae28656f1a4239c`.
It used a separate profile; the existing release ZIP, File Notes ZIP, Setup,
and normal packaged executable were not replaced.

The bounded launcher receipt separately records a timeout with no exit code.
It does not prove normal process exit and must not be relabeled a successful
automated run based on the later manual report. Live Explorer-restart recovery,
hidden duplicate-instance activation, and comparative idle CPU remain
unverified by this interaction. No performance, hosted CI, publication, or
beta-approval claim follows from this check. See [ADR 0024](../adr/0024-opt-in-system-tray.md).

## Windows Installer: 2026-10-02

The per-user Windows Setup executable is an optional artifact under ADR 0020,
not a beta gate: the portable unsigned ZIP remains the default output. The
install/removal engine and the Setup payload format are implemented in
`internal/installer`, `internal/setuppayload`, and `cmd/velox-setup`, and the
CLI exposes `velox build --installer`. The setup template commit is `69949da`
for the engine. The full `velox_test` Go suite plus `go vet` passed, and an
installer-enabled beta.3 release bundle was built locally.

All four installer intents are locally verified for beta.3:
`velox_installer_test`, `velox_installer_bundle`, `velox_installer_smoke`
(unique app ID `dev.velox.installer-smoke-3876`, identical two-build bytes,
installed tree plus real Start Menu shortcut and registry checks, installed GUI
startup with the two-render-frame readiness marker, real helper uninstall,
preserved user test document, and script cleanup), and
`velox_file_notes_installer` (a distributable File Notes Setup, 12,361,886
bytes) passed locally. This is local harness evidence, not hosted CI, a push, a
release, or a manual install, so it does not change the beta gate. Install
behavior and the residual normal-uninstall `%TEMP%` helper are documented in
`docs/ops/windows-installer.md`.

## Initial Beta Support Scope: revised 2026-09-24

The maintainer selected Windows 11 x64 desktop versions still serviced by
Microsoft, with an installed, updating Evergreen WebView2 Runtime. The display
scope is one physical monitor. Display scaling is best-effort in the initial
beta: 125% was previously observed with File Notes 0.1.1 on public alpha.62.
On 2026-09-24, the maintainer reported sharp, unclipped File Notes rendering
at 100% and 150% on another laptop. Its Windows/WebView2 versions and the
transferred ZIP's bytes were not independently recorded; this is a manual
visual report, not a picker/save or full platform verification at each scale.
Scale-specific visual checks remain nonblocking unless a known issue breaks a
core workflow. An initial beta would use a portable,
unsigned ZIP, with unsigned-execution warnings and the known same-profile
relaunch delay disclosed. Windows 10,
Windows Server, ARM64, multiple monitors (including mixed-DPI movement), and
fixed or absent WebView2 runtimes have no initial beta support commitment.
The older Windows/WebView2 floors in the product spec remain technical
compatibility checks, not evidence that those environments were validated.

This is a support-policy decision, not a completed DPI check, channel approval,
or published beta. Do not describe unverified scaling as tested. A configuration
outside the scope may run, but must not be advertised as supported by the
initial beta.

## Remaining Beta Decision: 2026-09-24

This is the current action list; dated sections below preserve historical
decisions and do not reopen checks that later evidence completed. ADR 0019
still requires a separate maintainer channel decision. No beta is authorized
by this classification, and no missing observation is relabeled as a pass.

### Outstanding Release Checks

| Item | Why it remains open | Completion criterion |
| --- | --- | --- |
| Channel decision and support disclosure | The beta support scope is now decided; public alpha.62 and the separate File Notes 0.1.1 ZIP are bound to source, ZIP and host digests below. Unsigned support limits, known restart latency and incomplete cross-laptop environment and artifact records remain | Disclose these limitations, then make a separate maintainer beta channel decision; no automatic promotion |

Public alpha.62 source-free verification, its 50-pair hosted lifecycle run,
three public pre-ready close/relaunch pairs, and the maintainer's installed
save/recovery confirmation are completed evidence, not another repetition
queue. Native cancellation, denied-write protection and post-denial Save as
also have recorded evidence. Human menu cancellation and acceptance, followed
by maintainer-confirmed saving, opening and restart/overwrite, are recorded
below on 2026-09-22. An automatic new release or full test campaign is not required merely
to update this checklist.

### Follow-Up Improvements

- Same-profile restart latency is a known performance limitation, not an
  observed lifecycle deadline failure in alpha.62: hosted p50 5.96 s, maximum
  6.36 s. Keep it visible in the candidate decision. A crash, missed existing
  shutdown deadline, residual profile lock or lost draft reopens a blocker;
  do not force browser termination or rotate user profiles to hide the delay.
- Broader Windows/WebView2 and mixed-monitor coverage, plus repeatable
  per-scale checks on a pinned candidate, can follow the initial beta. The
  manual 100%/150% visual report does not verify every interaction or platform
  configuration. A scaling issue that breaks a core workflow takes priority
  over cosmetic follow-up.
- AI/model evaluations and independent-user feedback are optional evidence,
  not product dependencies or substitutes for the remaining native checks.
- Signing and stable-channel support remain separate decisions under ADR 0019.
  Preserve unsigned warnings; the existing alpha verification is not a signing
  or stable-readiness claim.

The candidate evidence below binds the public alpha.62 source, ZIP and host
digests separately from the local File Notes ZIP. The maintainer's 100%/150%
visual report adds cross-laptop evidence, but the laptop environment and copied
artifact hash remain unrecorded. Mixed-monitor movement remains unverified and
outside the initial beta support scope. Further visual checks are not beta
blockers by themselves.
Any newly observed security, data-loss or core-workflow defect takes priority.

## Desktop Delivery: 2026-09-16

Public alpha.62 fixes the early user-close exit classification. The regression
fails on the unchanged public alpha.61 host and passes on the local candidate,
hosted CI, and the newly downloaded public alpha.62 EXE, including three
close/relaunch pairs and the genuine missing-runtime case. Publication run
35079091056 and public verification run 35079337819 passed. The installed
File Notes app was not replaced during publication verification. It was later
updated to the verified public alpha.62 host without changing its web assets,
runtime configuration or user profile. The maintainer confirmed that content
restoration and saving worked after installation; this is user-reported manual
evidence, not an automated permission-denial or every-path check.
Hosted stress run 35081507786 subsequently passed 50 complete fresh/immediate
pairs against that public host. See [lifecycle evidence](alpha61-lifecycle.md).

The user separately confirmed installed Noto File Notes 0.1.1 works at 125% on
the public alpha62 host; 150%/mixed-monitor was user-skipped and remains
unverified, not passed. This is not a beta approval. The isolated permission
fixture's later manual results are recorded below.

The earlier unsigned alpha.61 was published. Publication run 35063918809 and public-download
verification run 35064135758 passed. The maintainer separately confirmed
Explorer-launched Save as, exit, relaunch, and Save for the earlier CI alpha.61
example. Artifact identities and the native permission menu's human/API test
boundaries are recorded in [file-permission recovery](file-permission-recovery.md).
The unchanged beta checks above are not waived by this alpha publication.
The dated records below describe their original artifacts, not this new ZIP.

## Permission Recovery Manual Confirmation: 2026-09-22

The disposable `dev.velox.permissionmenu20260922` app used a private profile
and the public alpha.62 host, SHA-256
`651a9d87d16eee5687f4a1072226e3f9209a6ece438c0672e6c30e6680037679`.
Native setup seeded and read back FileReadWrite Deny for this app's origin;
the normal File Notes profile was not used. The executable hash was checked
again when recording these results.

- The maintainer selected No in the File access reset prompt. Reopening the
  menu showed the same blocked-state confirmation; the 40-character unsaved
  draft remained visible. This is observed menu behavior, not a new native
  permission-enumeration test.
- The maintainer then selected Yes. The observed result reported that the
  stored file access block was no longer present, no files were written and
  no automatic access was granted. The unsaved draft remained visible.
- After the test app was reopened with the same private profile, the
  maintainer confirmed Save as and file opening worked. After being asked to
  exit, relaunch, edit and Save, the maintainer confirmed that sequence worked
  as well. These are user-reported manual results, not automated disk-byte
  readback or process-exit measurements.

The disposable original input's SHA-256 remained
`0b7ec9664754aa3c30b33607c239f517848953efd32e19940d1ef7f4c2ff5aa6`.
That check concerns the original input, not the later saved output. Existing
native API tests retain responsibility for unrelated-origin/kind isolation;
this single-app manual check does not repeat them.

The earlier bounded harness timeout and later physical-Escape interruption
remain incomplete harness runs, not product failures or clean-exit evidence.
The subsequent maintainer confirmations complete the human menu interaction
and save/reopen/overwrite check at the evidence levels stated above. No
runtime, API, DB, version or release changed. Candidate reconciliation is
recorded below; user-skipped DPI coverage and the separate beta decision remain.

## Candidate Evidence: 2026-09-22

Public alpha.62 tag source:
`02c9acb5035014d9e29a0eb5881a3cf5310f5d6d`; public release ZIP
SHA-256: `10137ca603c5ba7f765d58f9e93fc78683f328aebad77659fd63e01367265871`.
Two development reloads, hosted 50-pair stress run 35081507786 and the
2026-09-22 permission-menu interaction used the public host SHA-256
`651a9d87d16eee5687f4a1072226e3f9209a6ece438c0672e6c30e6680037679`.
Earlier native cancellation and file-recovery checks retain their recorded
version boundaries; they were not all rerun on alpha.62.

The separate local File Notes 0.1.1 Noto ZIP was built from source
`a6d728dc4d0b634f7361ecf3db64d6ac45929b31`. Its SHA-256 is
`9658fbbee8f6e3c28029e2f2c48ae4214e1ac391a60903812cc4c4686f7cecee`
(7,746,360 bytes). Two builds matched; the extracted host matched the public
alpha.62 host, all eight web assets matched source, and a visible-window
startup exited 0 with a private profile. The initial hidden-window startup
timed out and remains an unresolved harness observation. On 2026-09-23,
the maintainer confirmed the extracted official ZIP's Save as, app restart,
Open and Save sequence worked with a disposable file. This is a user-reported
manual result, not independent on-disk byte readback. Other picker paths and
visual checks were not repeated on this exact ZIP. The maintainer separately
confirmed installed File Notes 0.1.1 at 125% and the save workflow. The 150%
and mixed-monitor checks were user-skipped and remain unverified.

Between that packaging source and this review, commit
`044070fad4bc5b8a132abe37a0de0d86f0759af6` changed only diagnostic
tool path selection. The product runtime is unchanged. Continue the unsigned
alpha; beta is not approved. Same-profile relaunch p50 was 5.96 s in the
hosted stress run. A separate maintainer channel decision remains required.

## Delivery Record: 2026-09-10

- File Notes now records only the submitted save snapshot as the on-disk
  baseline. Later edits stay dirty. File operations are serialized, and draft
  restore locks editing until completion.
- Eleven File Notes model/application tests pass using deterministic API
  doubles, including cancellation, denial, write failure and restored drafts.
- Native File Notes smoke passed normal and debug source startup, portable
  startup, inspection and byte-identical repeated builds locally.
- `velox run --debug` exposes existing host tools without changing app ID,
  profile selection, native permissions or packaged defaults.
- Alpha.49 publication and verification evidence is recorded in `docs/ops/release.md`.
- Final local Windows startup smoke failed once, then passed unchanged; its
  intermittent failure remains unexplained and is retained in the release record.
- Real native picker actions, interactive reload and durable restart recovery
  are not claimed by the automated tests above.

### File Save Permission Repair

Manual testing on 2026-09-10 reported `Write permission was not granted` for
Save and a platform-denied `showSaveFilePicker` call for Save as. The host's
global permission denial also denied WebView2 FileReadWrite (kind 8).
The alpha.50 gesture gate was incomplete: on 2026-09-11 a copied existing
profile emitted FileReadWrite requests with `IsUserInitiated=FALSE` while
restoring serialized handles. The host denied them before a picker action.
Clearing the saved write-guard entry alone did not fix this behavior.

The alpha.51 candidate delegates trusted-origin requests to browser DEFAULT
without treating that flag as an authorization decision. Browser activation
checks and consent remain in force; the host never returns ALLOW for these
requests. COM regressions for restored handles failed before the change and
pass after it. Foreign/missing origins, failed kind/origin reads, opt-out and
unrelated permissions remain covered.

With a copy of the affected profile, native diagnostics changed from kind 8 /
state 2 (DENY) to trusted-origin kind 8 / state 0 (DEFAULT). The user completed
Save as in the diagnostic app; the UI reported Saved to file and the resulting
file existed on disk. The original profile Preferences digest was unchanged.
Diagnostic instrumentation was an uncommitted Go overlay, not release code.
The immutable public alpha.49 release was not modified by this repair.

### Maintainer Confirmation: 2026-09-11

The maintainer confirmed that saving worked in the normal alpha.51 app, then
confirmed close/reopen recovery. This records user-reported manual evidence
for Save, Save as and restart recovery, separately from the copied-profile
diagnostic and automated tests. Those successful paths do not need another
repeat solely to update this record.

The existing application tests additionally verify that canceling a save
picker retains the draft and permits a later save, and that denied permission
opens no writable stream, preserves edits and permits retry when the browser
later grants access. These are browser API doubles, not native permission UI
evidence. The application does not override a browser denial or reset profile
permissions. Native cancellation and denied-consent recovery are still pending;
complete real picker and persistence checks remain unverified until those
remaining paths are covered. Use a disposable profile for a native denial test.

That confirmation update changed evidence and tests only. The runtime stayed
at alpha.51; publication followed separately as recorded below.

### Alpha.51 Delivery and Source Reload: 2026-09-11

The unsigned alpha.51 release pins source
`f18d7f3958c136b1f673b93255916771db3cde15`. Tag evidence run 34584656621,
publication run 34584828937 and public download verification run 34585168947
passed. The published ZIP digest and independent provenance binding are
recorded in `docs/ops/release.md`. This is same-repository consumer evidence,
not an external user attempt or beta approval.

A local `velox run --debug` smoke copied File Notes into a disposable project
and used a private profile. After modifying HTML, CSS and JavaScript, a normal
CDP Page.reload updated all three rendered probes while preserving origin.
Hard reload was not needed in that successful run. The smoke did not modify
the maintainer's open application, original project or profile.

An earlier run failed the combined reload condition before recording its
rendered state. The diagnostic was improved to retain normal and hard reload
results, and the next run passed normal reload. The first failure remains
unexplained; this is one successful browser-driven reload, not proof of
reliable repeated reload or a manually exercised keyboard shortcut. No
runtime reload fix was made. Native save cancellation, denied-consent recovery
and current-artifact lifecycle checks still keep beta held.

### Native Save Cancellation: 2026-09-11

An isolated session used the checksum-verified public alpha.51 ZIP, the
repository File Notes example and a fresh private profile. A real Save as
dialog was canceled with Escape. The editor retained the complete probe,
showed `Unsaved changes` and `Save canceled.`, and re-enabled its commands.
The subsequent Save opened another native picker and wrote `retry.md` in the
disposable test directory. The UI showed `Saved to file`; a separate disk
read confirmed the exact probe content.

Opening and editing the disposable `input.md` then clicking Save displayed
the browser's native write-consent prompt. The original file remained
unchanged while awaiting consent. Permission prompt choices require a human;
the agent did not grant, deny or reset permissions. At that point, denial and
later recovery were unverified; subsequent observations follow below.

The initial session hit its six-minute deadline while awaiting the manual
permission choice and was terminated by the wrapper. This is not a graceful
shutdown pass. Relaunching the same private profile restored the unsaved
63-character draft and its filename. Save then displayed the browser's
restored-file permission prompt; no permission choice is inferred from that
successful draft restoration.

The maintainer subsequently allowed the restored-file prompt, and `input.md`
saved successfully. A separate disposable README copy was opened and edited
to exercise a new write-consent request. The maintainer canceled that request;
the UI reported `Save failed: Write permission was not granted.` while
retaining the 783-character dirty draft. The on-disk copy's SHA-256 still
matched the source example README. This is observed native denied-write
protection, not an API double.

The second bounded session also reached its deadline. A third launch restored
the denied-write draft and its probe unchanged. Automated Save and Save as
clicks did not visibly advance the UI in that session; successful saving after
denial remains unverified. Automated title-bar and keyboard close attempts
also produced no visible change, while Windows reported the host responding;
input delivery versus application behavior was not isolated. The third session
also reached its deadline. These timeouts are harness cleanup events, not
evidence of a runtime crash or graceful lifecycle success.

### Input Delivery Investigation: 2026-09-11

A ninety-second passive DOM probe ran against a copy of the denied-write
test profile with the same public alpha.51 host. All eighteen CDP samples
returned successfully, without evaluation exceptions; the document command
buttons remained enabled. The native automation tool was asked to click
Save as, but the captured trusted pointer/mouse/click sequence targeted
`HEADER`, not `save-as-document`. No save-button click was recorded. Focus
also changed during observation, and a later screenshot was occluded by
another foreground application; no input was sent to that application.

This isolates the observed attempt to input targeting before the save
handler, not a reproduced save-handler failure. It does not establish the
underlying coordinate/focus cause or prove that every earlier attempt failed
for the same reason. Keyboard and manual-click comparison remain unverified.
No runtime fix or permission reset was applied. The temporary probe was
removed after collecting evidence and its process tree was stopped. A real
post-denial save remains required before completing that readiness check.

### Manual Post-Denial Save: 2026-09-11

The maintainer reopened the existing isolated alpha.51 session and confirmed
saving after being asked to use Save as. The new disposable file
`recovered-after-denial.md` contained exactly the restored 783-character probe
including `NATIVE-DENIAL-PROBE`, verified against the source example text with
the observed insertion and normalized editor line endings. Its 783 bytes had
SHA-256 `a165087afffcb95c18a73e2bb3b8330dac9e8b4e2c01f69fff739a4d524b850b`.
The session exited with code 0 before its deadline; it was not killed by the
wrapper. No automated clicks were used in this session.

The disposable original README also contained the same edited text after
this manual session. Its exact save interaction was not separately observed,
so this record claims the verified new-file recovery path, not a particular
same-file permission regrant sequence. The prior unchanged-original check
applies to the earlier denial instant, not the later manual save session.
The tracked example was unchanged. This closes the pending post-denial Save
as check without claiming the input automation coordinate issue is fixed.
Current-artifact lifecycle stress and repeated reload evidence remain separate
beta requirements; no runtime change or new release is included.

### Public Host DPI Diagnosis: 2026-09-12

A fresh extraction of the checksum-verified public alpha.51 ZIP was launched
with a disposable File Notes project and profile. Read-only Windows queries
returned process awareness 0 and window awareness 0 (DPI unaware), window DPI
96, and monitor scale 125 percent. All queried HRESULTs succeeded. The host
SHA-256 was `2b1da59f1ba61f5aac554a9ee3dc272a6edf42991f628015ddbb56bd275cd0b9`.
The probe stopped only its own process tree after collecting the result; no
display, compatibility or permission setting was changed.

This confirms that the public host is DPI unaware on the tested scaled
display. Windows can bitmap-scale such windows, making this a supported
explanation for the reported blurred text, rather than evidence that the
16px editor font itself is too small. See Microsoft's
[high-DPI guidance](https://learn.microsoft.com/en-us/windows/win32/hidpi/high-dpi-desktop-application-development-on-windows).
Per-monitor awareness, window resizing on DPI changes and WebView bounds need
a separate implementation and verification unit. No font or runtime change
was made in this diagnostic unit. Visual before/after comparison, mixed-DPI
monitor transitions and any relationship to the earlier automation coordinate
mismatch remain unverified; no Tauri/Wails parity claim is made.

### Per-Monitor DPI Candidate: 2026-09-12

The local alpha.52 candidate configures per-monitor V2 awareness before host
window creation, falling back to V1 only when the V2 context is unsupported.
An incompatible preconfigured awareness mode fails explicitly. Logical
96-DPI window dimensions and size limits are scaled; WM_DPICHANGED applies
the suggested physical rectangle and refreshes WebView bounds and position.
The editor font remains unchanged.

On the same 125-percent display, the local candidate reported process and
window awareness 2 (per-monitor) and window DPI 120, replacing the public
alpha.51 unaware/96 result. The candidate host SHA-256 was
`7529e3d80280dbecda8153a7a2a3aa18896010e9275a6ff0ecc32028e522d227`.
Focused host/CLI/build/inspect/runner/hygiene tests and the fork window/COM
suite passed, including 100/125/150/200-percent size arithmetic and a native
window test of suggested DPI-change bounds with a fake browser adapter.

The native window test sends a synthetic DPI-change message; it does not
prove actual cross-monitor WebView rasterization or visual sharpness. Real
mixed-DPI monitor movement, Server 2016 fallback execution and a visual
before/after comparison remain unverified. The candidate is local only;
public alpha.51 is unchanged, no release was published, and beta stays held.

### Alpha.52 Lifecycle Recheck: 2026-09-12

Local checks at source `3295d573017e978607472c86c0d2988a87b8b650`, using
Windows amd64, Go 1.26.4 and WebView2 152.0.4191.66, did not pass the
lifecycle gate:

- `velox_native_cancellation_test`: four of six source-fork cases passed.
  Controller-pending repetitions 2 and 3 failed with `late controller browser
  did not exit` at the ten-second process-exit boundary. Callback references
  had drained and destroyed-state assertions had passed; those cases never
  reached the profile-release assertion. The environment-completion cases
  and controller-pending repetition 1 passed. This test does not run the
  production host's DPI initialization, so it does not establish a DPI
  regression. A subsequent process inventory found no matching test browser
  still running; that does not turn the deadline failures into passes.
- `velox_build` passed. The rebuilt alpha.52 host SHA-256 was
  `cc44af54ecf500415e2f7ee7136664b58d802c3eaf29ac133a2937870d7ed24c`.
  This differs from the earlier local ZIP host above; the following result
  binds to this source rebuild, not the earlier ZIP or a public download.
- `velox_design_lifecycle_test`: nine of ten fresh/immediate same-profile
  pairs passed. Sample 0 failed at `immediate-launch` with `HOST_RUN_FAILED`;
  its first host reached ready and exited, but no immediate-launch result was
  retained. The harness drops the underlying run error at this boundary,
  leaving the specific failure cause unresolved. Successful samples observed
  both browser exits and profile removal. Raw v3 evidence covers
  07:49:38-07:52:07 UTC; no hosted-run claim follows from this local check.

Preserve these failures rather than increasing deadlines or replacing them
with unchanged passing retries. The diagnostic follow-up now logs each failed
sample's phase, stable error code and underlying cause in the Go test log;
the v3 JSON shape remains unchanged. Known workspace, temporary and user-profile
roots are masked, messages are quoted and limited to 4,096 bytes. Host failure
messages include the observed exit code (or `unavailable`), and distinguish a
requested test cleanup kill from a natural nonzero exit. Native cancellation
logs include browser PID, exit observation, elapsed wait, poll count, final
Windows wait status/error and callback references before asserting failure.
Existing timeouts and pass conditions remain unchanged. Focused diagnostics
tests cover a failed launch, exit code 6, bounded/masked messages and browser
wait timeout/error/success classification; this does not resolve or supersede
the observed native failures. Next isolate browser-exit delay and the detailed
immediate-launch error before deciding whether runtime, fixture or environment
changes are needed. Publication of alpha.52 remains held;
repeated reload, visual comparison and mixed-monitor checks were not run in
this unit. Public alpha.51 and the beta hold are unchanged.

### Native Cancellation Diagnostic Rerun: 2026-09-12

One six-case run at source `4213851c087e5e81852dd3b810e3ba9357a07934`
passed with the same WebView2 152.0.4191.66, Go 1.26.4 and Windows amd64.
The three controller-pending browser waits observed exit after 1,381, 193 and
194 ms; each ended with wait status `0x00000000`, no wait error and zero
callback references. All six cases passed profile cleanup. No matching test
browser remained in the post-run process inventory. Total native test time
was 3.10 seconds, with the existing ten-second exit deadline unchanged.

The earlier two deadline failures were not reproduced, not fixed or erased.
This single source-fork rerun does not identify their cause or establish
production-host or release-ZIP reliability. No further unchanged retries were
run. The separate immediate-relaunch failure is still unresolved; its new
diagnostics are the next bounded investigation. Publication and beta remain
held. No runtime or version change follows from this observation.

### Same-Profile Diagnostic Smoke: 2026-09-12

One `velox_startup_smoke` run at diagnostic source
`577dc1324e013b0999dca2c691d17ddca7076fd1` passed using the unchanged local
alpha.52 rebuilt host SHA-256
`cc44af54ecf500415e2f7ee7136664b58d802c3eaf29ac133a2937870d7ed24c`.
First startup reached ready in 772.8 ms and host exit in 115.4 ms. Immediate
same-profile startup reached ready in 1,659.5 ms and host exit in 94.0 ms;
browser exit and profile removal were observed after about 6.65 seconds.
The included security-policy and missing-runtime checks also passed. The
complete smoke took 17.73 seconds; no deadlines or pass criteria changed.

The earlier immediate-launch failure was not reproduced, and this one-pair
pass does not identify or fix its cause. It also does not validate the earlier
release ZIP, File Notes interaction or development reload. Those artifact and
workflow checks remain separate. No additional unchanged reruns or runtime
changes were made; alpha.52 publication and beta remain held.

### Pinned Alpha.52 ZIP Lifecycle: 2026-09-12

Three fresh/immediate same-profile pairs passed using the host extracted into
a new isolated cache directory from the unchanged local candidate ZIP:
3,605,419 bytes, SHA-256
`b2cb6b44152a4da5c38ea4925deb53ce4ec4b016418228062b69a88bfce0957c`.
The extracted host hash was
`7529e3d80280dbecda8153a7a2a3aa18896010e9275a6ff0ecc32028e522d227`.
Release identity and every manifest-listed artifact's size and digest were
checked before execution; ZIP and host digests were unchanged afterward.
No host rebuild or public download was used.

Harness source `0fc60070f0f3b63811cd9c98c64c5ea67e8ede0d` used the source
hello fixture and WebView2 152.0.4191.66 on local Windows amd64. Raw v3
evidence covers 14:18:03-14:18:47 UTC, SHA-256
`b2311d4b8b3c804523b7fad01a8345939f8652d409c1efde3110b7a004b4c39c`.
All six hosts reached ready and exited; both browser exits and profile
removal were observed for every pair. Fresh startup took 574-638 ms,
immediate startup 7.07-7.21 seconds, host exit 67-170 ms, and profile release
6.42-7.09 seconds. Total native test time was 43.93 seconds.

The roughly seven-second immediate-startup delay remains a usability concern;
passing the bounded lifecycle check does not establish performance parity or
identify the cause of that delay. Earlier intermittent failures remain in the
record. This is local extracted-host evidence, not File Notes interaction,
CLI launch, development reload, hosted stress or public-download evidence.
No additional retries, runtime edits or publication were performed. Reload,
visual checks and the release decision remain pending; beta stays held.

### Alpha.52 Normal Reload Failure: 2026-09-13

A private File Notes copy and fresh profile were launched through the local
alpha.52 `velox run --debug` CLI. The verified CLI SHA-256 was
`3b48f3d37d2baf2f476124514c480421266a59ee6f35f25fb7f418a82e1471bf`;
the host was the same `7529e3d8...522d227` artifact recorded above.
The probe added separate HTML, CSS and JavaScript markers only to the copy.

The first ordinary CDP `Page.reload` with `ignoreCache: false` failed:
HTML changed from `before` to `after-one`, while JavaScript stayed `before`
and computed CSS stayed `rgb(200, 10, 20)` instead of `rgb(10, 120, 30)`.
The virtual origin remained unchanged. The planned second change was not
attempted after this failure, and no hard reload or cache-disabling override
was used to turn it into a pass. The owned process tree was stopped
successfully. The result covers 06:12:19-06:12:26 UTC and is retained under
the ignored `reload52-1789279939175` evidence directory.

Source inspection shows `internal/webview2/runtime_windows.go` using virtual
host folder mapping without a separate debug cache policy. Cache behavior is
a candidate explanation, not yet a proven root cause; conditional requests,
file timestamps and development-mode cache handling need a targeted comparison.
The next implementation decision must preserve production asset serving and
origin/security boundaries. No runtime edit was made in this diagnostic unit.
Normal development reload is now a confirmed failing workflow for this run;
earlier successful checks do not supersede it. Alpha.52 publication and beta
remain held, and visual confirmation remains pending.

### Reload Cache Comparison: 2026-09-13

One bounded comparison used the same hash-verified alpha.52 CLI and host,
a copied File Notes project and a fresh private profile. Two separate CSS/JS
pairs changed from equal-length `base0` content to `next1` content: pair A
kept its original filesystem modification time, while pair B advanced by
exactly 2,000 ms. No system clock, source example or production setting changed.

After ordinary reload, HTML was current but both pairs still showed old JS
and CSS. All four resource requests emitted CDP `requestServedFromCache`,
reported status 200, `fromDiskCache: false` and `fromServiceWorker: false`,
and retained the original Last-Modified value. Captured cache-related request
headers contained no validators. A following `ignoreCache: true` reload of
the same unchanged files updated both pairs; no response was marked as a
disk/service-worker cache response, and pair B returned the advanced
Last-Modified value. No Cache-Control or Expires header was present in the
captured response headers. The virtual origin stayed unchanged.

This comparison localizes the stale-resource behavior to browser cache reuse
on normal reload, rather than a failure to write or read the edited files.
Advancing timestamps alone did not prevent it, so same-second modification
time precision is not a sufficient explanation. The next fix should control
cache reuse only in explicit development mode while preserving production
virtual-host serving and origin/security boundaries. The cache-bypassed
success is diagnostic evidence, not a normal-reload pass or a product fix.

Evidence covers 14:53:56-14:53:58 UTC under the ignored
`reload-cache-compare-1789311236574` directory; result SHA-256 is
`14cb4c764b3eabd7022584eb5f64d19f44041e073ac4de964f4190dde85e8b4d`.
The owned process tree was stopped successfully and the binaries were
unchanged. No runtime modification or publication occurred. Alpha.52 and
beta remain held pending the development-mode fix and remaining visual checks.

### Development Cache Fix: 2026-09-14

Local alpha.53 requests `Network.enable` followed by
`Network.setCacheDisabled` with `cacheDisabled: true` through WebView2's
in-process protocol API when constructing an explicit development WebView.
Production mode makes neither call. Virtual-host folder mapping, permission
and navigation rules, IPC and persistent browser storage are unchanged; this
policy opens no debugging listener. Dispatch HRESULT failures abort window
initialization. The commands are asynchronous with no retained completion
callback; the native regression checks their observed effect, not just dispatch.

The first candidate, which sent only the cache policy without enabling the
Network domain, still failed the first ordinary reload. That failed result is
retained under `normal-reload-1789312659928`; it is not counted as a pass.
After enabling the domain, `scripts/dev-reload-smoke.ts` passed two consecutive
HTML/CSS/JS edits using only `Page.reload` with `ignoreCache: false`. The script
does not send a cache-disabling command or perform a hard reload. It verifies
release manifest hashes, uses a private example/profile, checks the same
origin and retains the result before stopping its owned process tree.

The successful result covers 2026-09-13 15:20:14-15:20:16 UTC under
`normal-reload-1789312814831` (2026-09-14 in Korea). The tested host SHA-256 is
`0ee10fc617554fcf17b1f1c26a047bb7167b479c20e161cdc178e76408de0490`.
Production startup, same-profile relaunch, missing-runtime handling and
security-policy smoke also passed. Immediate relaunch still took 6.92 seconds;
this patch does not claim to fix that delay or historical cancellation failures.

Focused fork/COM, runtime, version-fixture and hygiene checks cover production
no-op behavior, invalid lifecycle state, exact protocol arguments, and native
dispatch failure. Version fixtures now use alpha.53; the public alpha.51 release
is unchanged. No publication or beta promotion occurred. Mixed-DPI visual
confirmation and the release decision remain pending.

### Relaunch Profile Comparison: 2026-09-14

Reanalysis of the three retained alpha.52 ZIP lifecycle samples places the
largest relaunch interval between `environment-created` and
`controller-created`: 6,684.62-6,917.44 ms. The latter marker follows core
WebView acquisition and event/security handler setup, so this interval is not
an isolated measurement of the asynchronous controller creation call.
Navigation dispatch to DOM plus two animation frames took 165.28-294.21 ms.
The harness started the second host 7.53-25.54 ms after the first host exited;
it did not insert a seven-second prelaunch wait. The first browser exited
6,403.48-6,460.77 ms after the second process started, and readiness followed
669.68-745.58 ms later. These timings are alpha.52 observations, not alpha.53
phase measurements. The original evidence SHA-256 remains
`b2311d4b8b3c804523b7fad01a8345939f8652d409c1efde3110b7a004b4c39c`.

One bounded alpha.53 comparison then used the unchanged host SHA-256 above,
the `examples/hello` fixture, harness source
`99fa5e3cd2ad804242a965701c5fec0389681f6e`, and WebView2 `152.0.4191.66`
on local Windows amd64. `TestStartupProfileComparisonEvidence` ran one
same-profile trial followed by one fresh-profile trial, four host launches
in total, with isolated test profiles and no runtime edits.

| Observed boundary | Same profile | Fresh profile |
| --- | ---: | ---: |
| Second host start to ready | 6,903.07 ms | 547.15 ms |
| First browser exit after second host start | 6,380.15 ms | 6,430.17 ms |
| Second ready after first browser exit | 522.92 ms | -5,883.02 ms |

The fresh-profile host became ready while the previous browser was still
alive. The observed 6,355.92 ms difference supports a same-profile reuse
dependency during browser shutdown; it does not establish why WebView2
retains its browser process for about 6.4 seconds. This is one sequential
comparison, not an order-balanced benchmark or a population percentile,
despite the harness summary's P50 field names. Warm-cache and trial-order
effects remain possible. No user profile was rotated, no browser was
force-terminated as a remedy, and persistent storage behavior is unchanged.

The test passed in 24.05 seconds. Evidence covers 06:10:50-06:11:14 UTC under
the ignored `profile53-compare-e73b203ded7e4b519e0a10d8e06352e2` directory;
result SHA-256 is
`b028fba33e0e857d75e561b5e55d6f04e68d02f7d1609321f50f56df2345a5c1`.
The host hash matched before and after execution. Next diagnosis should
separate controller callback entry from setup completion and inspect the
native close/release lifecycle before selecting a storage-preserving fix.
Historical cancellation failures, mixed-DPI visual confirmation and the
release decision remain open. This record does not publish alpha.53 or
promote beta; API, DB, runtime and repository hygiene rules are unchanged.

### Native Callback Phase Isolation: 2026-09-14

The opt-in `TestNativeRelaunchPhases` diagnostic wraps the fork's existing
controller-completion implementation in test code. It timestamps callback
entry before forwarding to the unchanged implementation and observes the
existing `controller-created` marker after setup. Each initialization runs in
a separate, timeout-bounded test subprocess, using an isolated profile and
hidden native window. Browser-exit waits occur during cleanup after both
launches, not as a prerequisite for relaunch. Ordinary fork tests skip this
native diagnostic. No product callback, lifecycle order or timeline schema
changed, and no release version bump is needed for this test-only addition.

One same-profile pair followed by one fresh-profile pair passed on local
Windows amd64 with WebView2 `152.0.4191.66` and Go `go1.26.4`:

| Initialization interval | Same first | Same second | Fresh first | Fresh second |
| --- | ---: | ---: | ---: | ---: |
| Environment marker to controller callback | 286.639 ms | 234.121 ms | 330.323 ms | 369.289 ms |
| Callback entry to setup marker | 0.000 ms | 0.000 ms | 0.590 ms | 0.000 ms |
| Destroy call duration | 12.972 ms | 9.877 ms | 19.311 ms | 14.496 ms |

Zero-valued intervals mean no difference resolved by this measurement, not
zero execution cost. All four callbacks reported success exactly once;
`Destroy` cleared the three owned COM interface fields and the callback
reference registry count was zero immediately afterward. Recorded shutdown
order was event removal, controller close, WebView release, controller
release, then environment release. Explicit close is consistent with
[Microsoft's controller lifetime guidance](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2controller#close),
but this observation does not verify every native COM reference or the close
HRESULT: the current close wrapper does not inspect that HRESULT and Destroy
discards its returned error. There is no evidence here that close failed.

The roughly seven-second release-host delay did not reproduce in this
initialization-only experiment. This test does not navigate, configure the
full product security policy, wait for DOM readiness, or use the release
executable. It also explicitly initializes and uninitializes its test STA,
unlike the product path's package-initialization lifetime. These differences
prevent transferring the fast setup timings or zero callback references to
the failing full-app path. The earlier alpha.53 same-profile result remains
unresolved; neither a generic COM leak nor an unavoidable WebView2 delay is
established. The next useful experiment must retain the production navigation,
settings and shutdown path while observing the same callback boundaries,
rather than repeat the minimal test or rotate user profiles.

The native comparison passed in 5.82 seconds and fork unit regressions passed.
Evidence is retained under the ignored
`native-phase-2c82cc79b3cc4c5891beebddd395c91c` directory, bound to base commit
`ab69cc70f1f639402d5f563eaa8e973954583f7c` and diagnostic source SHA-256
`cf54a8633b2bdf202e76c2f9a6cf35e15b9440d1df2e5cfd83c5f652967e6fe0`.
The native log SHA-256 is
`3fb22e75047cf00b5315a372c5b8412206107bac2760a4fe03fbfb4cbca4480a`.
No production binary was rebuilt or published. API, DB, runner selection,
persistent data and repository hygiene rules are unchanged. Full product
regression, historical cancellation failures and visual DPI checks were not
rerun in this bounded diagnostic unit.

### Full Host Controller Timing: 2026-09-14

`scripts/relaunch-host-phases.ts` builds a diagnostic-only Go overlay of the
real host, then runs one fresh/immediate same-profile pair through the existing
native lifecycle harness. The `examples/hello` navigation, production settings,
security policy, DOM-plus-two-frames ready boundary, dispatch draining and
shutdown sequence remain in place. The overlay adds controller callback-entry
timing, captures the HRESULT of the same controller Close call, and reads the
callback registry count after owner release. It neither reorders releases nor
adds another Close call or a browser-termination workaround.

Source files are hash-checked before and after execution. Instrumented startup,
shutdown and lifecycle records use separate diagnostic schema identifiers;
they are not submitted as ordinary v3 release evidence. Generated overlay files,
the host and results remain under one private cache directory. This does not
modify the production source, public diagnostic schemas or release version.

The delay reproduced on local Windows amd64, Go `go1.26.4`, WebView2
`152.0.4191.66`, with base source `70f4a7d3b7183664df636bb25653154f75a050d8`:

| Observed interval | First launch | Immediate relaunch |
| --- | ---: | ---: |
| Process start to ready | 1,697.16 ms | 6,358.18 ms |
| Environment marker to controller callback entry | 312.35 ms | 6,127.45 ms |
| Controller callback entry to setup marker | 2.21 ms | 0.00 ms |
| Navigation dispatch to DOM plus two frames | 1,296.52 ms | 190.41 ms |
| Controller Close HRESULT | 0x00000000 | 0x00000000 |
| Callback references after owner release | 0 | 0 |

The second process started 3.66 ms after the first host exited. The first
browser exited 5,840.93 ms after that second start, and second readiness
followed 517.25 ms later. Roughly 96% of second-startup time occurred before
the controller callback arrived, not in the subsequent settings work or page
rendering. Zero-valued timing means no difference resolved by this sample.

Decision: retain this as an unresolved same-profile relaunch latency limitation
while continuing the other readiness work. The evidence does not justify a
release-order change, profile rotation or forced browser termination. Close
failure and retained Go callback ownership were not observed in this run;
the reason for delayed native controller completion is still unproven.
This is not a finding that WebView2 universally requires six seconds, nor a
proof that every native reference is released. The separately identified
Close HRESULT reporting gap remains a worthwhile bounded correctness fix,
but must not be presented as a latency fix without new evidence.

The lifecycle pair passed in 14.38 seconds. Evidence covers 07:12:47-07:13:06
UTC under `host-phases-2605147f-b0a8-4ec8-9946-159d63787146`. Diagnostic host
SHA-256: `fd34b396481ba6755c35754346c7b9b9f19df86a3bc239bd91f543f323b94e35`;
raw evidence SHA-256:
`1c690289fa7456e73f4ef665ad9647295c1b28062d7fd3c2a3371d0f15cccf14`;
result SHA-256:
`a92a5553ff5e52890ee474e0fabe6f9fd0129eaacb3e258e2cc3e697d2fcd85f`.
This source-host observation does not replace a release-ZIP test, hosted
verification or the outstanding cancellation and mixed-DPI checks. No
publication, beta promotion, API, DB, persistent-data or runner change occurred.

### Controller Close Error Handling: 2026-09-14

Local alpha.54 checks the COM HRESULT returned by controller Close rather
than the unrelated Windows last-error value. Both ordinary Destroy and late
controller completion report a failure to stderr and emit
`controller-close-failed` instead of claiming `controller-closed`. Destroy
still releases the WebView, controller, environment and callback owner after
a failed Close; it does not retry, change ownership of a borrowed late
controller, or force browser termination. Public method signatures and the
successful shutdown timeline are unchanged.

The native startup harness rejects a recorded close failure even when the
process exits successfully and subsequent reference cleanup completes. Its
existing lifecycle failure record and diagnostic output retain the failure;
such a run cannot pass as ordinary lifecycle evidence. The diagnostic overlay
script's exact source anchor now follows the shared close helper and retains
the same HRESULT semantics. Its TypeScript bundle check passed, but another
overlay latency experiment was intentionally not run for this correctness fix.

Fork regressions passed for success, stale last-error, nonzero successful
HRESULT, failing HRESULTs, cleanup continuation, repeated Destroy and failed
late borrowed-controller close. Three shutdown-gate cases passed. Related
version, benchmark-recorder and hygiene tests, all 39 offline evaluation-tool
tests and the diagnostic tooling build passed. The evaluation fixtures require
no model calls or live Hermes session. Local version fixtures are synchronized
to alpha.54; public alpha.51 release records are unchanged.

The rebuilt source host passed fresh/immediate startup, security policy and
profile release in 23.77 seconds. Immediate readiness was 7.21 seconds, so the
known relaunch delay remains unresolved and this patch makes no latency claim.
The native run exercises successful real Close calls; the failing HRESULTs
are injected unit-test evidence, not a reproduced WebView2 close failure.
Historical initialization-cancellation and mixed-DPI visual checks were not
rerun. No release ZIP was assembled, published or promoted to beta. API, DB,
persistent storage, runner selection and repository hygiene rules are unchanged.

The ordered alpha.54 cancellation, DPI and candidate follow-up is tracked in
[Alpha.54 readiness](alpha54-readiness.md), with separate evidence levels.

## Optional Evidence

AI trials and external user feedback can reveal documentation or product
problems. Neither model identity nor session-log collection is a required
product dependency. Existing evaluator verdicts retain their historical ADR
0018 meaning. Human-adoption claims remain false unless separately supported.

Beta remains held; publishing another unsigned alpha does not promote it.
