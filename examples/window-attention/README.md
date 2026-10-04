# Window Attention

An isolated example of window.requestAttention and window.cancelAttention.
It requests only window.attention, has no network or persistence, and does
not replace File Notes or use its recovery profile. Icons are vendored Lucide.
The host has no added timer; the example has at most one cancellable frontend
timeout, scheduled only after a request-button action.

## Build and Test

Build with a local CLI/host bundle containing ADR 0032 support:

```powershell
velox build --config examples/window-attention/velox.json --out C:/absolute/output --json
bun test examples/window-attention/app.test.ts
```

Relative build output resolves from the manifest directory. The local
candidate keeps beta.20; no public release is implied.

## Manual Check

1. Set delay to 3 seconds, request attention, then switch to another app.
2. Observe bounded flashing of the test app's taskbar button without focus theft.
3. Request again and cancel; verify flashing stops.
4. Cancel during a scheduled delay; verify no later request occurs.
5. With delay 0 and the test app foreground, requesting should do nothing.

The result says Requested, not that a visual cue was observed. Windows
shell/accessibility behavior controls the actual display. No continuous
flashing, caption flashing, show/restore, notification or automatic request
on startup/focus/change is added. Request counts are bounded per call.
