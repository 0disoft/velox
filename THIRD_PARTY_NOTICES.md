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
  message-source validation, navigation/frame/popup/download policy events,
  explicit COM close/release, event unregistration, and native window-context
  cleanup

The upstream MIT license is preserved at
`third_party/go-webview2/LICENSE`. Fork maintenance notes are recorded in
`third_party/go-webview2/VELOX_FORK.md`.

## github.com/jchv/go-winloader

- Version: `v0.0.0-20250406163304-c1995be93bd1`
- License: MIT
- Purpose: Load the embedded Microsoft WebView2 loader used by go-webview2

## golang.org/x/sys

- Version: `v0.48.0`
- License: BSD-3-Clause
- Purpose: Windows system-call support used transitively and by the startup test

Microsoft WebView2 Runtime and loader redistribution obligations remain
separate from the licenses above and must be reviewed before a public release.

## Development-Only Result Validation

- `github.com/santhosh-tekuri/jsonschema/v6`: `v6.0.3`, Apache-2.0.
- `golang.org/x/text`: `v0.42.0`, BSD-3-Clause (validator dependency).
- Purpose: Validate raw and summary JSON in `cmd/velox-consumer-summary`.
- Neither dependency is imported by the shipped CLI, host, or setup executable.
- Source and license: <https://github.com/santhosh-tekuri/jsonschema> and
  <https://go.googlesource.com/text>.

## Lucide Text-Editor Starter Icons

The CLI embeds file-plus, folder-open, save and save-all from
<https://github.com/lucide-icons/lucide>. Generated text-editor projects retain
the complete notice in `web/icons-license.txt`.

ISC License

Copyright (c) 2026 Lucide Icons and Contributors

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.
THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

These icons are derived from the Feather project:

The MIT License (MIT)

Copyright (c) 2013-present Cole Bemis

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
