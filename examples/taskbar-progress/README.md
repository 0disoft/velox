# Taskbar Progress

This isolated example requests only `window.progress`. Select a state and
commit the slider value to request taskbar progress. None or the clear icon
removes it. The in-window indicator remains available when Windows hides or
groups taskbar progress. High contrast uses system colors and state labels.

`normal`, `error` and `paused` require an integer `value` from 0 to 100;
`none` and `indeterminate` omit it. The native host accepts the last request
before taskbar readiness and restores it when the shell creates the button.
An accepted request is not proof that Windows is showing the indicator.
All requests run on the existing UI/COM thread. The example has no timer,
network request, file permission, draft storage or frontend package.

Use a matching new CLI/host bundle supporting ADR 0037; an older bundle may
reject the permission or return METHOD_NOT_FOUND. Build with:

```powershell
velox build --config examples/taskbar-progress/velox.json --out <absolute-output-directory>
```

Extract the ZIP and run `dev.velox.progress.exe`. Native display checks include
normal percentages, indeterminate, error, paused, clear and close; separately
check Explorer button recreation before claiming Explorer recovery. No Windows
setting needs changing. Existing File Notes documents/profiles are not used.

The clear icon is an existing bundled Lucide asset; see `web/icons/LICENSE.txt`.
