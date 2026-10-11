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
to the configured profile directory. It does not create an alias or move data.
An extended `\\?\` spelling alone does not ensure Chromium IndexedDB support.
The native diagnostic in VALIDATION.md reproduced failed writes to a new
218-character profile and failed reads from an existing one when path shortening
was bypassed. Returning to the existing alias read the original draft again.
The diagnostic did not disable NTFS 8.3 naming or prove every volume/runtime case.

A test-only [directory junction](https://learn.microsoft.com/en-us/windows/win32/fileio/hard-links-and-junctions)
to the same isolated profile passed write/read and read recovery without an 8.3
alias. Its mount-point buffer follows the
[Microsoft reparse structure](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/ns-ntifs-_reparse_data_buffer).
The current host does not create such links. Product use needs verified link
ownership, stable concurrent-profile identity and shutdown/cleanup behavior;
the fixture experiment alone does not satisfy those requirements.

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
