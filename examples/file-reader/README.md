# Text Viewer

This read-only example grants only `file.open`. Open invokes `file.openText`
with no parameters; a Windows file dialog supplies the selected file. Cancel
keeps the current view. Files are shown as plain text, never interpreted as HTML.

Build it with an updated Velox release:

```powershell
velox build --config examples/file-reader/velox.json --out ../../dist/examples/file-reader
```

Only one local regular UTF-8 file up to 2 MiB is accepted. UTF-8 BOM is removed
from displayed text. Invalid UTF-8, NUL bytes, network/device/stream paths,
reparse-point files and offline placeholders are refused. No full path, file
handle, or durable read grant is returned. There is no save or profile storage.

For a manual check, open a disposable UTF-8 document, then cancel a second
selection and confirm the previous view remains. Oversized or binary files
should report an error without replacing the displayed text. Native dialog
selection is a separate manual check, not inferred from a successful build.
