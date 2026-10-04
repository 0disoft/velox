# Tray Notification

An isolated example of `notification.show` for an opted-in tray. The manifest
grants only `notification.show` and sets `window.tray: true`; the balloon
belongs to the host's registered notification-area icon.

The `velox.json` file is the build config, not a build output. Build with a
local CLI/host bundle containing ADR 0038:

```powershell
velox build --config examples/tray-notification/velox.json --out <absolute-output-directory>
```

Extract the ZIP and run `dev.velox.notification.exe`. The method needs
`window.tray: true` and a currently registered icon; otherwise it returns
`NATIVE_OPERATION_FAILED` and installs no icon, and the example shows a local
status instead.

`kind` is `info`, `warning`, or `error`. The message must contain a non-space
character, be at most 255 UTF-16 units, and avoid NUL, DEL and other C0/C1
controls except line feed and tab; the page validates the same limits and sends
only on an explicit Send. The balloon title is the fixed application name, not
the message. An accepted request (`null`) means the Windows shell took it, not
that a balloon was visible.

Manual check: click Send, choose the tray icon's Hide window, then click the
balloon; the existing window reappears. Closing the window removes the icon.
Windows may still suppress balloons (quiet time, disabled notifications or Do
Not Disturb), which is shell behavior. No Windows setting is changed. There is
no timer, worker, network request, storage or file permission, and no other
app-defined tray command. Existing File Notes documents and profiles are not
used.

The bell icon is an existing bundled Lucide asset; see `web/icons/LICENSE.txt`.
