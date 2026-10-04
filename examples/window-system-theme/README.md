# System Theme

Isolated, unsaved scratchpad with `window.followSystemTheme: true` and no native
permissions. The host follows the Windows app light/dark setting for the native
title bar where the platform supports it. Application CSS stays app-owned and
uses `prefers-color-scheme` for its own light and dark colors, with a
`forced-colors` fallback. JavaScript only reports the benchmark readiness
marker after two animation frames.

With a matching local/new CLI and host bundle:

```powershell
velox build --config examples/window-system-theme/velox.json --out dist/examples/window-system-theme --json
```

Launch the packaged EXE with a private profile. Switch the Windows app theme in
Settings > Personalization > Colors and check that the native title bar follows
it on a Windows 11 build 22000 or newer; the app content should switch with
`prefers-color-scheme` at the same time. Older or unsupported builds keep the
default title bar. Enable a high contrast theme and check that the system scheme
is kept. Live user-driven Windows theme changes remain manual evidence and are
not yet recorded. The scratchpad text is deliberately not saved.
