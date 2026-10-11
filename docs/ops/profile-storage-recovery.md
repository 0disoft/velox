# Profile Storage Recovery

## Preserve Current Text

When the editor reports `Draft recovery unavailable. Save changes to a file.`,
keep the window open and use Save or Save as. Canceling a destination keeps the
text unsaved; select a destination and wait for the file-save confirmation.
Native file saving is independent of IndexedDB draft storage. Cancel normal
window close if the current text has not reached a file. Forced termination
cannot recover text whose draft write never committed.

Current source text-editor starters retain this notice during typing, IME and
pending retries until a storage operation succeeds. File Notes keeps the notice
separate from the file-operation result. This also handles malformed storage
rejections without displaying their raw exception or blocking editing.
Existing generated applications need updated assets; a host update alone does
not replace their JavaScript. This source change is not a published release.

## Long Profile Paths

Velox uses an existing shorter Windows alias only after verifying that it refers
to the configured profile directory. Current source also creates a guarded
same-directory junction when the remaining local fixed-drive browser spelling
exceeds 120 UTF-16 units. Data is not moved or copied. The private root is
`%LOCALAPPDATA%\VeloxProfileLinks`; each entry is tied to the current user and
original directory identity. Existing mismatched roots, foreign/redirected
entries and linked/offline ancestors are refused. Failure stops startup with a
profile notice instead of retrying an unsafe long spelling. Remote/removable
profiles retain their previous path handling; this is not verified long-path
storage support for those drives. A long LocalAppData path can also prevent
preparing a short enough link. Contact the app author if preparation fails;
do not reset the profile or change the existing app's data directory.

The host holds directory guards through its run loop and releases them on
startup failure or loop exit. Published junctions remain because browser
subprocesses can outlive the GUI host. No automatic orphan sweep is performed.
Window state and single-instance locks continue to use the original profile.
An extended `\\?\` spelling alone does not ensure Chromium IndexedDB support.
The native diagnostic in VALIDATION.md reproduced failed writes to a new
218-character profile and failed reads from an existing one when path shortening
was bypassed. Returning to the existing alias read the original draft again.
The diagnostic did not disable NTFS 8.3 naming or prove every volume/runtime case.

The initial test-only [directory junction](https://learn.microsoft.com/en-us/windows/win32/fileio/hard-links-and-junctions)
to the same isolated profile passed write/read and read recovery without an 8.3
alias. Its mount-point buffer follows the
[Microsoft reparse structure](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/ns-ntifs-_reparse_data_buffer).
Subsequent manager tests cover ownership, concurrent acquisitions and lease
release. Actual Runtime.Open and built-host tests now verify write/relaunch-read
through the production fallback. These isolated local checks do not verify
installed profiles, and the change is not published. Later built-host tests
recover committed Korean/CRLF text after forcibly stopping the owned host, or
the host and its browser, then observing browser exit and reopening the same
profile. Another native check seeds a draft using a different short spelling
at the same trusted origin before reading it twice through Runtime.Open.
Neither test establishes power-loss durability or recovery of input whose
IndexedDB transaction had not completed. The production origin/database format
does not change; the fallback does not import data from other origins.

Do not delete a profile, remove its databases or change Windows volume settings
to dismiss this warning. A shorter profile location may be chosen for a new app
before user data exists. Changing `VELOX_DATA_DIR` on an existing app selects a
different profile; it does not migrate or recover the old draft. Existing-data
migration needs an explicit, verified procedure rather than a silent redirect.

## Developer Checks

`node scripts/editor-storage-failure-ui-smoke.mjs` generates a text-editor from
the source CLI and serves it alongside File Notes. It requires Playwright and
Microsoft Edge, and closes its own browser/server. Native file responses and
startup/write UnknownError failures are injected; no real picker or disk export
is claimed. Screenshots/results are left under `.cache`. The opt-in native
storage diagnostic separately uses real WebView2/IndexedDB and isolated profiles.
`VELOX_NATIVE_STORAGE_PROBE=1 go test -count=1 -run '^TestNativeRuntimeProfile' ./internal/webview2`
checks the actual runtime lifecycle and initialization failure. The built-host
guarded case uses `VELOX_STORAGE_PATH_TEST=1`, a freshly built `VELOX_BUILT_HOST`,
and `go test -count=1 -run '^TestBuiltHostIndexedDBLongPath/guarded-link-without-short-spelling$' ./tests/startup`.
Fixtures use their own LocalAppData roots and never change volume settings.

The separate apartment comparison builds `go test -c -o <test-host> ./cmd/velox-host`.
Set `VELOX_APARTMENT_PROBE_HOST` to that test executable and
`VELOX_APARTMENT_DIAGNOSTIC_RESULT` to a private evidence filename, then run
`go test -count=1 -run '^TestApartmentReleaseDiagnostic$' ./tests/startup`.
It compares one explicitly acquired STA reference with/without its matched
release, not all production apartment ownership. Safe-arm cleanup requires
observed exit timestamps for both native browsers. The final local comparison
still took about 6.2-6.4 s for browser exit and did not resolve the existing
ordinary relaunch delay. Never treat a missing lockfile or a closed error
observation channel as proof that a browser has exited.

`go test -count=1 -run '^TestBuiltHostGuardedStorageAfterAbruptExit$' ./tests/startup`
uses the same built-host/storage opt-in settings for committed-draft process
interruption checks. It stops only processes created or identified by that
fixture, never a user app or browser. Root/link cleanup waits for observed exit.
