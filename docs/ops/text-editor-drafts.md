# Text-editor Draft Recovery: 2026-10-05

> Follow-up: text-editor draft recovery now ships in the published alpha.66
> prerelease. The source-only evidence below is historical; its local hashes
> are not the alpha.66 public release bytes. See
> [the alpha.66 publication record](alpha66-publication.md).

> Source-only follow-up (2026-10-10): generated editors now preserve source
> newline sequences in the existing draft `text` field. Restore detects LF,
> CRLF or CR again; old LF records remain compatible without a schema change.
> Unedited mixed endings are retained exactly, while edited text follows the
> first newline sequence. Draft validation still applies to serialized UTF-8,
> so CRLF expansion beyond 2 MiB reports recovery unavailable and leaves the
> previous valid draft intact. Edge/IndexedDB reload and captured save-byte
> checks passed; native picker/end-to-end WebView2 save was not repeated.
> Public alpha.68 and existing generated projects are unchanged.

## Source-only Scope

Source-only follow-up (2026-10-11): the editor no longer retains an unbounded
chain of full-document snapshots when storage is slow. It keeps one in-flight
operation, one latest waiting snapshot and a protected pending clear boundary.
A clear removes older waiting snapshots; later snapshots cannot replace or
overtake it. Repeated pending clears share one completion promise. Callers wait
for clear completion, failures do not poison the queue, and only the current
revision updates the draft status. The 300 ms debounce and existing v1 record
are unchanged. These assets are not in public alpha.68 or existing editors.

Six new queue regressions cover delayed storage, superseded snapshots, first
write failure, awaited clearing, post-clear writes, failed clearing and repeated
clear requests. Four initially failed against the old serial chain. All 71
initializer Bun tests, initializer/CLI Go tests and scoped vet passed. A newly
built CLI generated a private editor; headless Edge with real IndexedDB and
mock native saves committed only the first/latest of six delayed snapshots,
restored the latest Unicode draft after reload, awaited a clear without storing
obsolete snapshots, and did not restore a cleared draft after another reload.
The browser and loopback server closed; `.cache/draft-queue-ui/result.json`
is an uncommitted local receipt. An initial UI probe waited for a dirty label
that only renders after cleanup and timed out; waiting for the immediate save
status fixed the probe. This is not native picker/WebView2 or release evidence.

New `init --template text-editor` projects include local IndexedDB draft
recovery. Public alpha.65 and existing generated projects are unchanged.
The host, IPC v1, native permissions, schemas, dependencies and CI workflows
are unchanged; only template assets and developer tests are added.

The `velox.text-editor` database stores one `drafts/current` record with exactly
`schemaVersion`, `name`, `text` and `updatedAt`. Text must fit both the
2 MiB character bound and 2 MiB UTF-8 byte bound. Names are control-free and
at most 255 characters. Invalid records are rejected. Native paths, save
tokens, permissions and saved-text baselines are never stored.

Typing pauses trigger a 300 ms debounce; writes and clears share one serial
chain. IME composition defers persistence until composition ends. Success
requires transaction completion, not merely request success. Restore keeps
the document dirty with no native target; its first Save chooses a fresh
destination. Discard must commit before dismissing recovery. New, successful
Open and successful Save clear the old draft; cleanup failure is reported.
Storage errors do not disable editing or bypass dirty-document close protection.

Drafts are local application data, not encrypted backups or remote sync.
Abrupt exit can lose edits still in the debounce or pending transaction;
storage cleanup failure can leave an older draft available on relaunch.

## Local Verification

- `bun test internal/initializer`: 23 passed (5 storage, 12 text-editor,
  6 folder-browser).
- `go test ./internal/initializer ./internal/cli ./tests/hygiene` and
  `go vet ./internal/initializer ./internal/cli`: passed.
- Edge mock-browser checks: 960/360 widths, light/dark modes, no horizontal
  overflow, recovery dialog fit, Escape retention, literal restored text,
  dirty state, four-field records and no restore-time native calls: 4/4 passed.
- Native WebView2/CDP smoke: write/persist, relaunch/restore, New/discard/clear,
  then cleared relaunch without resurrection: all passed. Each process closed
  with exit 0 and required no forced cleanup. This used an isolated profile,
  not the user's editor or draft data.

Native receipt: `.cache/text-editor-draft-1791203253751/result.json`.
An earlier fixture attempt failed because input was not focused; the fixed
fixture passed. The failed attempt is not counted as successful evidence.
Mock screenshots: `.cache/draft-recovery-*.png` and `.cache/draft-restored-*.png`.
These local cache receipts are not included in release packages.

Local CLI SHA-256:
`386b591217e41dd35f8ec30c2580c7b9323a903b84514570c73efd149c3a5b0e`.
Local host SHA-256:
`2b6a0214b2c82eee8790273fb7426de7e405b9afb262ac007e74bea16c1f66d3`.
Both use the alpha.65 source version string, not its public release bytes.
The host remains 4,861,440 bytes; no host-size change is attributed to drafts.
Generated text-editor output now contains 9 web assets and 11 total files.

Repeat with Bun on Windows and a matching local release bundle:

```powershell
$env:VELOX_DRAFT_RELEASE_DIR = '.cache/draft-recovery-bundle/velox-windows-x64'
bun scripts/text-editor-draft-smoke.ts
```

The smoke checks release-manifest hashes, generates a private project/profile,
opens a temporary loopback CDP port and cleans up its own processes. It does
not require changing packaged defaults or adding a CI job.

Actual save-dialog destination re-picking and the native beforeunload prompt
were not exercised in this draft smoke; their application-side behavior is
unit-tested. Hosted stress, installation and production startup/idle benchmarks
were not repeated for these template-only assets. No public release, push or
beta promotion is part of this step.
