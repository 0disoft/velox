# Text Writer

This example grants only `file.save` and calls `window.velox.saveText()`.
It stages at most 2 MiB of UTF-8 text in bounded messages and opens a native
Save as dialog for every save. No path or durable file grant is retained.
The editor keeps its text after cancellation or failure. It is not File Notes
and does not provide draft recovery or save-to-last-file behavior.

Build with `velox build --config examples/file-saver/velox.json --out <absolute-output-folder>`.
Relative output paths resolve from this example's project root, not the shell's
working directory.

For manual validation, use disposable files: cancel Save as (no file written),
save mixed Korean/English text, compare disk contents, then select the same
file again and cancel/accept overwrite confirmation. See
[ADR 0026](../../docs/adr/0026-selected-local-text-file-saving.md) for limits and recovery.
