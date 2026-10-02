# Text Writer

This example grants only `file.save`. Save as calls `window.velox.saveTextAs()`
and connects one selected target for this document; subsequent Save calls use
`saveTextTo()` without another picker. The initial Save also uses Save as.
It stages at most 2 MiB of UTF-8 text in bounded messages. No path or durable
file grant is returned. New revokes the target and asks before discarding edits.
Reload or exit also revokes it. The editor keeps its text on cancellation,
external-change conflicts and other errors. It is not File Notes and does not
provide draft recovery or restart-persistent file access.

Build with `velox build --config examples/file-saver/velox.json --out <absolute-output-folder>`.
Relative output paths resolve from this example's project root, not the shell's
working directory.

For manual validation, use disposable files: cancel Save as (no file written),
save mixed Korean/English text, edit and Save without a picker, compare disk
contents, then select the same file via Save as and cancel/accept overwrite
confirmation. Edit the saved file externally and confirm Save returns
`FILE_CHANGED` without losing either editor buffer or external contents. New
must require a fresh selection on the next Save. See
[ADR 0026](../../docs/adr/0026-selected-local-text-file-saving.md) for replacement
recovery and [ADR 0027](../../docs/adr/0027-document-scoped-save-target.md) for reuse.
