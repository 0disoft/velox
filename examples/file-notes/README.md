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
Unedited files retain their original line endings, including mixed endings.
After editing, newlines use the first original line-ending style; New resets
to LF. Drafts preserve this serialized text. Saving checks the resulting UTF-8
size before requesting native file access, without truncating the editor.
Names above the native 240-byte suggestion limit use `Untitled.md` in the
picker; canceling leaves the original document name unchanged.

Drafts retain names, text and saved-text baselines, never native save tokens or
browser handles. Old schema-v1 drafts remain readable, but their browser handles
are ignored. After restart or reload, recovered text requires fresh Save as
selection. Native saving does not rely on WebView2's stored FileReadWrite
permission; the system-menu File access recovery action is not needed for it.

On supported Velox hosts, normal native window close uses the browser's
`beforeunload` confirmation. Cancel keeps the current document open even when
draft recovery is unavailable. Forced process termination is not protected by
that confirmation, and recovery still requires a completed IndexedDB write.
Startup recovery and pending native file operations also request close
confirmation, even for a clean document. A blocked draft database reports
recovery as unavailable and unlocks the editor; it does not prevent native
file operations. Draft connections close when a database version change is
requested, including connections that open after an earlier blocked failure.
The failure notice asks the user to save changes to a file. It remains separate
from successful native-file save results, so a failed draft write does not turn
a completed file save into a reported failure. See
[profile storage recovery](../../docs/ops/profile-storage-recovery.md), including
the known long-profile fallback limitation when no usable shorter alias exists.
Draft persistence keeps at most one write in progress and one latest waiting
snapshot, skips intermediate IME composition text, and ignores status updates
from obsolete writes. Each text field is bounded to 2 MiB of UTF-8 before
writing; an oversized draft leaves the previous stored draft intact and the
current editor text available to correct or copy.

Loading requires supported schema-v1 text within the 2 MiB UTF-8 byte limit;
string saved-text baselines have the same limit. Legacy v1 metadata still uses
the model's existing normalization. Unsupported formats, malformed text and
oversized stored text are left unchanged with an explicit notice. Each automatic
save checks the existing row in the same readwrite transaction as its put, so
a newer incompatible draft appearing after startup is not overwritten. Editing
and native file export remain available, but New does not discard the protected
stored row. No migration is attempted; the explicit storage clear helper is
separate and is not called by the New button.

Because a buffer that looks clean can still have an older dirty draft as its
last committed write, the close guard requests confirmation until the serial
draft queue and any IME composition have settled; it returns without a prompt
once nothing is pending, so a clean document is not permanently blocked from
closing. Save, New and a successful Open enqueue an immediate commit, typing
keeps a short debounce, and the status line shows `Draft pending` while a
commit is queued or in flight. The `Draft recovery unavailable` guidance is not
replaced by that pending state; it stays visible until the latest draft
revision is actually committed, so a failed draft keeps showing its warning
from the first keystroke through IME composition until the next successful
write. Editor and file buttons are not disabled by pending IndexedDB work. A
draft write that finally fails keeps its own
`Draft recovery unavailable` notice and does not turn a completed native file
save into a reported failure. This guard covers the browser close path only
and does not promise recovery after an accepted close, a forced termination or
power loss.

Native saving remains subject to Windows access controls and Velox's file
policy. Cancellation, manifest permission denial, expired connections and
external-change conflicts have distinct status messages. Errors preserve the
editor buffer and its unsaved baseline. Recovery errors can follow a partially
completed replacement; do not treat an error as proof that disk bytes are unchanged.

## Development Checks

Run `bun test examples/file-notes` for the model, search, draft storage and
native-response regression cases. `node scripts/file-notes-ui-smoke.mjs`
requires Playwright and installed Microsoft Edge; `NODE_PATH` can point to an
existing development installation without adding consumer dependencies.
The smoke uses real IndexedDB, mocked native file responses and an injected
blocked-open response. It checks recovery, line endings, keyboard search,
empty-search close, local font loading and desktop/narrow layouts, and writes
screenshots plus its result under `.cache`. It closes its own browser and
local HTTP server, and does not read or change the installed app's profile.
This does not replace native picker, Windows window-close or installer tests.
`node scripts/file-notes-stats-bench.mjs` compares the old allocation-heavy
statistics function with the current single pass on bounded 2 MiB inputs.
Its medians describe this local function benchmark, not native startup or
end-to-end input latency; there is no timing threshold in the regression tests.

## Minimum Window Size

The manifest requests a minimum outer window of 720 x 520 in 96-DPI logical
units. The host scales it for current DPI and caps it to the monitor work area.
Older saved smaller rectangles are fitted on startup; no recovery data or
save connections are deleted. Minimize/maximize remain normal Windows actions.
A smaller screen may override this requested floor, so responsive layout
and scrolling remain necessary.

## System theme

The manifest requests `window.followSystemTheme: true`. On Windows 11 build
22000 and newer the native title bar follows the current Windows app light or
dark setting; older or unsupported builds keep the default title bar (ADR 0036).
The application CSS does not depend on that flag: it follows
`prefers-color-scheme` for light and dark and `forced-colors` for high
contrast, updating live without a reload. This adds no dependency, native
permission, timer, or change to saving and draft recovery.

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

The current example is File Notes 0.4.0, built with the unsigned Velox beta.20
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
