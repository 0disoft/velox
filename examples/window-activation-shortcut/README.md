# Activation Shortcut

An isolated scratchpad that sets
`window.activationShortcut: "Ctrl+Alt+Shift+V"` with `window.tray: true` and no
native permissions. The shortcut adds no permission, IPC method or event, key
logging, custom action or configuration UI; the host registers one system-wide
Windows hot key that reveals the existing window. The page keeps its text in
memory only and reports its own input and focus state.

Build with a matching new CLI/host bundle containing ADR 0039:

```powershell
velox build --config examples/window-activation-shortcut/velox.json --out <absolute-output-directory>
```

Older bundles reject the unknown `activationShortcut` field. Extract the ZIP and
run `dev.velox.activation.exe` with a private profile. No additional native
permission is needed.

## Manual Check

1. Type into the note so the text differs from the initial value.
2. Use the tray icon's Hide window, then press `Ctrl+Alt+Shift+V`. The same
   window reappears and the typed text is preserved.
3. Minimize the window and press the shortcut; it restores without resetting
   the note.
4. Close the window; the host unregisters the hot key on destruction, so a later
   press does nothing. Exactly-once registration and unregistration are covered
   by the existing host tests. The reveal is best-effort foreground and Windows
   foreground rules can still leave focus elsewhere.

The value must be exactly `Ctrl+Alt+<key>` or `Ctrl+Alt+Shift+<key>`, where
`<key>` is one uppercase `A`-`Z` or `0`-`9`; an empty value or an omitted
field registers nothing. A combo already owned by another application is
reported with one host warning; the app continues without the shortcut, installs
no binding, and does not retry. Two separate applications, two profiles, or two
instances that use the same combo conflict even though their configuration is
independent. The optional `app.singleInstance` flag can avoid the same-app
multi-instance case, but it is not enabled here because this sample keeps a
preview/private application id.

The note is unsaved and in memory only: no application storage, file permission,
timer, worker, network request or frontend package, and File Notes documents and
profiles are not used. The native title-bar icon is provided by the host, so this
utilitarian example adds no raster icon.
