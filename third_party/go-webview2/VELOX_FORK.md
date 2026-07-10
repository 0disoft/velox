# Velox fork notes

This directory is a narrow source fork of
`github.com/jchv/go-webview2@v0.0.0-20260205173254-56598839c808`.
The upstream MIT license is preserved in `LICENSE`.

Velox carries only the changes required by its Windows host boundary:

- deny all WebView2 permission requests by default;
- expose virtual-host-to-folder mapping through the public wrapper;
- close and release the WebView2 controller, webview, and environment;
- release queried `ICoreWebView2_3` interfaces; and
- remove window context after native window destruction.

Do not merge upstream changes mechanically. Review COM ownership, public API
changes, generated bindings, loader changes, and license notices before
updating the pinned source revision.
