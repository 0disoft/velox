# Third-Party Notices

Actutum M0 source depends on the following software. Release automation must
regenerate and verify this notice before distributing binaries.

## github.com/jchv/go-webview2

- Version: `v0.0.0-20260205173254-56598839c808`
- License: MIT
- Copyright: John Chadwick and contributors; portions Serge Zaitsev
- Source: vendored narrow fork in `third_party/go-webview2`
- Purpose: Pure-Go WebView2 and Windows host binding for the Go host
- Local changes: default-denied permissions, virtual HTTPS folder mapping,
  message-source validation, navigation/frame/popup/download policy events,
  explicit COM close/release, event unregistration, and native window-context
  cleanup

The upstream MIT license is preserved at
`third_party/go-webview2/LICENSE`. Fork maintenance notes are recorded in
`third_party/go-webview2/ACTUTUM_FORK.md`.

## github.com/jchv/go-winloader

- Version: `v0.0.0-20250406163304-c1995be93bd1`
- License: MIT
- Purpose: Load the embedded Microsoft WebView2 loader used by go-webview2

## golang.org/x/sys

- Version: `v0.0.0-20220412211240-33da011f77ad`
- License: BSD-3-Clause
- Purpose: Windows system-call support used transitively and by the startup test

Microsoft WebView2 Runtime and loader redistribution obligations remain
separate from the licenses above and must be reviewed before a public release.
