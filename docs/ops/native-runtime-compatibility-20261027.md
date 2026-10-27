# Native Runtime Compatibility: 2026-10-27

Local check of WebView2 Fixed Version 92.0.902.73, 111.0.1661.62 and
112.0.1722.68 against Velox `main`, with Evergreen 155.0.4283.62 as a control:
profile paths, Settings8, startup, lifecycle, security, draft recovery, File
Notes layout and native storage. Four local commits; no push, CI or release.
Receipt filenames below are relative to `.cache/webview2-compat-20261027/`.
The fixed source-host fixture SHA-256 is
`3979158f308a9cdee3d00e3e4079173d470db31f43449944f5772c754f1aedb4`.

## Profile path change (86ae186)

`internal/webview2/data_path_windows.go` hands WebView2 a plain DOS spelling of
the profile path. `prepareProfilePath` already budgets it against
`maxProfileLinkPath = 120` UTF-16 units before creating a junction. The new
`shortBrowserProfilePath` removes `\\?\` only when the plain spelling is absolute,
uses a drive-letter volume and passes an `os.SameFile` check, confirming
directory identity before choosing a spelling. Junction creation, native file I/O
and the pinned lease keep extended-length paths.

## WebView2 92 IndexedDB failure

The same physical profile passed all three draft phases as plain DOS and failed
the write phase with the extended `\\?\` prefix: `UnknownError`, "Internal error
opening backing store for indexedDB.open.", `storage` unavailable in a secure
context with the editor writable. Why 92 rejects the extended form while later
runtimes accept it was not established.

| 92 control | Result | Receipt |
| --- | --- | --- |
| Plain DOS, three phases | Pass | `draft-92-plain-control.log` |
| Extended prefix | Fail on write | `draft-92-extended-control.log` |
| Extended prefix, diagnostics | Fail, same storage error | `draft-92-diagnostics.log` |

## Settings8

Native witness, real window: running image, version string and
`ICoreWebView2Settings8` (`SetReputationCheckingRequired`) from a `-d=checkptr=2`
build.

| Version | Settings8 | Result | Receipt |
| --- | --- | --- | --- |
| 92.0.902.73 | unavailable | exit 0 | `native-92.0.902.73.log` |
| 111.0.1661.62 | unavailable | exit 0 | `native-111.0.1661.62.log` |
| 112.0.1722.68 | available | exit 0 | `native-112.0.1722.68.log` |

## Startup, lifecycle and security

92 and 112 passed all five `TestBuiltHostStartup` subcases; 111 passed lifecycle
and security policy only (`startup-fixed-*.log`). Two 92 failures were harness
defects fixed in c4f23e6: the missing-runtime case inherited a loader override
that beat the explicit folder and hung for 108s (`startup-92.log`), so
`testUnavailableRuntime` sets `WEBVIEW2_BROWSER_EXECUTABLE_FOLDER` to the same
missing folder and adds a 15s context timeout with a 2s `WaitDelay`; and
`security.js` requested notifications only, raising no permission event on 92, so
the fixture now waits for a geolocation denial. Why 92 emits no notification
event was not identified inside the browser.

One lifecycle sample each succeeded, first/immediate: 92 1585.36/1279.99ms,
111 694.72/7024.46ms, 112 414.77/438.31ms. The 111 ~7s gap is a remaining
performance difference in a Settings8-less environment; no same-condition
SmartScreen-off control was run, so that cause is unproven. Single samples make
these descriptive only; the product change adds no unsupported flags or browser
termination.

Follow-up: [111 relaunch investigation](webview2-111-relaunch-20261027.md)
isolates the delay with an A/B/A experiment and records the guarded legacy
fix: immediate-relaunch median 7092.05ms to 603.68ms across the recorded samples.

## Draft recovery

With the final harness all three phases pass on 92, 111, 112 and 155:
`draft-final-92.0.902.73.log`, `draft-fixed-111.0.1661.62-keyboard.log`,
`draft-final-112.0.1722.68.log`, `draft-final-evergreen.log`. Earlier 111
attempts lost the whole input with `storage` available (`...-foreground.log`).
Bulk `Input.insertText` inserted nothing; a bulk `Input.dispatchKeyEvent`
carrying all 28 characters as one `text` value returned "Invalid 'text'
parameter" (`...-char.log`). The final script uses `Page.bringToFront` and one
`Input.dispatchKeyEvent` of type `char` per character, with `\r` for the newline.
The empty editor fits harness input loss, not a product defect; the run sends
synthetic CDP input, not physical keyboard or IME checks.

## File Notes layout on 92

`[hidden]` works on 92; the `:has()` rule that switches the grid to two rows is
what 92 ignores, so the textarea auto-placed into the first row of the three-row
grid and shrank to 104.4px in a 427.2px shell, 317.2px opened
(`file-notes-layout-before-92.log`). Explicit `grid-row` 1, 2 and 3 for find bar,
editor and help line give 367.2px hidden and 317.2px opened on 92, 112 and 155
(`file-notes-layout-after-*.log`; commit 8a72f64). The probe copies real File
Notes web assets into a generated Text Editor project; native file dialogs and
per-API permissions were not exercised.

## Native storage and verification

`native-storage-fixed-results.json` records exit 0 for 92, 112 and Evergreen,
each passing four opt-in `-d=checkptr=2` tests: profile storage, existing draft,
terminate storage and startup failure lease release. The fixture starts from a
246-unit profile path and a 103-character retained junction spelling (99 after
removing the prefix for the browser), seeds a Korean CRLF draft
through a junction with a different same-directory spelling, and reads it back
through the production path. `vet-final.log` is empty with exit 0; the full Go
suite finished with exit 0 across 48 packages (`all-tests-final-result.json`).

## Provenance and limits

Each CAB and the extracted `msedgewebview2.exe` carry a Valid Microsoft
Corporation signature. The source URL for the archived CABs was not recorded, so
these are Microsoft-signed builds of unknown download provenance; this is not an
official-download claim. Runtimes were extracted into isolated fixed folders with
dedicated profiles; the Evergreen install was not changed and older runtimes are
not recommended for normal use. The 92.0.902.49 floor was not tested directly;
only 92.0.902.73 was, and the supported version line is unchanged. Version, API,
database, dependency and CI runner configuration are unchanged.

## Reproduction

```powershell
$env:WEBVIEW2_BROWSER_EXECUTABLE_FOLDER = '<runtime>\Microsoft.WebView2.FixedVersionRuntime.92.0.902.73.x64'
$env:VELOX_DRAFT_RELEASE_DIR = '<bundle>\velox-windows-x64'
$env:VELOX_DRAFT_EXPECT_BROWSER_VERSION = '92.0.902.73'
bun scripts/text-editor-draft-smoke.ts
$env:VELOX_BUILT_HOST = '<bundle>\velox-windows-x64\velox-host.exe'
go test ./tests/startup -run TestBuiltHostStartup -count=1 -v
$env:VELOX_NATIVE_STORAGE_PROBE = '1'
go test -gcflags=all=-d=checkptr=2 ./internal/webview2 -run TestNativeRuntime -count=1 -v
```

Microsoft documents [Fixed Version distribution](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution)
for apps that pin a browser build, and the
[WebView2 Win32 reference](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/webview2-idl)
for the environment, settings and user-data-folder interfaces used here.
Implementation commits: `7da29a2` (witness), `c4f23e6` (harness), `86ae186`
(profile fix) and `8a72f64` (CSS), all local on `main`.
