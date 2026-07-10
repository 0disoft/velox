# Third-Party Notices

Velox M0 source depends on the following software. Release automation must
regenerate and verify this notice before distributing binaries.

## github.com/jchv/go-webview2

- Version: `v0.0.0-20260205173254-56598839c808`
- License: MIT
- Copyright: John Chadwick and contributors; portions Serge Zaitsev
- Source: vendored narrow fork in `third_party/go-webview2`
- Purpose: Pure-Go WebView2 and Windows host binding for the Go host
- Local changes: default-denied permissions, virtual HTTPS folder mapping,
  explicit COM close/release, and native window-context cleanup

The upstream MIT license is preserved at
`third_party/go-webview2/LICENSE`. Fork maintenance notes are recorded in
`third_party/go-webview2/VELOX_FORK.md`.

## github.com/jchv/go-winloader

- Version: `v0.0.0-20250406163304-c1995be93bd1`
- License: MIT
- Purpose: Load the embedded Microsoft WebView2 loader used by go-webview2

## golang.org/x/sys

- Version: `v0.0.0-20210218145245-beda7e5e158e`
- License: BSD-3-Clause
- Purpose: Windows system-call support used transitively and by the startup test

Microsoft WebView2 Runtime and loader redistribution obligations remain
separate from the licenses above and must be reviewed before a public release.

## Microsoft.Web.WebView2

- Version: `1.0.4078.44`
- License: Microsoft software license included in the NuGet package
- Purpose: C++ headers and the x64 `WebView2Loader.dll` used by the reference host

The reference build also uses maintainer-only Pixi, LLVM/Clang/lld, CMake, and
Ninja packages pinned by `pixi.lock`. They are not copied into the application
runtime output. Their notices and the system Windows SDK and Visual Studio
toolset obligations must be collected before distributing a maintainer SDK or
source-build bundle.
