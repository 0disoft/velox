# File Notes

File Notes is a local UTF-8 Markdown and text editor using Velox's bounded
native file dialogs and IndexedDB for one recoverable draft. It requests only
`file.open` and `file.save`; it does not use browser File System Access pickers.

The application has no network request, frontend package, bundler, or generated
binding. The host accepts only explicitly selected local regular UTF-8 files
up to 2 MiB under ADRs 0025-0027. Remote drives and linked paths are rejected.
The editor never receives or stores a filesystem path and asks before discarding
unsaved changes.

Save as selects and writes a file, then connects subsequent Save operations to
that file for this document session. Save checks the previous file identity and
contents; an external modification, replacement or deletion blocks the write
without an automatic retry or force-overwrite. Cancellation keeps the previous
connection. New and successful Open release it. Open is read-only: its first
Save still requires a native Save as selection and overwrite confirmation.

Drafts retain names, text and saved-text baselines, never native save tokens or
browser handles. Old schema-v1 drafts remain readable, but their browser handles
are ignored. After restart or reload, recovered text requires fresh Save as
selection. Native saving does not rely on WebView2's stored FileReadWrite
permission; the system-menu File access recovery action is not needed for it.

On supported Velox hosts, normal native window close uses the browser's
`beforeunload` confirmation. Cancel keeps the current document open even when
draft recovery is unavailable. Forced process termination is not protected by
that confirmation, and recovery still requires a completed IndexedDB write.

Native saving remains subject to Windows access controls and Velox's file
policy. Cancellation, manifest permission denial, expired connections and
external-change conflicts have distinct status messages. Errors preserve the
editor buffer and its unsaved baseline. Recovery errors can follow a partially
completed replacement; do not treat an error as proof that disk bytes are unchanged.

## Editor font

The editor area uses the bundled offline font "Velox Noto Sans KR" at 400 1rem/1.7,
with "Malgun Gothic" fallback. Font: `web/fonts/NotoSansKR.ttf`; license: `web/fonts/OFL.txt`.
It is a proportional text font, not for code column alignment. No engine font default is changed.

## Portable build

The repository build output is `dist/examples/file-notes/dev.velox.filenotes.zip`.
Extract the archive to a local folder, then run `dev.velox.filenotes.exe` from
the extracted app folder. The portable ZIP does not require installation.

The current example is File Notes 0.2.0, built with the unsigned Velox beta.15
host and the bundled Noto Sans KR font. It enables window-state restoration,
single-instance activation, and the host-owned tray menu. Closing the window
still exits normally; hiding it requires the tray's Hide window command.

An installer-enabled release can also build
`dist/examples/file-notes/dev.velox.filenotes-setup.exe` with
`velox build --installer`. The optional Setup installs for the current user;
building it alone does not install or replace an installed app.

User documents must be explicitly selected through the editor open/save
gestures. The application draft/profile is managed by WebView2 and is not
bundled in the ZIP.
