# Validation

- Status: M4 complete; M5 narrow alpha active; beta gated by product workflows with no human adoption claim

## Validation Source of Truth

This document owns stable validation names for this scaffold.

## Standard Validation Names

Folder-browser starter (source-only, 2026-10-05): scoped initializer/CLI tests
cover generation, both native templates' flag forms, exact permissions,
escaped names, asset inventory and conflict preservation. The folder starter's
six Bun cases plus the text editor's six cases passed, covering literal text,
non-navigable directories, cancellation, refresh/release, pending guards,
expired/unsupported/denied reads and absent-bridge behavior. Edge mock checks
passed at 960/360 widths in light/dark modes with loaded icons, keyboard folder
selection and independently usable panes without horizontal overflow.
CommandCode DeepSeek 4.1 Flash/high supplied the checked design/doc drafts.
The generated project had 7 web assets / 12,667 bytes; local matching-bundle
init/validate/build/inspect passed. The inspected host digest remained
`ae5cd1bdd78b19743aebdfa80bb15e91e3f7b01b992e02aab6414e29eb0b151d`.
Native folder dialogs/reads, installation, hosted CI and performance tests
were not repeated: native implementation and dependencies are unchanged.
These are mock/local packaging results, not a new public release.
No IPC/schema/DB/host/dependency/workflow or version change is included.

Text-editor starter (source-only, 2026-10-05):
`go test ./internal/initializer ./internal/cli ./tests/hygiene` and scoped vet
passed. `bun test internal/initializer/text_editor.test.ts` passed six cases
covering save-target reuse, cancellation/error preservation, discard/open/new,
pending-operation guards, IME/shortcuts and absent-bridge behavior. Edge mock
checks passed at 960/360 widths in light/dark modes with long names, loaded
icons and no horizontal overflow. CommandCode DeepSeek 4.1 Flash/high supplied
the visual specification and documentation draft, checked against implementation.
The final generated project had 8 web assets / 13,697 bytes; portable ZIP and
Setup packaging passed using unchanged public alpha.64 host bytes. Native CDP
confirmed the bridge and enabled Open control, then a synthetic ready marker
closed the host with exit 0. This is not a native file-dialog/save test.
Earlier exit tests lacked the template's ready signal, and a native screenshot
timed out; those attempts were cleaned up, not counted as passes. Final visual
evidence is mock-browser only. Actual file selection/write, installation,
hosted CI, performance benchmarking and publication were not repeated.
No IPC/schema/DB/host/dependency/workflow or version change is included.

Portable consumer summary: `go test ./cmd/velox-consumer-summary` covers schema
validation, nested/single-file input, duplicate IDs, nearest-rank statistics,
missing/excess/failed samples, mixed release digests, hosted process-evidence
requirements, and input preservation. It needs Go but not Windows or
PowerShell. `go vet ./cmd/velox-consumer-summary` checks this development tool.
The workflow keeps aggregation separate from the compiler-free Windows
consumer job. On 2026-10-05, Windows tests and Linux amd64 test cross-compilation
passed; local Linux execution was unavailable because WSL registration failed.
An external Debian 13 amd64 / Go 1.26.7 execution receipt was subsequently
reviewed: root-module tests, vet and summary CLI checks passed without source
changes. Source/archive hashes, command exits and detailed test counts matched.
See [the Linux receipt record](docs/ops/linux-common-go-20261005.md) for the two
existing runtime skips and excluded Windows/nested-module coverage. No local
rerun was needed to inspect this evidence.

TypeScript bridge declarations: `tsc --noEmit -p tests/types/tsconfig.json` checks
all public method calls and save helpers, inferred responses, cancellation
narrowing, optional/readonly browser globals, invalid names/params and correlated
method/parameter pairs. `go test ./tests/hygiene -run '^TestTypeScriptMethodCoverage$'`
compares the declaration method keys with the native dispatcher. These checks
emit no application JavaScript and do not replace native permission or value
validation. Runtime-only checks need not be rerun for declaration-only changes.

TypeScript delivery: scoped initializer/release-bundle/CLI tests check root
declaration identity, editor reference, unchanged permissions and web asset
boundary, conflict preservation, required type-file failure cleanup, ZIP
contents, deterministic archives and artifact hashes. Type-check the shipped
example and a generated JavaScript project with `checkJs`. Reuse an unchanged
host for a matching local bundle's init/validate/build/inspect path; host UI and
performance tests are not required when the host dependency tree is unchanged.

Taskbar progress requirements: scoped IPC/manifest/runtime-config/build-report/
builder/WebView2/host/hygiene checks and related vet when implementation is ready.
Cover independent permission/default denial, strict state/value grammar
(including unknown fields, fractions and null), packaged propagation and
redacted errors. Verify BeforeShow registration before initial display,
latest-state caching with no ITaskbarList3 calls before TaskbarButtonCreated,
non-none lazy COM/HrInit, initialization failure release, value-before-state,
successful-repeat deduplication, Explorer button recreation, and best-effort
clear/exactly-once release under synchronous reentry and destruction.
Permission-absent windows must install nothing and make no progress COM calls.
Visible taskbar progress and Explorer restart recovery need separate native
evidence; size and operation counts require measured results. These are
requirements, with no completed validation claimed by this entry.

Tray notification requirements: scoped IPC/manifest/runtime-config/host/hygiene
checks when implementation is ready. Cover independent default-off
`notification.show`, strict two-field grammar (unknown fields, missing/wrong
types, kind values, whitespace-only and oversized messages, NUL, DEL and C0/C1
controls other than LF and TAB), packaged propagation and redacted errors.
Verify `window.tray: true` plus a registered icon is required and that a
missing or disabled tray installs nothing and returns `NATIVE_OPERATION_FAILED`,
fixed manifest-title truncation to 63 UTF-16 units, `NIM_MODIFY` with a
transient `NIF_INFO | NIF_REALTIME` copy, no stored body or Explorer-restart
replay, and `NIN_BALLOONUSERCLICK` reveal through normal modal/shutdown gates.
Actual balloon visibility and Windows quiet-time or disabled-notification
behavior remain shell-controlled manual evidence. These are requirements, with
no completed validation claimed by this entry.

Tray notification example requirements: the isolated `examples/tray-notification`
app must keep native work behind an explicit form submit. Its Bun interaction
checks must cover kinds `info`/`warning`/`error`, the 255-UTF-16 boundary
(empty, whitespace-only, oversized, and disallowed NUL/DEL/C0/C1 controls other
than LF and TAB), a local message counter with no native call on input or kind
change, pending-submit duplicate blocking, kind/message preservation with focus
return, redacted failure codes and the missing-bridge status. Browser checks
must cover keyboard submit, exact `Kind`/`Message` field names, light/dark and
both forced-color modes, 620 x 480 and 320 x 480 with no horizontal clipping,
the current-color bell mask, no page errors and no IPC on a live theme switch.
Visible Windows balloon display, suppression and click-to-restore remain
separate manual evidence. These are requirements, with no completed validation
claimed by this entry.

Activation shortcut requirements: scoped manifest/runtime-config/WebView2/host/
hygiene checks and related vet when implementation is ready. Cover the strict
grammar `Ctrl+Alt+[Shift+]<one A-Z or 0-9>` including rejected null and
non-string values, modifier order, spaces, lowercase, duplicates, `Win`,
`F12`, multibyte and empty keys. Verify one `RegisterHotKey` binding with
`MOD_NOREPEAT` on the existing HWND, id/modifier/key validation before reveal,
no native registration or subclass when the field is omitted or empty,
mock-message reveal of hidden/minimized windows with maximized placement
preserved, rollback that never unregisters an unowned binding on a conflicting
or failed registration, and exactly-once `UnregisterHotKey` across
`WM_DESTROY`/`WM_NCDESTROY`. A conflict must warn once and leave the app
running without the shortcut and without retries. Physical key presses
revealing hidden, minimized, and background windows remain separate manual
evidence. These are requirements, with no completed validation claimed by this
entry.

Activation shortcut example requirements: the isolated
`examples/window-activation-shortcut` app must set `window.activationShortcut` with
`window.tray: true` and no native permissions, and must not invoke IPC or show the
shortcut, tray menu or any how-to text in the page. Its Bun interaction checks
must cover a local UTF-16 counter from the initial `Note` value, a `Note changed.`
status on input with no native call, a `Window focused.` status that preserves the
note across repeated focus, and the 2048 `maxlength` boundary with the `Ready.`
initial status. Browser checks must cover the exact `Note` textbox label,
620 x 480 and 320 x 480 with no horizontal clipping, light/dark and both
forced-color modes, text and counter preserved across a live theme switch, no
page errors and no IPC. A physical key press revealing a hidden, minimized or
background window and the real conflict warning remain separate manual evidence.
These are requirements, with no completed validation claimed by this entry.

Fixed-size windows: scoped manifest/runtime-config/WebView2/builder/host/hygiene
Go tests and go vet. Cover omission/true/false, invalid types, default style,
native frame/system commands, IPC maximize rejection, no initial geometry or
visibility/focus change, saved-position-only restoration, ignored maximization,
DPI/work-area fitting and destruction. Physical frame dragging, snapping and
multi-monitor DPI changes remain manual evidence.

System theme checks: `go test ./internal/manifest ./internal/runtimeconfig
./internal/webview2 ./internal/builder ./cmd/velox-host ./tests/hygiene` and
scoped `go vet`. Cover false/default omission, true propagation, invalid
types, older or unsupported-build fallback to the default title bar, light/dark
`AppsUseLightTheme` reading, a missing preference reading as light,
high-contrast priority, error retention of the last successful appearance and
deduplicated native writes. Confirm no script method or permission, no
persistent state and no new dependency. Live user-driven Windows theme changes
and visual title bar comparison remain manual evidence.

Always-on-top checks: `go test ./internal/manifest ./internal/runtimeconfig
./internal/webview2 ./internal/builder ./cmd/velox-host ./tests/hygiene` and
scoped `go vet`. Cover boolean/default propagation, initial native topmost,
system-menu toggle/check synchronization, unchanged geometry/visibility/focus,
invalid handles, destruction and non-persisted defaults. Visible overlap and
user-driven system-menu/dialog interaction remain manual evidence.

Minimum window dimensions: `go test ./internal/windowlimits ./internal/manifest
./internal/runtimeconfig ./internal/webview2 ./internal/builder ./cmd/velox-host`.
Check opt-out/one-axis behavior, configuration bounds/round-trip, 96/120/144 DPI
and small work areas, native tracking and normal restore, DPI suggested rects,
maximum geometry, no activation and subclass cleanup. Physical screen dragging
and visually shrinking a packaged window remain manual checks.

Window attention checks use the same scoped Go packages as title checks below.
Cover independent permission propagation, strict optional count 1..5/default 3,
empty cancel parameters, shutdown, redacted errors, FLASHWINFO ABI layout and
bounded taskbar-only flags. Disposable native window checks must not activate
or show a window. Visible taskbar blinking/cancellation remains a manual check.

Dynamic window title checks: `go test ./internal/ipc ./internal/webview2
./internal/manifest ./internal/runtimeconfig ./internal/buildreport ./internal/builder`.
Cover independent permission propagation, strict parameters, UTF-8/control
limits, unchanged app identity, shutdown and native caption readback. Native
tests use a disposable hidden window. File Notes tests verify title transitions
and deduplication; packaged app interaction remains separately recorded.

- format
- lint
- typecheck
- test
- contract
- migration-check
- smoke
- docs
- check

## Required Final Report

Final responses must list executed validations, passed validations, skipped validations, skip reasons, and remaining risk.

## Runner Policy

Task runner files are optional. This repository still uses runner `none`.
The parent workspace command contract currently provides these bounded intents:

- `go fmt ./...` maps to format.
- `velox_lint` maps to lint.
- `velox_test` maps to test.
- `velox_build` maps to the production Go host build with `-H windowsgui`.
  Packaged apps do not allocate a console; the separate Velox CLI retains its
  console subsystem and redirected diagnostics. Native startup smoke checks the
  built host's PE subsystem as well as startup, shutdown, and failure reporting.
- Clipboard read checks cover independent opt-in permission propagation,
  deferred single completion, default denial and strict empty parameters,
  per-request approval/cancellation, pending-request rejection, navigation and
  shutdown without text disclosure, UTF-16 validation and UTF-8 byte limits,
  bounded memory copying, lock cleanup and redacted native errors. Windows
  allocation readback tests use owned temporary memory without accessing the
  user's clipboard. Actual native approval and paste require manual evidence.
- `velox_release_bundle` builds the Go CLI and host and assembles the unsigned,
  deterministic Windows x64 release bundle.
- `velox_alpha_evidence_smoke` verifies the release manifest and emits local
  checksum, SPDX, and unsigned provenance evidence for that bundle.
- `velox_signing_record_smoke` runs the deterministic signing-input packager,
  repository-owned signing-record package, and maintainer CLI tests; emits a
  non-publishable dry-run record; validates it against
  `velox.signing-record/v1`; and proves `publishable: true` is rejected for
  dry-run evidence. The Go test suite also exercises the fail-closed
  Authenticode policy boundary and `velox.authenticode-verification/v1`; a real
  signed-provider success remains a deferred future-channel gate rather than an
  M4 requirement.
- `velox_signpath_onboarding_smoke` verifies the repository-owned SignPath
  artifact configuration, GitHub source policy, dual-license files,
  CODEOWNERS, security policy, privacy policy, and application handoff packet.
- `velox_consumer_build_smoke` invokes only the assembled release CLI, creates
  a dependency-free starter, diagnoses its platform, WebView2, project, and
  bundled-host compatibility, builds it twice, checks
  byte-identical archive hashes, and inspects both the portable directory and
  ZIP.
- `velox_cli_run_smoke` launches source assets through the assembled release
  CLI, requires the host to reach its ready callback, exits it, and verifies the
  temporary runtime configuration was removed.
- `pwsh -NoProfile -NonInteractive -File scripts/measure-consumer-build.ps1 -Cli dist/release/velox-windows-x64/velox.exe -WorkRoot .cache/consumer-benchmark-smoke -ResultPath .cache/consumer-benchmark-smoke/latest.json -Repetitions 3` runs three local samples to validate the
  benchmark harness and schema without turning unavailable process tracing into
  a false pass.
- `pwsh -NoProfile -NonInteractive -File scripts/measure-consumer-build.ps1 -Cli dist/release/velox-windows-x64/velox.exe -WorkRoot .cache/consumer-asset-benchmark-smoke -ResultPath .cache/consumer-asset-benchmark-smoke/latest.json -Repetitions 3 -FixtureKind asset-pack` runs three local samples with the
  pinned 1,000-file, exact-10-MiB asset-pack fixture to expose archive and
  filesystem scaling regressions without making hosted comparison claims.
- `pwsh -NoProfile -NonInteractive -File scripts/measure-consumer-build.ps1 -Cli dist/release/velox-windows-x64/velox.exe -WorkRoot .cache/consumer-benchmark -ResultPath .cache/consumer-benchmark/latest.json -Repetitions 10 -Enforce` runs ten local clean-output samples and enforces
  build-duration, cache, intermediate-file, and compiler/package-manager
  child-process gates. It is expected to fail when Windows process-start
  tracing is unavailable.
- `velox_consumer_e2e_smoke` validates release extraction, initialization,
  build, inspection, success/failure result serialization, and the end-to-end
  JSON Schema using a local release ZIP. Its result is not hosted cold-build
  evidence. Child-process tracing may remain `unverified` locally.
- `velox_consumer_e2e_failure_smoke` injects a release-checksum mismatch and
  requires a schema-valid `release-verification` failure result.
- `go run ./cmd/velox-consumer-summary --results-root .cache/consumer-e2e-smoke/latest.json --output .cache/consumer-e2e-summary-smoke/latest.json --expected-samples 1` aggregates one local raw result and
  validates the summary schema without promoting it to hosted evidence.
- `velox_consumer_e2e_summary_failure_smoke` aggregates one success and one
  injected failure, requires the summary command to fail, and verifies the
  failed sample remains in the written summary.
- `velox_consumer_e2e_hosted_gate_smoke` simulates bounded hosted metadata and
  requires unavailable process tracing to preserve a raw result while failing
  the hosted evidence gate.
- `velox_consumer_e2e_hosted_summary_gate_smoke` requires an unverified hosted
  process trace to remain counted and fail the aggregate summary gate.
- `yq eval-all . .github/ISSUE_TEMPLATE/external-user-attempt.yml .github/workflows/alpha-evidence.yml .github/workflows/consumer-evidence.yml .github/workflows/public-preview-verification.yml .github/workflows/actions-warning-monitor.yml` parses the repository-owned GitHub Actions workflow
  with `yq` without modifying it.
- `velox_startup_smoke` maps to smoke.
- `bun test examples/deskboard/model.test.ts` exercises the functional example's persisted
  task-state normalization, mutations, filters, and derived progress without a
  browser or frontend dependency.
- `bun scripts/verify-example.ts examples/deskboard/velox.json dev.velox.deskboard deskboard` validates, diagnoses, builds twice, compares archive
  hashes, inspects, starts the packaged application directly from a non-app
  working directory, and starts `examples/deskboard` through the assembled
  Velox release. The harness is Bun/TypeScript and adds no PowerShell surface.
- `bun scripts/build-example.ts deskboard` leaves a portable Deskboard directory and ZIP under
  `dist/examples/deskboard` for manual use.
- `bun scripts/verify-example.ts examples/capability-probe/velox.json dev.velox.capabilityprobe capability-probe` validates, diagnoses, reproducibly builds,
  inspects, directly starts, and source-starts the browser capability probe.
- `bun test examples/capability-probe/model.test.ts` verifies operation-result replacement,
  rerun preservation, evidence-state summaries, and versioned report snapshots.
- `bun scripts/build-example.ts capability-probe` leaves a portable probe directory and ZIP
  under `dist/examples/capability-probe` for manual user-gesture checks.
- `bun test scripts/build-example.test.ts` verifies that the maintainer example builder can
  replace only the allowlisted `dist/examples` outputs and rejects
  arbitrary output names.
- `bun test examples/file-notes/model.test.ts` verifies draft restoration, dirty-state
  derivation, selected-file baselines, saved baselines, and Unicode statistics.
- `bun scripts/verify-example.ts examples/file-notes/velox.json dev.velox.filenotes file-notes` validates, diagnoses, reproducibly builds, inspects,
  directly starts, and source-starts the native-file editor.
- File Notes application/model/storage tests cover session-only target reuse,
  cancellation, conflict/error buffer preservation, explicit target release,
  legacy draft restoration and omission of write tokens from IndexedDB.
  File Notes 0.3.0 also tests app-local Ctrl+S / Ctrl+Shift+S / Ctrl+O / Ctrl+N
  routing through the same buttons, IME composition and key-code 229 guards,
  repeat/busy suppression and discard-dialog action preservation. Edge keyboard
  checks with mock native calls verify actual key events and dialog cancellation
  / acceptance; these are not a native WebView2 picker interaction claim.
  The maintainer later reported success after the requested actual Velox-window
  shortcut check. Product readiness records that general manual confirmation
  separately from per-shortcut, disk-readback or real IME evidence.
  File Notes 0.4.0 find tests cover literal non-overlapping Unicode matches,
  bidirectional wraparound, dense 2 MiB documents without position arrays,
  dirty/draft preservation, edited-query refresh, IME and pending-operation
  guards. Edge layout checks verify visible selection scrolling, focus return,
  icon loading and mirror cleanup at desktop/narrow widths; native WebView2
  interaction is still separate manual evidence.
- `bun scripts/build-example.ts file-notes` leaves a portable File Notes directory and ZIP under
  `dist/examples/file-notes` for manual picker and persistence checks.
- `bun test scripts/llm-agent-evaluation.test.ts` exercises clean-room trial shape checks,
  prompt and artifact digest verification, path-containment rejection,
  pass-gate consistency, failed-sequence preservation, model-diversity series
  gating, and the read-only Hermes attestation adapter's session counter,
  retry, forbidden-toolchain, maintainer-hint, workspace-escape, and exclusive
  output checks. It also exercises three-trial preparation, hash-only session
  binding, attestation, immutable summary creation, model diversity, and the
  prohibition on agent workspaces inside the Velox repository. V2 coverage
  includes prompt and isolated-state binding, sandbox staging, post-run session
  discovery, exclusive attestation, and admission only for three enforced
  receipts. The ordinary Go test intent runs the Windows AppContainer and Job
  Object adversarial test for denied outside access, contained child execution,
  state export, and ACL, profile, environment, and private-state cleanup.
- `bun scripts/llm-agent-orchestrator.ts live-smoke` reads one explicitly selected, finished
  local Hermes session through the read-only adapter and writes only a compact
  diagnostic attestation under ignored `.cache/hermes-attestation-smoke`. It is
  adapter evidence, not a qualifying beta trial or a replacement for the new
  three-session series.
- `bun scripts/llm-agent-orchestrator.ts live-diagnose` reads only completion metadata for an
  explicitly selected local Hermes session. It distinguishes stored `ended_at`
  values from a final active assistant `finish_reason=stop` without printing
  message content, tool arguments, or the raw session ID.

The manual `Consumer evidence` workflow exposes a disabled-by-default
`include_security_fuzz` action. It runs `FuzzParse` and `FuzzDispatcher`
serially with a bounded per-target duration, preserves a failing corpus for
seven days, and never runs for pull requests or release-candidate
tags. The ordinary `velox_test` intent continues to execute the fuzz seed
corpora without starting an unbounded campaign.

The hosted `Alpha release evidence` workflow builds the unsigned release twice,
requires byte-identical ZIPs, generates checksum, SPDX, and unsigned provenance
artifacts, and passes the artifact to a checkout-free consumer job. That job
invokes only `velox.exe`; it does not prove signing, authenticated provenance,
public-release download, or adoption by an external user.

An explicit manual dispatch can publish those verified files only from an
existing `vX.Y.Z-alpha.N` tag after the exact unsigned-preview confirmation is
entered. The isolated publication job alone receives `contents: write`, refuses
replacement, and creates a prerelease with SmartScreen and managed-device
warnings. Workflow validation proves this contract; it does not publish a
release.

Manual hosted [run 29806946109](https://github.com/0disoft/velox/actions/runs/29806946109)
passed for exact commit `d8495b8aa2a399505b583a8ed881b5bc7fa9f304` after the
browser-owned file workflow examples were added. The reproducible release and
checkout-free consumer jobs succeeded; publication was disabled and skipped.
ADR 0017 treats this as technical alpha evidence, not independent adoption or
permission to add an application-specific Go backend or broad native API.

ADR 0015 retains Velox as the maintainer-approved public identity and supersedes
ADR 0013's replacement-name gate. The known `velox.exe` and search collisions
remain documented risks, but the publication job may run for the exact
`0disoft/velox` repository after every ordinary unsigned-preview gate passes.

The manual `Public preview verification` workflow performs no source checkout
and downloads the ZIP, checksum, SPDX, and provenance assets from the public
GitHub Release URL. It requires an independently supplied ZIP SHA-256, binds the
tag to the release manifest and CLI version, builds twice, inspects, and reaches
the startup-ready marker. Its schema fixes `externalUserAttempt` to `false`, so
this same-repository check cannot prove independent adoption.

The first public preview is
[`v0.5.10-alpha.1`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.1)
from commit `9f10c545b6bde23d2c3dad5bbb12bffdac513712`. Tag evidence run
`29714104653`, publication run `29714173324`, and public-download verification
run `29715002921` passed. The verifier downloaded SHA-256
`5df53090e1e67ce54c8639f061ffc7b03b7c3aa38f95a725c29342cfaff73b68`,
validated the sidecar evidence, built twice, inspected the output, and reached
startup-ready without source checkout. This remains historical release
evidence, not an external-user attempt or authenticated publisher identity.

The second public preview was
[`v0.5.10-alpha.2`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.2)
from commit `9bbb6bfcc1393058cb80d72c79df601caa970f2f`. Tag evidence run
`29894943737`, publication run `29895087658`, and public-download verification
run `29895490556` passed. The public verifier observed ZIP SHA-256
`abd07aab653db7d67adf822e6a944a6f85f54c9fb0752cce367724fb0ce62fb7`,
validated checksums, SPDX, provenance, deterministic builds, doctor readiness,
inspection, and startup readiness without checkout. Its evidence level remains
`same-repository-public-download` with `externalUserAttempt: false`.

The previous public preview is
[`v0.5.10-alpha.40`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.40)
from commit `d206fe4ef1be9df198d86809742ef480549344b8`. Tag evidence
[run 34214224962](https://github.com/0disoft/velox/actions/runs/34214224962),
publication [run 34214445883](https://github.com/0disoft/velox/actions/runs/34214445883),
and public-download verification
[run 34215188131](https://github.com/0disoft/velox/actions/runs/34215188131)
passed. The expected ZIP digest was independently computed from the publication
producer's retained Actions artifact and matched the public release:
`771173b6eec2f74d92228e7ac5b52332160b0f9fb4d7baecf01976864a00f8c8`.
The public verifier checked sidecars, version, deterministic builds, doctor,
inspection, and startup without checkout. This remains
`same-repository-public-download` with `externalUserAttempt: false`, not a
qualifying LLM trial or beta promotion.

The previous public preview is `v0.5.10-alpha.49`, source
`b708cdd64fbc489729fb9ba519629fe72ee9242f`, publication run `34453796476`.
Its ZIP digest is `236e71ce0fa19bae2b2bb56c44bd1daed1083931d275b64f8a11425d9d1c59fd`.
Public-download verification run `34454305875` passed every public CLI step;
its evidence remains `same-repository-public-download` with
`externalUserAttempt: false`. The initial expected digest referred
to the separate Go 1.26.7 tag build instead of the Go 1.26.8 publication build.
The release record preserves that failure and the local startup-smoke failure
followed by an unchanged passing rerun. No beta or independent-adoption claim
follows from publication.

The current public preview is `v0.5.10-alpha.64`, source
`10202571236801451fa697ade15f0e7a799a77ad`. Tag CI `37286184342` passed
reproducible builds and checkout-free consumer packaging. Its verified four
assets were reused for publication without a second producer run.
Unauthenticated public downloads matched all checksum/manifest/SPDX digests
and provenance source/run. ZIP SHA-256:
`010478c5eea256ae1892fec5c186f61ae133db0542327677cf14f28b21744c35`
(5,876,352 bytes). The downloaded alpha.64 CLI/host passed actual native
automatic HTML/CSS/JS reload with debug off, canceled-input preservation,
subsequent retry and normal close in a copied File Notes/private-profile test.
This is automated public-byte verification, not maintainer/manual testing.
No additional hosted stress, installer execution or beta promotion occurred;
see `docs/ops/release.md` for the byte identities and scope.

The previous public preview was `v0.5.10-alpha.63`, source
`fce9955bdb355fd1b1a377dec277a60727c4ad39`. Tag CI `37224406021` passed
reproducible builds and checkout-free consumer checks; its four verified
assets were published without a second producer run. Unauthenticated public
downloads matched checksum, manifest, SBOM and provenance digests locally;
the public CLI version was alpha.63. ZIP SHA-256:
`19205e691e79dcddaeeb414cbbeb4cb055e59344d85dabfd5f7bfe5ea99b27ca`
(5,825,098 bytes). This local public-byte inspection did not repeat native
startup or UI tests. The previous alpha.62 public-download verification run
`35079337819` remains `same-repository-public-download` with
`externalUserAttempt: false`, not evidence for alpha.63 native launch.
Maintainer-confirmed saving and restart recovery
for the prior CI-built alpha.61 package are recorded in
`docs/ops/file-permission-recovery.md`. That manual check is not claimed for
the later publication bytes, nor as a human Deny-reset prompt test.
No beta promotion is claimed.

The now-archived separate public
[`0disoft/velox-consumer-smoke`](https://github.com/0disoft/velox-consumer-smoke)
repository consumed the pinned release without checking out Velox source.
Hosted [run 29736140250](https://github.com/0disoft/velox-consumer-smoke/actions/runs/29736140250)
at consumer commit `ed003602d65cbaef12bf95ee78b2cf16466bdfcd`
validated every release sidecar, all seven public CLI paths, deterministic
build output, inspection, and startup. The evidence records no consumer
toolchain command and zero Actions cache upload bytes. ADR 0016 accepts this as
the technical M4 distribution gate while requiring
`maintainerControlled: true` and `externalUserAttempt: false`; it is not
independent adoption evidence. The repository is retained read-only as the
one-shot receipt; future release verification stays in this repository.

The bounded M5 readiness records are the immutable historical
`docs/product/maintenance-cost-v1.json`, the current manual-only scheduling
snapshot `docs/product/maintenance-cost-v2.json`,
`docs/product/04-maintenance-cost-record.md`, and
`docs/engineering/08-m4-security-review.md`. Hygiene tests validate their
version, observation boundary, non-claim language, roadmap synchronization,
and the unsigned-preview security baseline. The security review remains
internal and does not replace external-user evidence.

ADR 0019 now owns channel admission through `docs/ops/product-readiness.md`.
The required checks cover public consumption, File Notes behavior, development
reload, Windows lifecycle, security and data integrity. AI evaluation is
optional. Real native picker, interactive reload and persistence checks remain
unverified; unit tests and native startup do not substitute for them.

Historically, ADR 0018 replaced the uncontrollable human-attempt beta gate with three
consecutive clean-room LLM agent trials across at least two model identifiers.
The versioned task is `evals/llm-agent/v1/task.md`; each trial must conform to
`schema/llm-agent-evaluation-v1.schema.json`, preserve failed and held outcomes,
keep `humanAdoptionClaim: false`, and match an external v2 attestation for
actual session identity, timestamps, tool counts, budget, forbidden actions,
and enforced sandbox evidence. V1 remains diagnostic-only. The v2 path is
implemented and locally tested, but no qualifying three-trial set is recorded.
`docs/QUICKSTART.md` is the public source-free discovery path for those trials;
hygiene tests reject moving release URLs, source checkout, consumer toolchain
installation, and local maintainer-copy fallbacks in that path.

The hosted `Consumer evidence` workflow additionally runs three startup
lifecycle samples for pull requests and `quick` manual dispatches. A `full`
manual dispatch or release-candidate tag runs ten lifecycle and ten consumer
samples. It has no recurring schedule trigger. It validates
`velox.startup-lifecycle/v3`, derives and validates
`velox.startup-lifecycle-summary/v1` plus
`velox.startup-lifecycle-phase-summary/v1`, and uploads all results with
`always()`. The phase summary computes interval p50 and p95 values and the
dominant immediate-startup interval directly from raw v3 evidence.
Lifecycle v3 preserves the host-local startup and shutdown phase timelines for
both the first launch and the immediate same-profile relaunch.
This longer evidence path is intentionally separate from the local one-sample
`velox_startup_smoke` intent.

An explicit manual `include_profile_comparison` input runs three alternating,
serial same-profile versus fresh-profile pairs and validates
`velox.startup-profile-comparison/v1`. It is disabled for ordinary pull-request
and release-candidate evidence.

An explicit manual `include_startup_history` run builds
`velox.startup-history/v1` from the current lifecycle summary and up to eleven
retained historical scheduled artifacts. The history is grouped by runner
image version and WebView2 version, retained for 90 days, and remains diagnostic
evidence rather than an automatic regression gate. No recurring collector is
enabled.

The `Actions warning monitor` workflow allocates a runner after
release-candidate consumer evidence or for an explicit manual run ID. Pull
request and ordinary manual consumer evidence produce only a skipped monitor
job. The monitor scans the bounded workflow-log archive for the known
`actions/download-artifact` `DEP0005 Buffer()` warning. It validates and uploads
`velox.actions-warning-monitor/v1`. Presence is diagnostic rather than a failed
product check; malformed or inaccessible log evidence still fails the monitor.
The platform-independent scanner uses the pinned `ubuntu-24.04` runner.

The C++23/Pixi M0 reference intents were retired after ADR 0005 selected Go
for both production executables. Historical comparison results remain in ADR
0004 and the performance budget.

Unconfigured validation names remain skipped and must not pass with a fake
success.

## M2 Security Evidence

| Contract | Executable evidence |
| --- | --- |
| Trusted virtual origin and top-level messages | `internal/webview2` origin tests and Windows startup security fixture |
| Navigation, frame, popup, download, and permission denial | Windows startup security fixture policy audit |
| Closed method and permission table | `internal/ipc` dispatcher tests |
| Payload, nesting, request ID, duplicate, and in-flight limits | `internal/ipc` malformed and concurrency tests |
| Frozen JavaScript bridge | embedded bridge contract test and Windows startup IPC invocation |
| Production development-tool restrictions | runtime security source guard and startup production path |
| No listening socket or broad native API | production-host source guard and closed dispatcher tests |
| Missing runtime and malformed configuration | startup and runtime-configuration failure tests |
| Path, archive, staging, and release checksum controls | asset, build-plan, builder, inspector, host metadata, and release tests |

The security fixture must complete both a trusted `app.getInfo` invocation and
the five browser-policy denials before emitting `security-ok`.

## Hygiene Validation

Repository hygiene file changes must check line-ending churn, binary diff pollution,
tracked secret files, ignored build/cache artifacts, and generated-output drift.

## Executable Branding

`velox_test` covers optional branding, ICO validation, signed-template refusal,
Windows resource loading, executable startup, code-section preservation, and
two-build archive determinism. Branded builder fixtures must pass directory
and ZIP inspection using the final executable size and SHA-256. Default builds
retain the unchanged-host test. `velox_file_notes_build` produces an example
with application-specific version resources after `velox_release_bundle`.
The installer follow-up is implemented and locally verified for beta.3; see
Windows Installer below.

## Windows Installer

The opt-in per-user Windows Setup executable is implemented in
`internal/installer`, `internal/setuppayload`, and `cmd/velox-setup`. The CLI
exposes `velox build --installer`, and `velox-release --setup` includes the
unsigned prebuilt `velox-setup.exe` template in the release bundle. Before
packaging, the CLI re-verifies the template against the adjacent
`release-manifest.json` release version, size, and SHA-256.

Unit tests cover ownership refusal, changed and unowned file refusal,
isolated-registry removal, Setup payload tamper refusal, and the
release-template check. The engine and payload unit tests passed as part of the
full `velox_test` Go suite plus `go vet`, and an installer-enabled beta.3
release bundle was built locally.

All four installer intents are locally verified for beta.3: `velox_installer_test`
(ownership and isolated-registry tests), `velox_installer_bundle` (all three
executables and an installer-enabled release ZIP), `velox_installer_smoke`
(unique app ID `dev.velox.installer-smoke-3876`, identical two-build bytes,
installed tree plus real Start Menu shortcut and registry checks, installed GUI
startup with the two-render-frame readiness marker, real helper uninstall,
preserved user test document, and script cleanup), and
`velox_file_notes_installer` (a distributable File Notes Setup, 12,361,886 bytes)
all passed locally. Final source passed `go vet` and workflow YAML parsing
passed; the release CLI smoke verified the source-free consumer compilation
boundary. This is local harness evidence, not hosted CI, a push, a release, or a
manual install. Behavior and layout are in `docs/ops/windows-installer.md`.

## Clipboard Text Write

ADR 0029 adds opt-in `clipboard.write` and only `clipboard.writeText`.
Related clipboard, IPC, manifest/runtime and host tests cover default denial,
strict text-only parameters, UTF-8/NUL/byte limits, Unicode termination,
shutdown, busy/error redaction and ownership-transfer cleanup. A real Windows
movable allocation and Unicode copy/readback test does not open or change the
system clipboard. That write-only change adds no read API, monitoring or
dependency; ADR 0030 introduces reads separately under per-request approval.
The maintainer reported successful copying in the isolated beta.19 Folder
Browser after the requested copy/paste check; this is manual evidence, not an
automated clipboard readback or independently verified Unicode coverage.
The current evidence is recorded in `docs/ops/product-readiness.md`. The initial
`go vet` pointer-conversion warning was corrected by using the native memory
copy function; targeted vet and tests then passed. Host size is measured with
matching build flags and recorded in product readiness; startup equivalence
is not inferred from code inspection or size alone.

## Clipboard Text Read Example

`examples/clipboard` tests explicit-click invocation, plain-text Unicode display,
empty text, cancellation/error preservation, pending-read suppression and no
browser fallback. Mock-native Edge checks at 880 x 620 and 360 x 620 cover button
names, textbox labels, keyboard activation/focus return, loaded icons, and long
text/error overflow. These are not real native clipboard approval evidence.
Local beta.20 packaging builds twice, compares ZIP digests and inspects directory
and archive permission/host/asset metadata. The maintainer later reported success
after the requested actual No/Yes and Notepad check; product readiness records
that general manual confirmation separately from independent per-action or
clipboard-byte evidence. The sample never saves pasted text or touches File
Notes data.

## Local Folder Access

Folder engine/IPC/Windows tests cover selected-directory tokens, explicit
permission/parameter denial, cancellation/replacement/revocation, document
generation and shutdown cleanup, real native dialog options and path rejection.
Listing tests exercise local disk identity, entry and escaped-JSON limits,
empty/Unicode folders, immediate-only enumeration and excluded reparse/offline
entries. The dependency-free `examples/folder-browser` additionally tests
token-only routing, text-safe rendering, cancellation, invalid-target handling
and busy controls. Packaging/startup do not replace real picker interaction.
Folder text-read tests cover the separate `folder.readText` permission,
strict basename/target validation, serialized/deferred work, redacted failures,
revocation before/during reads and shutdown. Actual Windows reads cover empty,
Unicode/BOM/exact-limit/oversized/invalid text, unchanged disk bytes,
directory/offline/hard-link/reparse rejection and handle-relative reads after
directory rename/replacement. Folder Browser tests also cover click-only reads,
literal content, cancellation preserving the preview, refresh/release clearing
it, detached stale rows, busy reads and unsupported/expired-target failures.
Manual UI preview remains separate evidence.

## Scope

general validation routes must stay stack-neutral unless a runner file explicitly defines a command.

## Repository Shape

cli-tool validation must stay repository-shape focused and must not imply generated application source code.
