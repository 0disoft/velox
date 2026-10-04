# IPC v1

- Status: Active
- Owner: Runtime host

## Public Surface

The host injects one frozen application-facing object into the trusted top-level
document:

```js
const info = await window.velox.invoke("app.getInfo");
```

`window.velox` and `window.velox.invoke` are non-configurable, non-writable, and
frozen. The transport binding named `__veloxInvoke` is internal and is not a
supported application API. Calling it directly does not bypass native origin,
protocol, limit, method, or permission checks.

## Request and Response

Requests use the `velox.ipc/v1` logical contract represented by
`schema/ipc-v1.schema.json`:

```json
{"v":1,"id":1,"method":"app.getInfo","params":{}}
```

Successful and failed responses preserve the request identifier:

```json
{"v":1,"id":1,"ok":true,"result":{"id":"dev.example.app","name":"Example","version":"1.0.0","platform":"windows"}}
```

```json
{"v":1,"id":1,"ok":false,"error":{"code":"PERMISSION_DENIED","message":"The native method permission is not granted."}}
```

Request identifiers are unsigned 32-bit integers greater than zero. Parameters
must be objects. Unknown fields, duplicate object keys, malformed JSON, and
unsupported versions fail before method dispatch.

## Limits

- Maximum WebView message and decoded native request: 64 KiB.
- Maximum JSON nesting depth: 16 composite levels.
- Maximum concurrent requests: 64.
- Duplicate in-flight request identifiers are rejected.
- New requests are rejected after shutdown begins.

The transport checks the UTF-8 size of the complete serialized message before
posting it and rejects oversized calls with `PAYLOAD_TOO_LARGE` without retaining
a pending request. Native size checks remain authoritative. Invalid parameters
and unsupported versions preserve an unambiguously decoded request identifier;
malformed or ambiguous envelopes may use identifier zero.

The JavaScript bridge and native dispatcher both enforce the concurrent-request
limit. Native enforcement remains authoritative when application code calls the
internal transport binding directly.

Except for the deferred selected-text reader below, the Windows transport
invokes native bindings synchronously on WebView2's UI/COM event thread.
Multiple unresolved JavaScript promises therefore do not
run Win32 window operations in parallel. The dispatcher still protects its
shutdown and in-flight bookkeeping so direct tests and any future transport
adapter must preserve the same ownership invariant.

Shutdown rejects newly dispatched requests with `SHUTTING_DOWN`; it does not
cancel a native call that already started. `window.close` schedules teardown
after its successful response, but this is not a guarantee that all outstanding
JavaScript promises settle before the page closes. Responses queued when native
window destruction begins are discarded, and no application code runs after its
document is destroyed. Applications must persist required state before asking
the window to close, not in a pending native-response continuation.

## Methods

| Method | Permission | Parameters | Result |
| --- | --- | --- | --- |
| `app.getInfo` | `app.info` | `{}` | application ID, name, version, and platform |
| `window.getState` | `window.basic` | `{}` | `normal`, `minimized`, or `maximized` |
| `window.minimize` | `window.basic` | `{}` | `null` |
| `window.maximize` | `window.basic` | `{}` | `null` |
| `window.restore` | `window.basic` | `{}` | `null` |
| `window.close` | `window.basic` | `{}` | `null` before deferred shutdown |
| `window.setTitle` | `window.title` | `{title}` | `null` after native title update |
| `window.requestAttention` | `window.attention` | `{}` or `{count: 1..5}` | `null` after bounded taskbar attention request |
| `window.cancelAttention` | `window.attention` | `{}` | `null` after stopping attention |
| `window.setProgress` | `window.progress` | `{state, value?}` (see below) | `null` when accepted, possibly cached before taskbar readiness |
| `notification.show` | `notification.show` | `{kind, message}` | `null` when the shell accepts a transient tray balloon |
| `external.open` | `external.open` | `{"url":"https://example.com/"}` | `{"queued":true}` before native confirmation |
| `clipboard.writeText` | `clipboard.write` | `{text}` | `null` after native Unicode text write |
| `clipboard.readText` | `clipboard.read` | `{}` | `{cancelled: true}` or `{cancelled: false, text}` after native approval |
| `file.openText` | `file.open` | `{}` | selected UTF-8 text or a cancellation result after the native dialog |
| `file.beginSave` | `file.save` | `{name, bytes}` | `{token}` for one bounded text upload |
| `file.appendSave` | `file.save` | `{token, offset, text}` | `{bytes}` received so far; byte offset must match |
| `file.commitSave` | `file.save` | `{token}` | `{cancelled, name, bytes}` after native save selection and disk commit |
| `file.cancelSave` | `file.save` | `{token}` | `null`; discard staged text without file access |
| `file.commitSaveAs` | `file.save` | `{token}` | native selection/save result with a document-scoped `target` on success |
| `file.commitSaveTo` | `file.save` | `{token, target}` | save result after validating and reusing the connected target |
| `file.releaseSaveTarget` | `file.save` | `{target}` | `null`; revoke that target without file access |
| `folder.select` | `folder.read` | `{}` | native folder selection, basename and document-scoped target; cancellation keeps the prior target |
| `folder.release` | `folder.read` | `{target}` | `null`; revoke that exact folder target |
| `folder.list` | `folder.read` | `{target}` | bounded immediate names/kinds, truncation and examined exclusion counts |
| `folder.openText` | `folder.read` + `folder.readText` | `{target, name}` | immediate UTF-8 file contents, basename and byte count; no write grant |

`window.maximize` returns redacted `NATIVE_OPERATION_FAILED` when the manifest
sets `window.resizable: false`. Basic permission does not override this sizing
policy. Minimize, restore and close retain their existing behavior.

The method table is a closed switch. Reflection is confined to the private
WebView transport adapter and cannot select a product method dynamically.

### Taskbar Progress

[ADR 0037](../adr/0037-taskbar-progress.md) adds
`window.velox.invoke("window.setProgress", params)` under independent,
default-disabled `window.progress`. It requires no basic, title or attention
permission and grants none of those operations.

- `{state: "none"}` clears progress; `{state: "indeterminate"}` requests
  activity without a percentage. Neither accepts a `value` field.
- `{state: "normal", value: 50}` requests determinate progress. `error` and
  `paused` use the same required integer `value` range, 0..100.

Unknown fields/states, missing required values, fractions, null and out-of-range
values return `INVALID_PARAMS`. Permission denial and shutdown retain the
existing errors. Native failures use redacted `NATIVE_OPERATION_FAILED`,
never a frontend HRESULT. A successful `null` acknowledges acceptance, including
latest-state caching before `TaskbarButtonCreated`; it does not confirm pixels
or report later shell restoration results.

Calls use the existing UI/COM thread. Permission-absent windows install no
progress machinery and make no progress COM calls. An opted-in `BeforeShow`
hook attaches the handler before initial display. No `ITaskbarList3` method is
called until the readiness message. Non-`none` progress then lazily creates
and initializes COM; determinate updates set the value before the state and
deduplicate successfully applied repeats. A recreated taskbar button restores
the latest request through a fresh COM object. Destruction clears best effort
and releases exactly once even under reentry.

Apps must clear completed/cancelled progress with `none` and keep an in-window
indication: high contrast or taskbar grouping can suppress their taskbar cue.
No polling, timer, worker, new dependency, file or persistent state is added.
Existing origin, wire limits and shutdown rules apply.

### Window Attention

`window.requestAttention({count})` and `window.cancelAttention({})` require
independent, default-disabled `window.attention`. Basic controls and title
permissions do not grant attention, nor does attention grant those methods.
Count is an optional JSON integer from 1 to 5, default 3. Unknown fields,
null, nonintegers and out-of-range values are `INVALID_PARAMS`. Cancel accepts
no fields. Missing/unavailable windows return redacted `NATIVE_OPERATION_FAILED`.

Use only bounded taskbar-button flashing at the Windows default blink rate,
without caption flashing or continuous timer flags. A request for the current
foreground window is a no-op. Cancel uses FLASHW_STOP. Neither operation shows,
restores, focuses or activates a window. Hidden-to-tray windows may have no
taskbar button; shell/accessibility settings determine the actual visual cue.
Success means the host issued the request, not that flashing was observed.
Repeated calls may restart flashing; the count bound applies per request.

Both methods run on the existing UI/COM thread. No host timer, worker, listener,
stored state or dependency is added. FlashWindowEx's BOOL is the previous
active state, not a success flag; validity is checked before invoking it.
See [FlashWindowEx](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-flashwindowex)
and [FLASHWINFO](https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-flashwinfo).
Existing origin, wire limits and shutdown rejection remain authoritative.

### Tray Notifications

[ADR 0038](../adr/0038-tray-notifications.md) adds
`window.velox.invoke("notification.show", {kind, message})` under independent,
default-disabled `notification.show`. `window.tray` neither grants nor
requires it, and it grants no other window or tray control.

- `kind` is required and one of `info`, `warning`, or `error`.
- `message` is required, must contain at least one non-whitespace character, is
  bounded to 255 UTF-16 code units, and may not contain NUL, DEL or another C0/C1
  control character except line feed and tab.

Only those two fields are accepted; unknown fields, missing/wrong types,
whitespace-only or oversized messages, and disallowed controls return
`INVALID_PARAMS`. Permission denial and shutdown keep the existing errors.

The manifest must set `window.tray: true` and the tray icon must be currently
registered; otherwise the method returns redacted `NATIVE_OPERATION_FAILED` and
installs no icon. A successful `null` means the Windows shell accepted a
transient request, not that a balloon was seen.

The host reuses the registered icon with `NIM_MODIFY` and a transient
`NOTIFYICONDATAW` copy using `NIF_INFO | NIF_REALTIME`, the manifest app name
as a fixed title truncated to 63 UTF-16 code units, and
`NIIF_NOSOUND | NIIF_RESPECT_QUIET_TIME` with the requested kind. The body is
not stored and is not replayed after an Explorer restart. A
`NIN_BALLOONUSERCLICK` (0x405) reveals the existing window through the normal
modal-owner and shutdown gates, like the tray Open command.

This adds no dependency, timer, worker, polling, file, persisted state, version
change, or toast registration. Windows notification settings can suppress the
balloon. Existing origin, wire limits and shutdown rejection remain
authoritative. See
[NOTIFYICONDATAW](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-notifyicondataw)
and
[Shell_NotifyIconW](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shell_notifyiconw).

### Window Title

`window.setTitle({title})` requires the independent, default-disabled
`window.title` permission, not `window.basic`. Accept only a title string
up to 512 UTF-8 bytes without Unicode control characters. Empty string
restores the manifest app name. Invalid fields/types/control characters return
`INVALID_PARAMS`; oversized titles return `PAYLOAD_TOO_LARGE`; native failures
return redacted `NATIVE_OPERATION_FAILED`.

The call runs synchronously on the existing UI/COM thread and updates only
the native window caption. App identity, native confirmation text, tray
tooltip, executable metadata and manifest stay unchanged. It adds no timer,
worker, dependency or persistent state. Call only when the desired title
changes; a dynamic caption is app-provided text, not a trusted app identity.

### Clipboard Text Write

ADR 0029 permits only writing text. Add `clipboard.write` to
`security.permissions` and call `window.velox.invoke("clipboard.writeText", {text})`
from a copy action. Text is capped at 32 KiB of UTF-8, without NUL; empty text
is allowed. The existing serialized-request budget remains 64 KiB, including
JSON escaping. Success is `null`. Busy clipboard access returns
`CLIPBOARD_BUSY` without retry; other native failures return
`NATIVE_OPERATION_FAILED` without text or OS details. Failure after emptying
the clipboard can leave it empty. This write grant enables no reading or background monitoring.
The opt-in permission allows trusted scripts to overwrite the clipboard without
a host prompt; user activation is not attested. Normal browser copy/paste stays
unchanged. Calls run on the existing UI thread and are rejected during shutdown.

### Clipboard Text Read

[ADR 0030](../adr/0030-confirmed-clipboard-text-read.md) adds the independent
`clipboard.read` permission. Invoke `window.velox.invoke("clipboard.readText")`
only from an explicit paste action. Each request requires a native confirmation
with default No; refusal returns `{cancelled: true}` and reads nothing. Approval
returns `{cancelled: false, text}` for Unicode text up to 32 KiB of UTF-8,
including empty text. The response's text budget is independent of the 64 KiB
serialized request limit; JSON escaping can expand returned text.

The request reserves its ID until deferred completion. One confirmation may be
pending per host; another read returns `TOO_MANY_REQUESTS`. Contention returns
`CLIPBOARD_BUSY`, unsupported/invalid Unicode `UNSUPPORTED_TEXT`, oversized
text `PAYLOAD_TOO_LARGE`, and native or stale-document failures
`NATIVE_OPERATION_FAILED`, all without contents or OS details. Navigation or
shutdown prevents stale text disclosure. No background reads, format enumeration,
history, writes, permanent approval or automatic retries are provided.
IPC does not attest a click, and returned text is available to opted-in app
scripts after approval. Clipboard contents can change while the dialog is open;
Windows retrieval can block the UI thread. Browser-native paste is unchanged.

### External HTTPS Links

ADR 0023 permits this one opt-in capability. Add `external.open` to the
manifest's `security.permissions` and invoke it from an application action:

```js
const result = await window.velox.invoke("external.open", {
  url: new URL("https://github.com/0disoft/velox").href,
});
```

`queued: true` acknowledges a scheduled native confirmation, not an opened
browser or loaded page. The UI queue shows a host-owned Yes/No prompt after
the WebView event returns, with No selected by default. Only Yes dispatches the
validated URI to the registered Windows HTTPS handler. Cancellation opens
nothing; a later native failure shows a generic host-owned error dialog. No
completion event is sent to the application. Shutdown drops pending work.

Only one confirmation may be queued or open per host; further requests fail
with `TOO_MANY_REQUESTS`. URLs are capped at 2,048 ASCII bytes. Use punycode
for international host names and percent-encoded UTF-8 for paths and queries.
Credentials, invalid hosts or ports, backslashes, whitespace, control characters
(including percent-encoded controls), and all non-HTTPS schemes are rejected.
No URL or query is logged. The host makes no network request, follows no
redirect, and cannot guarantee that the handler opens a browser or that a remote
site is safe. The OS handler registration remains a user-controlled boundary.

The existing IPC does not attest browser user activation. The native prompt,
not a JavaScript gesture claim, supplies per-request human approval. Ordinary
navigation and popup policies remain denied; this is an explicit method only.

### Selected Local Folders

ADR 0028 adds separate `folder.read`. `folder.select` opens a native folder
dialog and returns `{cancelled, name, target}`; `target` is positive only after
successful validation. It accepts no path. Only one local regular directory
is connected; network/device/stream paths, linked components, offline and
encrypted folders are rejected. Names are basenames, never full paths.
Selecting another folder replaces the target; cancellation/failure preserves it.
`folder.release({target})`, navigation, reload and shutdown revoke the connection.
It is not persisted; `folder.read` alone grants no file-content access or write capability.
Selection is deferred to native UI dispatch, with no watcher or background work.
`UNSUPPORTED_FOLDER` indicates a rejected location; `FOLDER_TARGET_INVALID`
indicates a stale/released connection.

`folder.list` returns `{entries: [{name, kind}], truncated, skipped}`. It accepts
only the active positive target token, revalidates directory identity and reads
from that handle without opening child paths. At most 129 entries are materialized
(one look-ahead), and the first 128 are considered. Reparse/offline/encrypted
entries are excluded; `skipped` counts exclusions in the examined portion.
The JSON result is bounded to 32 KiB after escaping; `truncated` indicates that
the entry or byte limit cut the result short. Filesystem enumeration order is
unsorted and not a stable snapshot. There is no pagination, total count, recursive
traversal, file-content access or write grant. Deletion/replacement invalidates
the target. No handle remains open between operations. See `examples/folder-browser`.

`folder.openText` additionally requires opt-in `folder.readText`; existing
listing-only apps remain denied. Only an immediate basename (the text-save
grammar, at most 240 UTF-8 bytes) and the active target are accepted. The host
revalidates folder identity and opens the child relative to that handle, never
by joining an ambient path. Directories, reparse/offline/encrypted children and
multiply hard-linked files are rejected. Traversal, ADS, absolute paths, trailing
dots/spaces and reserved device names are invalid parameters.

The result is `{cancelled: false, name, text, bytes}`, with the existing 2 MiB
UTF-8/BOM/NUL rules and no path or write token. `folder.readText` permits reads
of current immediate files, including names outside a truncated listing; it
does not freeze file identity or contents at selection/listing time. Missing or
inaccessible children return redacted `NATIVE_OPERATION_FAILED`; unsupported
types return `UNSUPPORTED_FILE`, oversized text `PAYLOAD_TOO_LARGE`, and a stale
directory `FOLDER_TARGET_INVALID`. Queued reads are revoked on release,
navigation or shutdown and do not retry. No child-folder navigation is added.

### Selected Local Text Files

ADR 0025 adds a read-only method. Declare `file.open` and invoke:

```js
const file = await window.velox.invoke("file.openText");
if (!file.cancelled) {
  editor.value = file.text;
}
```

The result is `{cancelled, name, text, bytes}`. Cancel returns `cancelled: true`
with empty name/text and zero bytes. Successful selection returns only the base
filename, not its path or a reusable handle. No parameters, including a path,
are accepted. One pending dialog is allowed; further selection requests fail
with `TOO_MANY_REQUESTS`. Each new read requires another native selection.

Only local disk UTF-8 files up to 2 MiB are accepted. A BOM is stripped from text;
`bytes` includes it. Network/device/ADS paths, directories, final-component
reparse points, offline placeholders, invalid UTF-8 and NUL bytes fail closed.
Mapped network drives and reparse points in parent components are also refused
before file opening. Oversize returns `PAYLOAD_TOO_LARGE`; unsupported files return `UNSUPPORTED_FILE`.
Other read failures are generic and never reveal the full path. Text is returned
as data; applications must not render it as untrusted HTML.

This one method is deferred to the UI queue after the WebView callback returns.
Its request ID remains in flight until completion. Navigation before reading
invalidates the selection; per-document transport correlation prevents response
delivery to a replacement document. Same-document navigation can reject the
pending operation without replacing its contents. Shutdown discards responses.
The 64 KiB bound applies to requests, not this response: text is capped at 2 MiB,
with JSON escaping expanding the encoded response to about 12 MiB at worst.

### Selected Local Text Saving

ADR 0026 adds an independent `file.save` permission. Use the frozen helper:

```js
const saved = await window.velox.saveText(editor.value, "notes.md");
if (!saved.cancelled) status.textContent = `${saved.name} saved.`;
```

Every call shows a native Save as dialog with overwrite confirmation. Cancel
returns `{cancelled: true, name: "", bytes: 0}` and writes nothing. The suggested
name must be a nonempty base filename (at most 240 UTF-8 bytes); paths, reserved
names, controls and trailing dots/spaces are invalid. Successful results contain
only the selected base name and UTF-8 byte count. No BOM or newline conversion
is added. No persistent grant or save-to-last-path is provided. The save dialog
offers Text documents, Markdown and All files; selection controls the default
filename extension, not the body encoding or document format.

The helper stages at most 2 MiB through the four wire methods above, using
surrogate-safe chunks of at most 4,096 UTF-16 code units. Both the existing
64 KiB request bound and WebView message bound remain unchanged. `bytes` is the
complete expected UTF-8 byte count; `offset` is the current received byte count.
Tokens are positive uint32 values, host-lifetime unique, and identify text only.
One upload or save may be pending. Invalid chunks do not advance it; incomplete
commits fail without consuming it. Cancel an abandoned upload with its token.
Accepted navigation and shutdown discard staged text. All four methods require
`file.save` before any staging or dialog interaction.

Commit consumes a complete upload and defers selection outside the WebView
callback; its ID remains reserved through completion. Navigation before writing
invalidates selection. New files cannot overwrite a target appearing during
commit. Existing files are written via flushed sibling temporary files and
ReplaceFileW with a backup, preserving the selected file's DACL. Linked,
remote, offline, readonly, encrypted and multiply-linked targets are refused.
Errors never contain a full path. `SAVE_RECOVERY_REQUIRED` means replacement or
backup cleanup failed: keep editor text and inspect recovery files in the chosen
folder. The target may already contain the new text. Crash or power-loss
atomicity and same-user adversarial path replacement are not guaranteed.

### Document-Scoped Save

ADR 0027 provides an explicit alternative to per-call selection:

```js
let saved = await window.velox.saveTextAs(editor.value, "notes.md");
if (!saved.cancelled) {
  saved = await window.velox.saveTextTo(editor.value, saved.target);
  // Before New or a different editor document in the same page:
  await window.velox.invoke("file.releaseSaveTarget", { target: saved.target });
}
```

The upload methods and limits are unchanged. `file.commitSaveAs` and
`file.commitSaveTo` are deferred, with IDs reserved through completion.
`target` is a positive uint32 in its own namespace, never an upload token or
caller path. Only one target is retained; successful connected Save as replaces
it and cancel keeps it. Navigation/reload/shutdown clear it. Do not persist it
or treat restored drafts as authorized to save. `saveText` still shows a picker
every time and does not return a target; `file.openText` remains read-only.

Before Save, compare the file's ID, size, last-write time and SHA-256 with its
last successful baseline. `FILE_CHANGED` rejects deleted/replaced/modified
targets without overwriting them, including same-size edits with restored
timestamps. `SAVE_TARGET_INVALID` rejects revoked or stale tokens. Keep editor
contents on errors; there is no force-overwrite flag. Native Save as supplies
explicit selection and overwrite confirmation when a conflict must be resolved.
Successful writes refresh the baseline only after verifying disk contents.
Verification/replacement failures can occur after writing, so retain buffers.
No file handle, watcher, timer or background worker is retained. Same-user
path races and power-loss atomicity retain ADR 0026's limitations.

## Stable Error Codes

- `INVALID_REQUEST`
- `INVALID_PARAMS`
- `METHOD_NOT_FOUND`
- `PERMISSION_DENIED`
- `PAYLOAD_TOO_LARGE`
- `TOO_MANY_REQUESTS`
- `DUPLICATE_REQUEST_ID`
- `UNSUPPORTED_VERSION`
- `SHUTTING_DOWN`
- `NATIVE_OPERATION_FAILED`
- `UNSUPPORTED_FILE`
- `SAVE_RECOVERY_REQUIRED`
- `FILE_CHANGED`
- `SAVE_TARGET_INVALID`
- `UNSUPPORTED_FOLDER`
- `FOLDER_TARGET_INVALID`
- `INVALID_RESPONSE` (JavaScript bridge validation)

Native failures return a stable message and do not expose paths, stack traces,
HRESULT values, configuration contents, or WebView message payloads.
