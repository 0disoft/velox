# ADR 0031: Dynamic Window Title

- Status: Accepted
- Date: 2026-10-04
- Owner: Runtime host

## Decision

Add only `window.setTitle({title})` under independent, default-disabled
`window.title`. The existing `window.basic` permission does not grant it,
and the new permission does not grant basic window controls.

Require exactly one string field, with a 512-byte UTF-8 bound and no Unicode
control characters. Empty resets to the manifest app name. The native caption
is app-controlled presentation, never proof of identity. Preserve manifest
identity, native approval text, tray tooltip and executable branding.

Use the existing synchronous UI/COM binding and check SetWindowTextW success.
Return null on success, INVALID_PARAMS for malformed fields/control characters,
PAYLOAD_TOO_LARGE for oversized input and redacted NATIVE_OPERATION_FAILED for
native failures. Existing origin, request limits and shutdown checks apply.
No watcher, goroutine, dependency or stored state is added.

## Consumer and Cost

File Notes will show basename, dirty marker and app name. Deduplicate identical
titles so typing sends one update on the clean-to-dirty transition, not one
request per keystroke. Caption failures must not block file editing or saving.
Measure host size against the same toolchain rather than infer a speed gain.

## Verification and Rollback

Check permission independence and propagation through manifest/build output,
strict fields, UTF-8/control bounds, identity preservation, shutdown rejection,
native Unicode caption readback and destroyed-window failure. Real packaged
File Notes title transitions remain a separate interaction check.
Remove window.title to disable it. No DB migration or runner change is needed.
