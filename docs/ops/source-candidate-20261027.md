# Alpha.68 Source Candidate `739490b`: 2026-10-27

## Identity

Local source-candidate receipt, not a published payload. The version string
remains `0.5.10-alpha.68`, but these bytes differ from the public alpha.68
prerelease. The repository records alpha.68 as the current unsigned public
preview; this receipt makes no live-service or latest-release claim. No push,
tag, release, hosted CI run or beta promotion is part of this step.

- Source: `739490b325522f8e5c7d26a237607a26eea959a6` (`main`), a test-only
  fix that isolates installer registry fixtures by full temporary path in
  `internal/installer` test files.
- Product head before merge: `b197891`.
- Merge: `065cee6a3103ca6c89bda33487d762e0db3154f1`, with parents `b197891`
  and remote `5f8df6d`. The remote commit moves the Clarissimi Action pin from
  `v0` to `v1` and updates its CI note; it changes no product code.
- Toolchain: `go version go1.27.1 windows/amd64`; `go.mod` declares Go 1.26.0.
- Signing: `velox.exe`, `velox-host.exe` and `velox-setup.exe` report
  Authenticode `NotSigned`.
- Candidate root: `.cache/review-candidate-20261027-065cee6/`; the directory
  keeps the earlier merge-commit name.

Scope is local candidate evidence plus the test-fixture fix. It adds no public
API, database schema, new runner, product dependency or application version change. The candidate
groups installer and uninstall data protection, profile and shutdown
preservation with File Notes/Text Editor draft handling, and editor, folder
browser and CLI improvements. The adopted review item is to record candidate
provenance and verification before any release decision.

## Build

Builds used the alpha-evidence flags, with a private
`GOCACHE=<candidate-root>/go-cache-resumed`:

```
go build -trimpath -ldflags='-s -w' -o <input>/velox.exe ./cmd/velox
go build -trimpath -ldflags='-s -w -H windowsgui' -o <input>/velox-host.exe ./cmd/velox-host
go build -trimpath -ldflags='-s -w -H windowsgui' -o <input>/velox-setup.exe ./cmd/velox-setup
go run ./cmd/velox-release --cli <input>/velox.exe --host <input>/velox-host.exe --setup <input>/velox-setup.exe --source-root <repo> --out <output> --json
```

Only `clean-first` and `clean-second` are authoritative. Earlier `first`,
`second`, `resumed-second` and `verified-second` artifacts are excluded. After
the interruption, the original `first-release` CLI had a zeroed header; the
cause is not established. None of those older hashes is reused.
`clean-reproducibility.json` reports `equal: true` for every
executable and both ZIPs; the three executables are byte-identical across
builds.

| File | Bytes | SHA-256 |
| --- | --- | --- |
| `velox.exe` | 5,122,560 | `d9be07bc50834aa8858fa9e94405dc754f92a655a2081208cbfb7e204c3bf046` |
| `velox-host.exe` | 4,934,144 | `27a4fe981f2edff112c4322f72859258f8984f9fee5bc8f80d96a1cb420bf65d` |
| `velox-setup.exe` | 4,170,240 | `c8bfbfb54c94531baf88bcfc024b933c02a56c6368814094e224d195bd56e038` |
| `velox-windows-x64.zip` | 6,948,234 | `b33ab953a6ccd2cc9c965116ca8ab541dba51a2c5867fd7433a5b18b5a523477` |

## Verification

| Check | Result | Evidence |
| --- | --- | --- |
| Source tests | Pass: `go test -json -count=1 -timeout=5m ./...`, 44 packages, exit 0, no failed actions | `source-tests-fixed-status.json`, `source-tests-fixed.jsonl` |
| Installer fixture isolation | Pass: `internal/installer` package 112.219s | `installer-isolated-fixed-status.json`, `installer-isolated-fixed.jsonl` |
| Consumer packaging | Pass: three templates, deterministic ZIP/Setup hashes, `inspect` succeeded | `consumer-results.json` |
| Host startup | Pass: `go test ./tests/startup -run '^TestBuiltHostStartup$' -count=1 -v`, 5/5 subcases, 47.18s | `native-startup.log`, `native-startup-status.json` |
| Immediate relaunch | Pass: 3 pairs, 46.08s, WebView2 155.0.4283.62, profiles released | `native-lifecycle.json`, `native-lifecycle.log` |
| Installer smoke | Pass: shortcut launch exit 0, browser exit, uninstall, 163 profile files and 5 IndexedDB files preserved | `installer-smoke-status.json`, `installer-smoke.log` |
| Text Editor draft smoke | **Fail**: no text entered; both attempts stopped before restore and clear | `draft-smoke.log`, `draft-smoke-focus-fixed.log`, `draft-smoke-status.json` |

The consumer ran only the candidate release's `velox.exe`, with this sequence per template,
with projects `consumer-basic`, `consumer-text-editor` and
`consumer-folder-browser` under the candidate root:

```
velox.exe version --json
velox.exe init <project> --json
velox.exe validate --config <project>/velox.json --json
velox.exe doctor --config <project>/velox.json --out dist-first --json
velox.exe build --config <project>/velox.json --out dist-first --installer --json
velox.exe build --config <project>/velox.json --out dist-second --installer --json
velox.exe inspect <archive> --json
```

The text-editor and folder-browser `init` commands additionally used
`--template text-editor` and `--template folder-browser`, respectively.
Native Go checks used `GOCACHE=<candidate-root>/go-cache-resumed` and
`VELOX_BUILT_HOST=<candidate-root>/clean-first-release/velox-windows-x64/velox-host.exe`:

```
go test ./tests/startup -run '^TestBuiltHostStartup$' -v -count=1 -timeout=3m
# VELOX_STARTUP_LIFECYCLE_RESULT=<candidate-root>/native-lifecycle.json
# VELOX_STARTUP_LIFECYCLE_REPETITIONS=3
go test ./tests/startup -run '^TestStartupLifecycleEvidence$' -v -count=1 -timeout=3m
```

| Template | App ZIP SHA-256 | Setup SHA-256 |
| --- | --- | --- |
| basic | `df59ed3e0de2c71b3b533c2429331a284ee0d8506892834f29ed7f8c6c3afd71` | `e434ef6ca6ed2105bc27e9398b32f70f432067e2bafddc0cbf940b545374ebff` |
| text-editor | `62f775920c0713b40e02f471300089a2b985437a4da5e1558576ef1f1020ec37` | `fe3f0a597af2614ba86130e5ed2b04505c747c198e02b6cfff8b84340cae8bc3` |
| folder-browser | `ea981412b5733fbe52dea9b86ae86b78ee0986f2bbec4a0028838d2661229b76` | `d38a94845cf175f248e088d3d8ee244f33169dbe43c3fd3638cca06aceec829c` |

## Draft Smoke Failure

`node scripts/installer-smoke.mjs --cli .cache/review-candidate-20261027-065cee6/clean-first-release/velox-windows-x64/velox.exe --example examples/file-notes`
passed its scripted checks but records `manualUIVerified: false`,
`actualDraftRestoreVerified: false` and `domReadyVerified: false`; it does not
establish manual UI operation, DOM readiness or real draft restoration.

`bun scripts/text-editor-draft-smoke.ts` used
`VELOX_DRAFT_RELEASE_DIR=.cache/review-candidate-20261027-065cee6/clean-first-release/velox-windows-x64`
and launched `velox.exe run --config <my-editor>/velox.json --json` with
`VELOX_DATA_DIR=<profile>` and a loopback `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS`
diagnostic port. The original run exited 1 (`draft-smoke.log`,
`draft-smoke-status.json`, `.cache/text-editor-draft-1793103912785/result.json`):
CDP `Input.insertText` left `#editor` at length 0, `#draft-state` at `No draft`,
status `Ready` and active element `editor`. A diagnostic retry with
`Page.bringToFront`, focus emulation and measured editor coordinates failed at
the editor-value precondition (`draft-smoke-focus-fixed.log`,
`draft-smoke-focus-fixed-status.json`, `.cache/text-editor-draft-1793103984682/result.json`).
Both host process trees were cleaned with `taskkill` (`cleanupExitCode: 0`).
Neither run reached restore or clear. The experimental script edits were
reverted; no shipping script or product fix is inferred.

Draft persistence and recovery are **not verified**. The failures neither
establish a draft-storage defect nor demonstrate OS keyboard or IME behavior.
Replacing the check with synthetic DOM value mutation would not be a native
typing pass. This failed mandatory check keeps beta held.

## Known Issue

Immediate readiness measured 7.2981501s in the startup test and
7,244.4123 to 8,617.9463ms across the three lifecycle pairs. The multi-second
relaunch delay remains unresolved; its cause is not established, and profile
locking is not confirmed as the cause.

## Boundaries

Skipped: manual file pickers and dialogs, IME and screen-reader checks, watch
and diagnostics follow-up, hosted CI and Go 1.26.0 parity, signing activation,
asset sealing, additional platforms and wider IPC limits, and public
promotion. Absence of these checks is not a pass. No push, tag, release,
hosted run or beta promotion follows from this receipt.
