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

The Windows transport invokes each native binding synchronously on WebView2's
UI/COM event thread. Multiple unresolved JavaScript promises therefore do not
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

The method table is a closed switch. Reflection is confined to the private
WebView transport adapter and cannot select a product method dynamically.

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
- `INVALID_RESPONSE` (JavaScript bridge validation)

Native failures return a stable message and do not expose paths, stack traces,
HRESULT values, configuration contents, or WebView message payloads.
