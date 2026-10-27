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
| Text Editor draft smoke | **Fail at this receipt** (superseded by the follow-up): no text entered; both attempts stopped before restore and clear | `draft-smoke.log`, `draft-smoke-focus-fixed.log`, `draft-smoke-status.json` |

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

Draft persistence and recovery were **not verified at this receipt**. The
failures neither establish a draft-storage defect nor demonstrate OS keyboard
or IME behavior. Replacing the check with synthetic DOM value mutation would
not be a native typing pass. This failed mandatory check kept beta held; the
follow-up below supersedes the result for the local harness.

## Follow-up: 2026-10-27

`258a96f` fixes the draft smoke harness, not the product. It launches the
host with `windowsHide: false`, waits for the window and `#editor` to be
visible, focused and active, waits two animation frames, sends CDP
`Input.insertText`, and asserts the exact editor value before continuing.
Three consecutive native WebView2 runs passed the three stages: write;
relaunch, restore and clear; relaunch after clear with no resurrection. Every
receipt reports `passed: true`, close exit 0 and cleanup exit 0:

- `.cache/text-editor-draft-1793104762829/result.json`
- `.cache/text-editor-draft-1793104986123/result.json`
- `.cache/text-editor-draft-1793105012320/result.json`

A final run used a private bundle with the rebuilt host and the original
candidate CLI and setup executables:
`.cache/text-editor-draft-1793106602528/result.json`. It passed the same
three stages without forced termination.

These are CDP-driven native runs against an isolated profile. They are not OS
IME, manual keyboard or manual UI evidence. The installer smoke still records
`manualUIVerified: false` and `actualDraftRestoreVerified: false`.

### Relaunch readiness

The baseline and diagnostic runs used the original candidate host. The product
run used the rebuilt host. Both used the same app-specific virtual origin,
WebView2 `155.0.4283.62`, Go `1.27.1` and three same-profile pairs. With
the default reputation setting, immediate-host ready times were
6,644.45ms, 6,747.0031ms and 6,855.6814ms. The first host's browser process
exited 6,084.3361, 6,210.7842 and 6,328.1335ms after that host. A diagnostic
`msSmartScreenProtection` disable, used only for comparison and not by the
product, measured 417.4746ms to 479.2941ms. The product path below measured
419.1895ms to 454.3207ms.

| Run | Immediate host ready (ms) | Browser exit after immediate host (ms) | Profile removal (ms) |
| --- | --- | --- | --- |
| Default reputation setting | 6,644.45; 6,747.0031; 6,855.6814 | 6,176.6002; 6,183.4495; 6,367.5748 | 6,282.9814; 6,284.4441; 6,456.1052 |
| Diagnostic SmartScreen off | 417.4746; 479.2941; 478.8425 | 60.3695; 65.0244; 77.4392 | 100.0226; 103.8589; 140.9768 |
| Product Settings8 path | 443.592; 419.1895; 454.3207 | 61.2008; 63.9437; 67.5260 | 103.0381; 108.3971; 112.8394 |

The product path's first-host browser exits were 74.5098, 72.6771 and
78.8355ms.

The evidence supports SmartScreen-related browser work in this environment
keeping the old browser alive and delaying the same-profile controller. It
does not establish an abnormal COM leak. Netlog showed SmartScreen, certificate
and network activity, but did not establish TLS or DoH individually as the
root cause. Shutdown COM work was fast, and an initial message-pump and GPU
comparison did not explain the delay.

The product host sets `DisableReputationChecking: true`; the fork default is
false. Enabling it requires a `NavigationAllowed` callback plus
`DenyFrames`, `DenyNewWindows` and `DenyDownloads`. The fork queries
`ICoreWebView2Settings8` from SDK 1.0.1722.45 (IID
`9E6B0E8F-86AD-4E81-8147-A9B5EDB68650`; `Get` and `Put` occupy slots 35
and 36 after the Settings1-7 methods) and sets
`IsReputationCheckingRequired` to false before the first navigation.
`E_NOINTERFACE` leaves the default enabled and does not raise the supported
floor of 92. Other HRESULT failures fail initialization. Successful queries
release the Settings8 pointer; the initial settings reference is released on
both success and failure. The lifecycle receipt is
`.cache/draft-lifecycle-fix-20261027/settings8-lifecycle-three.json`.

The app-specific `https://<hash>.app.invalid` origin, profile, permissions,
IPC and navigation restrictions are unchanged. The omitted protection is
SmartScreen web navigation and download reputation in that app-specific
WebView2 profile. Windows and Edge-wide SmartScreen, TLS validation,
antivirus and Authenticode warnings are unchanged. Local assets remain
untrusted and are not tamper-proof. This receipt makes no separate subresource
protection claim. Remote navigation or downloads would require a new policy
review.

### Rebuilt host and bundle

The product change is commit `acf27a3`, after source candidate `739490b`.
The rebuilt host is 4,937,728 bytes, 3,584 bytes larger than the original
candidate host at 4,934,144 bytes.

| Artifact | Bytes | SHA-256 |
| --- | --- | --- |
| `settings8-host.exe` | 4,937,728 | `57052a425ff10454e198898098dd3c1d4a93e4af7d3296690a6be3b910a904a9` |
| Private `velox-windows-x64.zip` | 6,950,756 | `e9e2e9e4ef9e32e6eef80f813f6ab03555f6dd762a55229f4466c29f3dd66904` |

The bundle under `.cache/draft-lifecycle-fix-20261027/settings8-release/`
reuses the original candidate `velox.exe` and `velox-setup.exe` and replaces
only the host. It is a private verification bundle, not a release payload, and
does not replace the candidate table above.

Files named without a directory in the table below are under
`.cache/draft-lifecycle-fix-20261027/`.

| Follow-up check | Result | Evidence |
| --- | --- | --- |
| Source tests | Pass: `go test ./... -count=1 -timeout=5m` exit 0; 44 packages `ok`, 10 no-test entries, 0 failures | `source-tests.log` |
| Fork tests | Pass: `go test github.com/jchv/go-webview2/... -count=1`; 4 packages | `fork-tests.log` |
| Rebuilt host startup | Pass: 5/5 subcases in 3.87s | `settings8-startup.log` |
| New host lifecycle | Pass: three pairs | `settings8-lifecycle-three.json` |
| Private bundle assembly | Pass: `cmd/velox-release`; host replaced, CLI and setup reused | `settings8-release.json` |
| Bundle draft smoke | Pass: `bun scripts/text-editor-draft-smoke.ts`; three stages, no forced exit | `.cache/text-editor-draft-1793106602528/result.json` |
| Product vet | Pass: `go vet ./internal/webview2 ./tests/startup` exit 0, empty output | `product-vet.log` |
| Combined vet | **Fail at this receipt**: exit 1 from eight existing `possible misuse of unsafe.Pointer` warnings in unchanged fork sections; resolved in the linked follow-up | `vet.log` |

Combined vet remains a failed historical result. The eight warnings were
subsequently resolved; see the [pointer audit](native-pointer-audit-20261027.md)
for checkptr reproduction, the Go source compatibility change and passing vet.
Source `go test ./...` ran without
`VELOX_BUILT_HOST`, so the opt-in native built-host cases were skipped there.
The rebuilt-host `TestBuiltHostStartup` and three-pair lifecycle results above
come from separate runs with `VELOX_BUILT_HOST` set. These checks are local
and do not change the candidate identity. The beta hold, public promotion and
release decision remain unchanged. This follow-up clears only the local
draft-smoke and same-profile relaunch blockers for this evidence set.

## Known Issue at This Receipt

Immediate readiness measured 7.2981501s in the startup test and
7,244.4123 to 8,617.9463ms across the three lifecycle pairs. The multi-second
relaunch delay was unresolved at this receipt; its cause was not established,
and profile locking was not confirmed as the cause. The follow-up records the
local Settings8 comparison and does not make a release decision.

## Boundaries

Skipped: manual file pickers and dialogs, IME and screen-reader checks, watch
and diagnostics follow-up, hosted CI and Go 1.26.0 parity, signing activation,
asset sealing, additional platforms and wider IPC limits, and public
promotion. Absence of these checks is not a pass. No push, tag, release,
hosted run or beta promotion follows from this receipt.
