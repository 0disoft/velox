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
| `external.open` | `external.open` | `{"url":"https://example.com/"}` | `{"queued":true}` before native confirmation |
| `file.openText` | `file.open` | `{}` | selected UTF-8 text or a cancellation result after the native dialog |

The method table is a closed switch. Reflection is confined to the private
WebView transport adapter and cannot select a product method dynamically.

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
- `INVALID_RESPONSE` (JavaScript bridge validation)

Native failures return a stable message and do not expose paths, stack traces,
HRESULT values, configuration contents, or WebView message payloads.
