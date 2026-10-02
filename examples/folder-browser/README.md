# Folder Browser

Dependency-free, read-only example for Velox beta.17's `folder.read` permission.
Uses only `folder.select`, `folder.list` and `folder.release`. There is no
frontend bundle, network request, saved token, file-content access or write API.

Select one local folder. Refresh reads immediate names and kinds from its
identity-checked directory handle. Results use filesystem enumeration order,
not a sorted or stable snapshot. At most 128 immediate entries are considered
(one additional entry detects truncation), and JSON result bytes stay below
32 KiB. Reparse/offline/encrypted entries are excluded without following them;
`skipped` counts exclusions in the examined portion. `truncated` means more
entries or result bytes were omitted. No pagination or total-count claim is made.
Ordinary hard-linked file names can be listed; no contents or file grant is returned.

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
package startup alone is not proof that selection or enumeration was exercised.
