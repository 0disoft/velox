# Capability Probe

This example reports which browser-owned capabilities are available inside the
current Velox WebView2 security boundary. It does not add a native API or claim
that feature detection proves an operation succeeds.

Automatic checks cover the secure context, local storage, IndexedDB, native app
identity, file-picker surfaces, clipboard write surface, drag and drop, and the
current notification permission state. File and clipboard operations remain
explicit user gestures so the probe does not open a picker, read a file, write
a file, or change the clipboard during startup.

The result is diagnostic and environment-specific. A supported API can still be
denied by WebView2 policy, Windows policy, user choice, or the current runtime.
