# Always On Top

Isolated, unsaved scratchpad with `window.alwaysOnTop: true` and no native
permissions. JavaScript only reports the benchmark readiness marker after
two animation frames; topmost control belongs to the host system menu.
File Notes keeps its default off.

With a matching local/new CLI and host bundle:

```powershell
velox build --config examples/window-always-on-top/velox.json --out dist/examples/window-always-on-top --json
```

Launch the packaged EXE with a private profile. Switch to a normal non-topmost
window and check that the scratchpad stays above it. Alt+Space -> Always on top
unchecks the item; a normal window can now cover it. Toggle back on and check
the checkmark. Close/reopen: true is reapplied, not the last user toggle.
Minimize/restore should retain topmost without changing geometry. Close the
scratchpad after testing; its text is deliberately not saved or recovered.
