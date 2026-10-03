# Clipboard

Dependency-free Clipboard 0.1.0 example for Velox beta.20. It opts into only
`clipboard.write` and `clipboard.read`. Copy sends the left textarea to the
native writer; Paste asks for per-request native approval and displays returned
Unicode text in the right, read-only textarea. No automatic reads or writes,
network requests, browser clipboard fallback, storage, draft recovery, logs,
history, timer or worker are added. Closing the app discards displayed text.
It is independent of File Notes and uses a separate app identity/profile.

Cancellation and errors retain the prior pasted text. Pending work disables
both native action buttons. Empty Unicode text is accepted. Read/write text
is capped at 32 KiB of UTF-8; writes also use the 64 KiB serialized request
budget. Clipboard content is read after approval and may change while the
prompt is open. Only use disposable text for manual tests.

## Local Checks

`bun test examples/clipboard/app.test.ts` checks click-only native access,
literal text rendering, empty text, cancellation/error preservation, pending
request suppression and missing-bridge handling. Browser tests with mocked
native calls do not prove actual Windows approval or clipboard interoperability.
Runtime tests use fake clipboard calls or owned temporary memory, never the
maintainer's current clipboard.

The prepared portable output is under
`dist/manual/clipboard-beta20/example/dev.velox.clipboard/`.
Run `dev.velox.clipboard.exe`, type disposable Korean/English text on the left
and click Copy (this replaces the system clipboard). Click Paste, choose No
and verify the previous result stays unchanged; click Paste again, choose Yes
and verify the copied text appears unchanged on the right. Approval is required
again for each subsequent Paste. Confirm the copied text also pastes into
Notepad. Close only the test app; no installed File Notes app is replaced.

Bundled Lucide icons and their licenses are in `web/icons/`.
