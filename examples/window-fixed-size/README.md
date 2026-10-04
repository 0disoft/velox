# Fixed Window

Unsaved scratchpad using `window.resizable: false`, `rememberState: true`, and
no native permissions. Text is not saved; only host window position is retained
in a private test profile. JavaScript reports the existing readiness marker.

```powershell
velox build --config examples/window-fixed-size/velox.json --out dist/examples/window-fixed-size --json
```

Use a matching new/local CLI and host. Launch the EXE with a private profile.
Check that border dragging, title-bar double-click and Alt+Space Size/Maximize
cannot resize/maximize it. Minimize, restore, move and close should work.
Move it, close and relaunch with the same profile: the position should return
with the configured DPI-scaled outer 480 x 360 size. If moving across monitors,
check rendering and OS DPI adjustment rather than expecting constant physical
pixels. The restored window is capped to the target monitor work area.
