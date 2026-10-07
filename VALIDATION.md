# Validation

- Status: M4 complete; M5 narrow alpha active; beta gated by product workflows with no human adoption claim

## Validation Source of Truth

This document owns stable validation names for this scaffold.

## Standard Validation Names

Text-editor word wrap (2026-10-08): source generation adds a default-on,
session-only Word wrap icon toggle and licensed local Lucide `text-wrap.svg`.
Off enables horizontal scrolling; the display change preserves text,
selection direction, logical line/grapheme status, save target, dirty state,
drafts and replacement Undo. IME, busy/load/recovery and modal guards apply.
47 Bun editor/position/replace/draft tests, initializer/CLI Go tests and vet,
and `git diff --check` passed. Edge mock-native checks verified mouse and
Enter/Space activation, editor focus return, backward selection and logical
position preservation, unchanged draft, saved target reuse and real horizontal
scrolling. Both modes passed at 960x640 and 320x560 in light, dark and forced
colors with loaded icons and no page overflow; desktop-off/mobile-on
screenshots were inspected. Owned browser/server closed. Generated delivery
has 19 web assets plus two root files. ZIP build/inspection passed using the
unchanged public alpha.68 host, SHA-256
`2654d1551f889b46241fe58ba5e51eefd0547ad617566187a0602cf6c9fd0b65`.
Web assets total 47,575 bytes, 1,743 above the preceding position fixture.
Receipts/screenshots are under `.cache/wrap-ui/` and are not committed.
README, CLI contract, product specification and icon notices updated. Native
Windows manual interaction, installer rerun and full release/startup/host-size
checks were skipped for this web-only change; manual native UI remains open.
No host/IPC API, native permission, DB/storage schema, dependency, repository
hygiene or CI runner changes. No version bump, push or publication; public
alpha.68 and existing generated apps are unchanged.

Text-editor position UI (2026-10-08): source generation delivers the cached
position helper and a footer for logical line, grapheme column and selection
count. Selection-only refresh does not reread text; larger edits debounce
80 ms and segment in 4096-grapheme chunks. IME defers indexing, cancellation
preserves the previous completed index, and unavailable indexing leaves text
untouched. Footer cursor changes are not live announcements.
44 Bun position/editor/replace/draft tests, initializer/CLI Go tests and vet,
and `git diff --check` passed. Edge mock-native checks passed for typing,
backward grapheme selection, replace/Undo, Open/New, recovery, IME pending
status and a 66,000-grapheme document. At 960x640 and 320x560, including dark
and forced-color modes, controls/icons loaded with no overflow or footer
overlap; desktop/mobile screenshots were inspected. Owned browser/server
closed. Generation delivers 18 web assets plus two root files.
Build with installer and ZIP inspection passed using the unchanged public
alpha.68 host, SHA-256
`2654d1551f889b46241fe58ba5e51eefd0547ad617566187a0602cf6c9fd0b65`.
Receipts/screenshots are under `.cache/position-ui/` and are not committed.
README, CLI contract and product specification updated. Native Windows manual
interaction remains unverified for this UI; host rebuild/startup/size and
full release checks were skipped because host bytes are unchanged. No host/IPC
API, native permission, DB/storage schema, dependency, repository-hygiene or
CI runner change. No version bump, push or publication; public alpha.68 and
existing generated apps are unchanged.

Text-editor position index (2026-10-07): an unused source helper indexes
logical LF line starts and extended grapheme boundaries using built-in
`Intl.Segmenter`, with an ASCII fast path. Cursor/selection queries use binary
search over cached boundaries, not repeated segmentation. Columns are
one-based; backward selection uses its start endpoint, and selection counts
intersected graphemes, including newline. Five Bun tests and
`git diff --check` passed for empty/trailing lines, combining marks, Korean,
flags/ZWJ emoji, repeated cached queries and 2 MiB ASCII/dense newlines.
UI delivery follows separately. No host/IPC API, permission, DB/schema,
dependency, native UI, CI, version bump or publication change.

Text-editor replace manual follow-up (2026-10-07): after the requested
Ctrl+H current/all replace, Undo button/editor-focused Ctrl+Z and
Save-then-Undo dirty-state checklist in `Velox Editor Replace Test`, the
maintainer reported that it worked. This is overall confirmation of the
requested scenario, not instrumented per-action assertions, disk readback
or an IME event trace. Test PID 44860 was absent at follow-up; no normal-exit
claim is inferred. The generated test used source replacement UI and unchanged
public alpha.68 host bytes, not replacement UI from the public alpha.68 CLI.
Existing automated results are reused; only `git diff --check` was rerun for
this record. No host/IPC API, DB/schema, permission, dependency, repository
hygiene or CI change. No version bump, push or publication.

Text-editor replace UI (2026-10-07): source generation adds Ctrl+H and an
optional literal replacement row with current/all/undo icon actions. The
target count remains visible before mutation. Replacement/undo update dirty
state and serial draft scheduling while preserving the native save target.
One memory-only before/after snapshot restores a replacement transaction;
editor-focused Ctrl+Z or the Undo button applies it. Direct input and successful
New/Open/Restore clear it; Save and closing find preserve it. IME, busy/modal,
empty-query and 2 MiB UTF-8 guards leave rejected text unchanged.
34 Bun editor/draft/replacement tests, initializer/CLI/hygiene Go tests,
scoped initializer/CLI vet and `git diff --check` passed. Edge mock-native
checks passed at 960x640 and 320x560, including dark and forced-color modes:
all/single/undo, literal tokens, native save-target retention, stored draft,
dirty baseline after Save/Undo, IME and empty-bar close, loaded icons and no
overflow. Desktop/mobile screenshots were inspected; owned browser/server
closed. Generated delivery now has 17 web assets plus two root files.
A portable test app built and inspected using unchanged public alpha.68 host
bytes; hash matched `2654d1551f889b46241fe58ba5e51eefd0547ad617566187a0602cf6c9fd0b65`.
Receipts/screenshots are under `.cache/replace-ui/` and are not committed.
Product specification, CLI contract, README and icon notices updated.
No host/IPC API, native permission, DB/storage schema, dependency,
repository-hygiene configuration or CI change. Manual native UI/picker,
host rebuild/size/startup measurement and release checks were skipped for
this web-only starter change. No version bump, push or publication; public
alpha.68 and existing generated apps are unchanged.

Text-editor replacement calculation (2026-10-07): an unused source helper
shares find's escaped literal pattern and supports validated UTF-16 single
match positions or non-overlapping whole-document replacement. Replacement
tokens remain literal; empty replacement deletes and empty query is a no-op.
Input and output are bounded to 2 MiB UTF-8, with expansion checked before
allocating output. 28 Bun editor/draft/replacement tests and
`git diff --check` passed, including Unicode offsets, case folding, dense
matches and rejected expansion. UI integration follows separately; public
alpha.68 and existing generated apps are unchanged. No host/IPC API,
permissions, DB/schema, dependency, native UI, CI or release change.

Alpha.68 publication (2026-10-07): the four verified artifacts from tag CI
`37628664660/1` were reused for unsigned prerelease publication without a
new build, rerun, tag move or hosted public-verifier dispatch. All four
unauthenticated public downloads matched CI sizes/SHA-256 and GitHub digests;
the public annotated tag resolved to source `5ffedc8`. Earlier CI manifest,
SPDX, checksum, provenance, CLI version, metadata and PE verification is
retained, not repeated as a new native test. No native UI, startup, picker,
find/draft/watch interaction, installer execution, performance/stress or
independent-adoption check ran against these public bytes. Evidence remains
`same-repository-public-download` with `externalUserAttempt: false`; beta is
held. README, quickstart, product/CLI status, release/readiness and risk
records now identify public alpha.68 while preserving historical records.
No API/IPC, DB/schema, permission, dependency, hygiene or CI change. Full
identity and receipts: `docs/ops/alpha68-publication.md`.
Documentation follow-up hygiene Go tests, scoped vet and `git diff --check`
passed; the current-channel test now identifies published alpha.68.

Alpha.68 tag CI (2026-10-07): verified source `5ffedc8` and annotated tag
were pushed atomically and their remote SHAs matched. Run `37628664660/1`
passed releasebundle/releaseevidence/hygiene tests, two independent release
builds and basic checkout-free init/validate/doctor/build twice/inspect.
Downloaded CI artifacts passed 16 manifest entries, 17 SPDX SHA-256 entries,
three sidecar checksums, source/run/ZIP provenance identity, exact CLI version,
PE subsystems and Go 1.26.0 clean-source metadata. The binaries are unsigned;
the publication job was skipped and GitHub's alpha.68 release lookup returned
HTTP 404. Public alpha.67 remains unchanged. No native UI, installer execution,
watch/find/draft native repeat, performance/stress or beta promotion. No new
API/IPC, DB/schema, permission, dependency, hygiene or CI workflow change.
Full identity and receipts: `docs/ops/alpha68-tag-evidence.md`.

Alpha.68 local candidate (2026-10-07): clean source `f6c4081` produced a
Go 1.27.1 CLI, GUI host and GUI Setup. Two packaging runs from the same
binaries matched; independent recompilation was not tested. Outside-checkout
prebuilt consumers passed version/templates and all four starters'
init/validate/build --installer/inspect, with repeated portable ZIP hashes
and generated host/type identity checks. All 16 manifest artifacts, three
sidecar checksums, 17 SPDX file SHA-256 entries, source/ZIP provenance and
PE subsystem checks passed. Host hash/size stayed identical to the recorded
alpha.67 local candidate; CLI grew 21 KiB. No native UI, installer execution,
startup/memory comparison, hosted stress, signing, push, tag or publication.
No host/IPC API, DB/schema, permissions, dependency or CI change. Exact
hashes and receipt paths are in `docs/ops/alpha68-preparation.md`.

Alpha.68 version preparation (2026-10-07): source version and dependent
fixtures identify `0.5.10-alpha.68` as a local unpublished candidate while
public alpha.67 and historical release records remain unchanged. Buildplan,
builder, CLI, inspector, runner, releasebundle, releaseevidence and hygiene
Go tests and scoped `go vet` passed; `git diff --check` passed. Candidate
identity follows the version commit in `docs/ops/alpha68-preparation.md`.
No host/IPC API, DB/schema, permissions, dependency or CI change. Native UI,
installer execution, performance and hosted stress were not repeated for
this version step. No push, tag, signing or publication.

Text-editor find close fix (2026-10-07): the maintainer confirmed that a saved
native file reopened and search worked, but reported Escape and X failing
with an empty query. Two mocked tests reproduced close failures with stale
find-input IME state before the fix; the precise native event sequence was
not captured. The fix uses a text input and capture-phase key handling,
permits explicit close despite pending query composition, honors actual
composing key events, clears state on blur and ignores late composition-end
events after close. 24 Bun tests, initializer/CLI Go tests, scoped vet and
Edge mock-native empty-query/document, late-IME, dark and forced-color checks
passed. A corrected portable test app built and inspected with unchanged
public alpha.67 host bytes. In the manual follow-up, the maintainer reported
that the corrected find bar now closes in response to the empty-query
Escape/X check. The corrected test process (PID 38496) was absent afterward.
This is an overall manual close confirmation, not an instrumented native
IME event trace or separate assertion for every keyboard/composition case.
Command contract updated; `git diff --check` passed. No host/IPC API,
DB/storage schema, permission, dependency, repository-hygiene or CI change.
No host rebuild/performance check, version bump, push or publication.

Text-editor starter find (2026-10-07): new source-CLI editor generation adds
a hidden find bar, literal case-toggle search, counts, wraparound navigation
and IME-safe shortcuts. Initializer/CLI Go tests, scoped vet and 21 Bun
editor/draft tests passed, including original UTF-16 offsets, 2 MiB dense
matches, replacement/recovery, save preservation and draft authority.
Generated delivery includes 14 web assets and the same two root files.
Playwright Edge mock-native checks passed at 960x640 and 320x560, plus dark
and forced-color modes: selection, wrapped long-text scrolling, keyboard
focus, no overflow, loaded icons, no native calls during find and unchanged
persisted draft text. Screenshots were inspected. The owned browser/server
closed; local receipts are under `.cache/find-ui/` and are not committed.
`git diff --check` passed. This is browser/mocked-native evidence, not a
manual Windows host/picker check. Native UI, host size/startup measurement
and release checks were skipped for the web-only starter change. Product
specification, command contract and icon notices updated; host/IPC API,
DB/storage schema, permissions, dependencies, repository hygiene and CI
unchanged. No version bump, push or publication; public alpha.67 and
existing generated projects are unchanged.

Init next-step guidance (2026-10-07): successful human init output includes
run/build commands for the generated manifest, labeled and literally quoted
for PowerShell on Windows or POSIX shells elsewhere. CLI Go tests and scoped
`go vet` passed for spaces/apostrophes/shell metacharacters, JSON/quiet modes,
failure output and byte-identical generated files. An actual source CLI init
created a text-editor fixture under a path containing spaces, an apostrophe,
`$`, a backtick and `&`; PowerShell AST parsing recovered the exact manifest
argument from both printed commands without executing them.
`git diff --check` passed. POSIX forms were unit-tested only; no Linux shell,
native UI, host rebuild or release check ran for this CLI-output change.
Command contract updated; host/IPC API, DB/schema, dependencies, repository
hygiene and CI unchanged. No version bump, push or publication; this guidance
is not in public alpha.67.

CLI help discovery (2026-10-07): source top-level help now lists command
purposes and generation/run/build examples. Init help points to `templates`
and retains its flag list. CLI Go tests and scoped `go vet` passed, including
help aliases, stdout/stderr and exit-code preservation, silent JSON help,
and no project creation. Actual `go run ./cmd/velox --help` and
`go run ./cmd/velox init --help` both exited 0 with the expected text;
`git diff --check` passed. README and command contract distinguish these
source-only additions from public alpha.67. Host/IPC API, DB/schema,
dependencies, repository hygiene and CI are unchanged. Native UI, host
rebuild and release checks were skipped for this help/documentation change.
No version bump, push or publication.

Template catalog command (2026-10-07): the source CLI adds read-only
`templates` with human/JSON/quiet/help output and usage errors. Its four
entries share permissions with project generation. Initializer and CLI Go
tests and scoped `go vet` passed, including catalog/generation agreement,
independent catalog values, JSON envelope and an unchanged empty working
directory. `go run ./cmd/velox templates` displayed all four generation
commands; `git diff --check` passed. No host/IPC API, DB/schema, dependency
or CI change. Native UI, host rebuild/size measurement and release checks
were skipped because only CLI/initializer code and documentation changed.
No push or publication; public alpha.67 does not include this command.

Alpha.67 publication (2026-10-06): source `cb801f5` and annotated tag were
pushed together. Tag CI `37471273685/1` passed independent reproducible
producer builds and a basic checkout-free consumer smoke with Go 1.26.0.
The four verified files were reused for unsigned prerelease publication
without another producer, tag move or public-verifier dispatch. Unauthenticated
downloads matched CI and GitHub digests; three checksum entries, 16 manifest
artifacts, 17 SPDX SHA-256 entries and provenance source/run identity passed.
The CI-extracted CLI reported alpha.67 and matched the public ZIP's bytes.
No new native UI, startup, watch, draft, picker, tray/balloon, installer,
performance/stress or external-adoption check ran against these public bytes.
Evidence remains `same-repository-public-download` with `externalUserAttempt: false`.
Exact identities and skipped checks are in `docs/ops/alpha67-publication.md`.

Alpha.67 local artifact follow-up (2026-10-06): clean source `4d73cfb` produced
matching-version CLI, GUI host and GUI Setup with Go 1.27.1. Two packaging
runs from those binaries matched. All four outside-checkout prebuilt consumer
template checks passed, as did 16 manifest entries, three sidecar checksums,
17 SPDX file SHA-256 entries and unsigned provenance identity. The exact native
tray smoke initially failed input visibility with hidden GUI startup; visible
manual-check and default automated startup after removing that fixture flag
passed readiness, native notification acceptance, single-instance preservation
and normal close. No owned process remained. The failed receipt is retained.
No new manual alpha.67 observation, installer execution, manifest-watch native
repeat, hosted/performance/stress, signing, push, tag or publication is claimed.
No API/IPC, DB/schema, dependency or CI change. Exact artifact identity and
boundaries are in `docs/ops/alpha67-preparation.md`.

Alpha.67 version preparation (2026-10-06): source version and version-dependent
fixtures now identify `0.5.10-alpha.67`. Source/public documentation and hygiene
checks keep alpha.67 local/unpublished while preserving public alpha.66 and
its historical records. Artifact identity follows the version commit in
`docs/ops/alpha67-preparation.md`. No API/IPC, DB/schema, dependency, CI, tag,
push or publication is included.
Buildplan/builder/CLI/inspector/runner/releasebundle/releaseevidence/hygiene Go
tests and `git diff --check` passed for this version change.

Source candidate `9dfcb8b` preparation (2026-10-06): compiled the CLI once
with Go 1.27.1 and reused manifest-checked public alpha.66 host/Setup bytes.
Two packaging runs matched, without claiming independent recompilation.
An outside-checkout prebuilt consumer passed init/validate/build --installer/
inspect for all four templates. Generated hosts/types matched bundled inputs;
tray packaging repeated identically. All 16 release-manifest entries, three
sidecar checksums, 17 SPDX SHA-256 entries and provenance source/ZIP identity
passed. Releasebundle/releaseevidence/hygiene Go tests passed. Prior matching
tray native bytes/manual confirmation and unchanged implementation tests were
reused; no installer execution, hosted/performance/stress or native-watch
repeat occurred. No version, push, tag, release, API/IPC, DB/schema, dependency
or CI change. See `docs/ops/source-candidate-9dfcb8b.md` for exact identity.

Tray starter manual follow-up (2026-10-06): the maintainer confirmed hide,
restore, visible bell-button notification and tray Quit all worked in the
replacement diagnostic window. The diagnostic process exited 0; the owned
host/supervisor processes were absent afterward. The first launch produced a
blank window and was force-cleaned, not accepted. Its cause is unresolved.
The smoke tool now has visible `--manual` and readiness-only `--manual-check`
modes, requiring a ready document and visible Message input before handoff.
Automated readiness/cleanup is separate from the maintainer's report; details
and retained failed receipts are in `docs/ops/tray-app-starter.md`.
The default native/CDP smoke, `--manual-check`, `go test ./tests/hygiene`,
Node syntax check and `git diff --check` passed. The default run also retained
notification acceptance and single-instance document preservation checks.
No host/public IPC, DB/schema, dependency, CI, version or release change.

Tray starter UI/native follow-up (2026-10-06): the source CLI from `83e78c1`
generated/validated/built/inspected the starter with only `notification.show`.
Five web assets total 8,387 bytes; the packaged EXE is byte-identical to the
unchanged public alpha.66 host. Edge/Playwright light/dark checks at 620x480
and 360x540 passed without horizontal overflow, with a fixed 40x40 rendered
bell button, Space activation, literal text retention and focus return. Mock
bridge UI evidence is separate from the packaged native/CDP run: one real
notification request was accepted, a second invocation retained the same
document/text, and normal close/cleanup returned 0 with no owned process
remaining. The first native screenshot timeout was force-cleaned; an
intermediate file-URL preview showed blank masks. A loopback HTTP fixture
fixed rendering without product changes; both earlier receipts are retained.
Source/public hashes, repeat command and skipped manual tray/OS-balloon,
installer/stress/performance checks are in `docs/ops/tray-app-starter.md`.
Implementation Go/vet/Bun results were reused; final hygiene/diff checks passed.
No host/public IPC, DB/schema, dependency, CI, version or public release change.

Tray starter implementation (source-only, 2026-10-06): initializer/CLI tests
and scoped vet passed, plus four Bun cases for explicit submission, UTF-16/
control-character bounds, duplicate-request suppression, failure recovery,
literal text retention and missing bridge without browser fallback. Go cases
cover exact `notification.show` permission/default settings, seven-file/five-
asset inventory, escaped app names, preserved conflicting icons, unchanged
basic output and all CLI template argument forms. UI/native checks are pending
at this implementation step, not claimed as passes. README, CLI contract and
product spec identify the starter as source-only. No host/API/IPC, DB/schema,
dependency, CI, version or release change is included. Existing host tray
behavior is reused; no new scheduler, background worker, shortcut or font.

JSON manifest-watch output (source-only, 2026-10-06): focused runner/CLI tests,
scoped vet, hygiene and diff checks passed. Debug-off/on CLI tests observe
manifest error/recovery notices on stderr while stdout parses as one success
envelope; injected host stdout is suppressed and injected host diagnostics
appear only with debug enabled. The runner now separates watch and host stderr,
retaining a shared output lock and joined cleanup without changing host flags.
The native smoke's new `--json` mode passed with a local source CLI and the
unchanged public alpha.66 host: two notices/one error, retained unsaved text,
document/origin/config digest, no dialogs, one successful stdout envelope,
close/cleanup exit 0 and temporary-config removal. It did not inject a native
host diagnostic; that suppression result belongs to the launcher tests.
README, CLI contract and `docs/ops/development-watch.md` record the behavior
and hashes. No public IPC, DB/schema, dependency, CI, version or release change
is included; installer, hosted stress and performance/size checks were not
repeated for this CLI-only change.

Developer documentation refresh (2026-10-06): README capability/permission
rows were checked against IPC tables, manifest settings and example manifests;
all 19 referenced local documentation/example targets exist. Hygiene tests and
diff checks passed. The roadmap now names public alpha.66 using the existing
publication/native records, while keeping alpha.65/64 evidence historical.
Source-only manifest-watch notices are separated from the published download.
CommandCode DeepSeek 4.1 Flash/high drafted the table wording; names and paths
were checked locally. No runtime/API/IPC, DB/schema, dependency, runner, CI or
release change was made. Native, build, performance and release checks were
not repeated for documentation-only edits.

Manifest watch notices (source-only, 2026-10-06): scoped manifest, devwatch,
runner, CLI and hygiene tests passed, as did scoped vet and diff checks.
Fresh verbose manifest/runner tests passed without skips, including captured
byte parsing, custom config paths, same-size/time edits, quiet-period debounce,
reverted edits, invalid JSON/schema recovery, missing/oversized/link rejection
and cancellation. A runner test with a fake host launcher observed the real
CLI polling loop: invalid input emitted a nonfatal error, corrected input
emitted a restart notice, the launcher ran once, runtime config bytes stayed
unchanged, host exit code 5 was preserved and the loop was joined on return.
This is automated CLI evidence, not native GUI or manual preservation evidence.
Normal runs and packaged apps start no manifest loop. The internal parser is
shared; no public API/IPC, DB/schema, dependency or CI change is included.
Runner changes are limited to opt-in observation, serialized stderr output
and loop cleanup. Contracts/spec and `docs/ops/development-watch.md` were
updated. Native UI, release/installer, hosted stress and performance/size
measurements were not repeated for this CLI-only change. Public alpha.66
remains unchanged; no version bump, publication or beta promotion occurred.

Manifest watch native follow-up (2026-10-06): the source CLI from `5b7dda3`
was built with Go 1.27.1 and paired with the unchanged public alpha.66 host.
`bun run scripts/manifest-watch-smoke.ts` passed in an isolated File Notes
project/profile with watch on and debug off. A valid name/size/permission edit,
invalid JSON, then corrected settings emitted exactly two restart notices and
one error. All three phases retained the document marker, unsaved editor text,
origin and runtime-config SHA-256; no browser dialog opened. Normal close
returned CLI/cleanup exit 0 and removed the temporary runtime configuration.
The first attempt failed its dirty-editor baseline before initialization;
waiting for editor readiness fixed the fixture without a product change.
This is native/CDP automation, not a new human confirmation. Hashes, receipts
and the repeatable command are in `docs/ops/development-watch.md`. Existing
unit/vet results were reused; script bundling, hygiene and diff checks passed.
Release/installer, hosted stress and performance/size benchmarks were not
repeated for this verification-only follow-up. No runtime/API/IPC, DB/schema,
dependency, CI, version or public-release change is included.

Alpha.66 local preparation (2026-10-05): source version is
`0.5.10-alpha.66`, not published; public alpha.65 evidence is preserved.
Version-dependent buildplan, builder, CLI, inspector, runner, host metadata,
release-bundle and release-evidence tests passed, as did hygiene tests and
scoped vet. Buildinfo has no test files. `go run ./cmd/velox version --json`
reported alpha.66. The publishing script parsed and its release-notes text
contained no control escapes. Existing implementation/native evidence was
not repeated for a version/prose-only preparation. No alpha.66 distribution
bundle, digest, hosted result, tag, push or publication is claimed; old
source-11efd69 binaries remain alpha.65-string artifacts. API/IPC, DB/schema,
dependencies and runner/job scheduling are unchanged; workflow changes only
correct the generated release description. Scope and skipped-check boundaries
are in `docs/ops/alpha66-preparation.md`.

Alpha.66 tag CI (hosted, 2026-10-05): the prepared source `842fead` and
annotated tag `v0.5.10-alpha.66` were pushed in sequence and triggered one
alpha-evidence run 37317775399 (attempt 1), which passed the reproducible
unsigned producer and checkout-free consumer jobs with Go 1.26.0 on Windows
amd64. The producer ran release-contract tests, two release builds, evidence
generation and bundle upload; the consumer checked out no source and invoked
only `velox.exe`. The release ZIP is 5,903,850 bytes, SHA-256
`3e1bc83cc8ee31e8feb4870a5b4fa26a4f992a5e22754d0625de891394cfacdc`, both
consumer app ZIPs hashed
`0ee5221c541312848255165c2dc8b0b1ef97bf157d4e17d4b33c7919f694fa33`, and the
three checksum entries, 16 manifest artifacts, 17 SPDX members and the
provenance subject/source/invocation `37317775399/1` were checked against the
ZIP. The publication job was skipped in that CI run, so at that step alpha.66
was not yet published and public alpha.65 was unchanged; publication followed
separately and is recorded below. No native UI, startup, installer, watch, draft or
public-download interaction ran against these CI bytes, and those are not
upgraded from the earlier local results. Receipts are in
`docs/ops/alpha66-tag-evidence.md`.

Alpha.65 source candidate `11efd69` (source-only, 2026-10-05): local source
`11efd69a7dfeda04ec84e31b804b6af68cc29bad` was clean at build time and kept the
`0.5.10-alpha.65` string, but its bytes differ from the public alpha.65
prerelease. CLI, host and Setup were built once with Go 1.27.1
(`-buildvcs=false -trimpath -s -w`, host/Setup GUI); the candidate ZIP is
6,863,300 bytes, SHA-256
`7e57f3d18015923ee69873b8baa55516b788666ab78dcc3d714c8dd4596278dc`. The
release ZIP was extracted outside the checkout and only its prebuilt CLI
passed init/validate/build --installer/inspect for the text-editor and
folder-browser templates, with both extracted packaged EXEs byte-matching the
candidate host and each generated project's root `velox.d.ts` matching the shipped
types. Runtime permissions stay exactly `file.open`/`file.save` and
`folder.read`/`folder.readText`. A packaged text-editor native Save used an
isolated profile: automated exact readback was 44 UTF-8 bytes, the draft
cleared, relaunch did not resurrect it and no candidate-owned process
remained. Scoped Go tests/vet and 23 Bun cases plus prior draft/diagnostics/
watch evidence were reused instead of broad repeats. No installer execution,
hosted stress, benchmark, signing or publication was performed; absence is not
a pass. Full hashes, manual-save notes and boundaries are in
`docs/ops/source-candidate-11efd69.md`.

Text-editor draft recovery (source-only, 2026-10-05): initializer/CLI/hygiene
Go tests and scoped initializer/CLI vet passed; 23 Bun cases passed across
draft storage, text-editor and folder-browser. Storage cases cover transaction
completion, aborts, blocked/late opens, invalid/oversized records and omission
of native authority. UI cases cover restoration, durable discard, serialized
write/clear ordering, failure protection and IME coalescing. Edge mock checks
passed at 960/360 widths in light/dark modes with no overflow, modal Escape
protection, literal restored text and no restore-time native calls. Native CDP
passed write, restore/clear and cleared-relaunch with close/cleanup exit 0.
That smoke does not verify an actual save-dialog destination re-pick or the
OS beforeunload prompt. The unchanged host is 4,861,440 bytes; no performance
benchmark, hosted stress or installer test was repeated for template-only
assets. Public alpha.65 is unchanged. New template-local IndexedDB storage is
explicit; host DB, public API/IPC, schemas, dependencies and CI are unchanged.
Hashes and the repeatable smoke command are in `docs/ops/text-editor-drafts.md`.

Development diagnostics (source-only, 2026-10-05): scoped devdiagnostic,
WebView2, CLI, runner and host tests/vet passed. Six Bun listener cases cover
same-origin path/query removal, untouched error/rejection contents, rejected
remote/credential/file sources, frames, missing binding, storm bounds and
failing bridge suppression. Native normal/debug `run --json` both passed
with one stdout envelope and close/cleanup exit 0. Default mode had no
diagnostic binding or output; debug recorded actual error/rejection metadata
without secret bodies, URL tokens or forged absolute paths. An invalid
extra-field request plus a 50-call storm confirmed the 20-attempt budget.
The same-toolchain host grew 6,656 bytes. Source/public bytes and the known
unknown-location limitation are in `docs/ops/development-diagnostics.md`.
No public IPC/permission, DB/schema, dependency or CI change is included.
JSON stderr forwarding changes only with explicit debug. Hosted stress,
installer and production performance checks were not repeated for this
development-only extension; no public release or beta promotion occurred.

Image/font development watch (source-only, 2026-10-05): scoped devwatch,
runner, CLI and WebView2 tests/vet passed. Detector cases cover all supported
extensions including uppercase forms, metadata-only same-size edits, stable
debounce, addition/rename/removal, a sparse font larger than the 64 MiB text
budget, the documented same-size/time limitation and link rejection without
skips. The native smoke used a matching local CLI/host, copied assets and a
private profile with debug off: image-only SVG edits changed decoded canvas
pixels, font-only edits changed loaded-font metrics, and each caused normal
automatic reload without HTML/CSS/JS edits or test-side cache/navigation
overrides. Existing text reload, canceled-input protection, subsequent retry
and close/cleanup with exit 0 also passed. The first inline test fixture was
blocked by CSP; external local CSS/JS fixed the fixture, not the runtime.
The same-Go/flags host grew 1,024 bytes. Public alpha.65 is unchanged; no
hosted release/stress, installer or production performance test was repeated
for this development-only extension. Receipts and hashes are in
`docs/ops/development-watch.md`. No API/IPC/DB/schema/dependency/CI change.

Folder-browser starter (initial source-only step, 2026-10-05): scoped initializer/CLI tests
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

Text-editor starter (initial source-only step, 2026-10-05):
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

The current public preview is `v0.5.10-alpha.68`, source
`5ffedc8d6e17b16e6e13ca8c57f21c06d76dfe8c`; tag CI `37628664660/1` and
public-byte checks passed. ZIP SHA-256:
`b8e916c41c0d9fa6bc8ce21c6254e8adab8121c37d183da7a9e2cea99ba05bf8`.
See [alpha68-publication.md](docs/ops/alpha68-publication.md). Prior alpha.67
identity and scope remain in [alpha67-publication.md](docs/ops/alpha67-publication.md).

An earlier public preview is `v0.5.10-alpha.66`, source
`842fead0c889e9f161c2567a91c8d0fd4c2ca260`. Tag CI `37317775399` (attempt 1)
passed reproducible unsigned producer builds and a basic checkout-free
consumer smoke with Go 1.26.0. Its four verified assets were published as an
unsigned prerelease without a second producer, a tag move or a hosted public
verifier dispatch. Unauthenticated public downloads matched the CI and GitHub
digests; the three checksum entries, release manifest, SPDX, provenance
`37317775399/1`, ZIP/CRC and the extracted public CLI version
`0.5.10-alpha.66` were checked. ZIP SHA-256:
`3e1bc83cc8ee31e8feb4870a5b4fa26a4f992a5e22754d0625de891394cfacdc`
(5,903,850 bytes). This verifies public artifact identity and the public CLI
version only; no native UI, startup, installer, watch, draft or public native
interaction ran against these published bytes. It is
`same-repository-public-download` evidence with `externalUserAttempt: false`;
receipts are in [the alpha.66 publication record](docs/ops/alpha66-publication.md).

Alpha.66 public native follow-up (2026-10-05): a later local step ran the
existing text-editor draft, dev-reload watch (HTML/CSS/JS plus image/font) and
development-diagnostics scripts against the exact downloaded public bytes,
plus one manual packaged-window draft/save/recovery check. Every script run
exited 0; the manual check passed post-save no-resurrection on the same
profile and unchanged host hash. No normal GUI close or no-residual-process
claim is made. Installer, folder-browser, hosted-verifier, stress, performance
and external adoption were not run. Evidence stays `same-repository` with
`externalUserAttempt: false`. See the
[alpha.66 public native follow-up](docs/ops/alpha66-public-native.md) and the
[publication record](docs/ops/alpha66-publication.md).

The previous public preview was `v0.5.10-alpha.65`, source
`c8f618bd94e48cb7c01d61aa0e65e3bc7116875c`. Root-module tests/vet and all
12 Bun template cases passed locally before tagging. Tag CI `37298703202`
passed reproducible unsigned builds and basic checkout-free consumer
packaging. Its four assets were reused for publication without a second
producer. Unauthenticated downloads matched CI and GitHub digests, checksum,
manifest/SPDX files, ZIP/CRC, provenance source/run and the public CLI version.
ZIP SHA-256:
`7f837fe69ff4ec9efcf528c97c63ea1d3dc010ba2a7d75d2eb824a1ff372c42b`
(5,896,489 bytes). Both templates passed public-CLI generation, exact
permissions/inventory, validate/doctor, portable plus Setup packaging and
inspection. The exact public CLI/host also passed native watch reload,
canceled-input preservation, retry and normal close/cleanup with debug off
and a private profile. Host/Setup sizes are unchanged from alpha.64, but
their hashes differ; the CLI grew 47,104 bytes. Maintainer-confirmed text
saving and Korean folder preview used local source-generated apps with
the alpha.64 host before publication, not alpha.65 public-byte native dialogs.
No hosted native template interactions, installer execution, full stress or
beta promotion occurred. This is same-repository-public-download evidence,
with `externalUserAttempt: false`; receipts are in `docs/ops/release.md`.

The previous public preview was `v0.5.10-alpha.64`, source
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
