# Velox fork notes

This directory is a narrow source fork of
`github.com/jchv/go-webview2@v0.0.0-20260205173254-56598839c808`.
The upstream MIT license is preserved in `LICENSE`.

Velox carries only the changes required by its Windows host boundary:

- deny all WebView2 permission requests by default;
- expose virtual-host-to-folder mapping through the public wrapper;
- validate WebMessage sources before dispatching bound callbacks;
- bound accepted WebMessages to 64 KiB and reject native conversion failures;
- deny untrusted top-level navigation and all frame navigation;
- deny popup and download events;
- allow an explicit fixed-runtime folder for missing-runtime conformance tests;
- expose the main WebView2 browser process ID for lifecycle measurement;
- expose a phase-only startup observer for benchmark instrumentation without
  changing WebView2 initialization decisions;
- expose a phase-only shutdown observer around handler removal, controller
  close, COM release, native window destruction, and message-loop exit;
- close and release the WebView2 controller, webview, and environment;
- unregister native event handlers during shutdown;
- discard queued binding responses after native window shutdown begins;
- fail initialization when mandatory WebMessage or permission policies cannot
  be registered, without terminating the embedding process from the library;
- release queried `ICoreWebView2_3` interfaces; and
- remove window context after native window destruction.

Do not merge upstream changes mechanically. Review COM ownership, public API
changes, generated bindings, loader changes, and license notices before
updating the pinned source revision.

The upstream x86 and ARM64 loader files remain checked in even though the first
supported target is Windows x64. Removing them saves only about 231 KiB from
the source checkout, does not reduce the x64 executable because build tags
already exclude them, and would make upstream review noisier. Revisit only if
repository or CI transfer measurements make that source-only cost material.
