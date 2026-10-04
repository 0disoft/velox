# File Notes

File Notes is a local UTF-8 Markdown and text editor using Velox's bounded
native file dialogs and IndexedDB for one recoverable draft. It requests only
`file.open`, `file.save` and `window.title`; it does not use browser File System Access pickers.

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

## Minimum Window Size

The manifest requests a minimum outer window of 720 x 520 in 96-DPI logical
units. The host scales it for current DPI and caps it to the monitor work area.
Older saved smaller rectangles are fitted on startup; no recovery data or
save connections are deleted. Minimize/maximize remain normal Windows actions.
A smaller screen may override this requested floor, so responsive layout
and scrolling remain necessary.

## Window title

The native caption shows the filename and app name, with a leading dirty marker
when edits are unsaved. Open, New, Save as and draft restoration update it.
Identical captions are deduplicated, including consecutive edits to an already
dirty document. App identity, permission prompts and tray tooltip stay unchanged.
If the native method is unavailable or denied, editing and saving continue;
the browser document title still updates.

## Keyboard actions

- `Ctrl+S`: Save, or Save as when no document-scoped target is connected.
- `Ctrl+Shift+S`: Save as.
- `Ctrl+O`: Open.
- `Ctrl+N`: New document.

These use the same buttons, native permissions, cancellation handling and
unsaved-change confirmation as pointer actions. Recognized shortcuts prevent
browser defaults, but do not dispatch while a file operation or discard dialog
is pending, or on repeated keydown. IME composition is left untouched, including
the Windows key-code 229 fallback. Matching uses physical key codes so Korean
input mode does not require Latin `event.key` values. Alt/AltGr, Meta and
unsupported Shift combinations are not intercepted. These are app-local keys,
not OS-wide shortcuts. Buttons expose matching `aria-keyshortcuts` metadata;
visible labels, tab order and layout remain unchanged.

## Find in document

`Ctrl+F` or the search icon opens a local search bar. Enter a literal,
case-sensitive query; Enter finds the next match, Shift+Enter the previous,
and Escape closes the bar and returns focus to the editor. Arrow buttons do
the same navigation and wrap at either end. The counter shows the selected
ordinal / total, or 0 / total after editing invalidates the selection.
Matches do not overlap. Regular expressions, replacement and case folding are
not supported. Search text is temporary, not part of the recovery draft.

Search uses native string operations with constant per-match storage, never
an array of all positions. Wrapped text is measured by browser layout in an
offscreen, accessibility-hidden mirror only when needed to reveal a result.
That mirror holds at most one copy of the current editor text and is removed
when search closes; it is not a background watcher. Large 2 MiB documents can
still require noticeable scanning/layout work, particularly when many matches
or long wrapped lines exist. No constant-latency guarantee is made.
Searching and moving selection do not change document text, dirty state or
the saved baseline. IME composition, repeated Enter, pending file work and
the discard dialog cannot navigate search results. Native dialog Escape is
left to the dialog. Bundled Lucide icon licenses are in `web/icons/LICENSE.txt`.

## Editor font

The editor area uses the bundled offline font "Velox Noto Sans KR" at 400 1rem/1.7,
with "Malgun Gothic" fallback. Font: `web/fonts/NotoSansKR.ttf`; license: `web/fonts/OFL.txt`.
It is a proportional text font, not for code column alignment. No engine font default is changed.

## Portable build

The repository build output is `dist/examples/file-notes/dev.velox.filenotes.zip`.
Extract the archive to a local folder, then run `dev.velox.filenotes.exe` from
the extracted app folder. The portable ZIP does not require installation.

The current example is File Notes 0.4.0, built with the unsigned Velox beta.19
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
