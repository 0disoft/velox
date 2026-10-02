# Folder Browser

Dependency-free, read-only example (0.2.0) for Velox beta.18's `folder.read` and
`folder.readText` permissions. Uses `folder.select`, `folder.list`,
`folder.openText` and `folder.release`. There is no frontend bundle, network
request, saved token or write API.

Select one local folder. Refresh reads immediate names and kinds from its
identity-checked directory handle. Results use filesystem enumeration order,
not a sorted or stable snapshot. At most 128 immediate entries are considered
(one additional entry detects truncation), and JSON result bytes stay below
32 KiB. Reparse/offline/encrypted entries are excluded without following them;
`skipped` counts exclusions in the examined portion. `truncated` means more
entries or result bytes were omitted. No pagination or total-count claim is made.
Ordinary hard-linked file names can be listed; no contents or file grant is returned.

Click an immediate file name to read a UTF-8 preview (at most 2 MiB). Contents
are rendered literally in a read-only textarea, never as HTML, and are not saved.
Listing does not read contents automatically. Directories have no open action.
Names use the host's text-save basename grammar (at most 240 UTF-8 bytes);
unsupported/oversized/binary, encrypted/offline/reparse and multiply hard-linked
files fail without losing the folder connection. No extension is assumed safe.
The capability permits current immediate files, not a snapshot from selection
or listing. Refresh clears the preview; selection cancellation preserves it;
replacement/release clear both connection-dependent views. An expired target
clears both panes without retry or reopening a dialog.

Cancel preserves the previous connection. Select replaces it; release,
navigation, reload and shutdown revoke it. Directory replacement/deletion
invalidates the token. Folders can change between refreshes; this does not
monitor changes or implement recursive traversal. Local reads can still be
slow on a slow device/driver, despite entry and allocation bounds.

Build from the repository root using the assembled release CLI and an absolute
output path:

```powershell
$out = Join-Path (Get-Location) 'dist/examples/folder-browser'
.\dist\release\velox-windows-x64\velox.exe build --config examples/folder-browser/velox.json --out $out --json
```

Run the resulting `dev.velox.folderbrowser/dev.velox.folderbrowser.exe`, or
extract `dev.velox.folderbrowser.zip` before running. This does not install or
replace File Notes. Real native selection/cancellation are manual checks;
package startup alone is not proof that selection, enumeration or preview was
exercised. Manual check: select a disposable local folder, click a UTF-8 `.txt`
or `.md` file, verify the text, cancel reselection, then refresh and release.
