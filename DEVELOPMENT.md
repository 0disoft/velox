# Development

- Status: M0 spike active
- Owner: Project maintainer

## Current State

This repository contains Windows-only Go and C++23 host spikes, a strict
external runtime configuration parser, a dependency-free hello fixture, and a
named-pipe startup benchmark harness.

The parent workspace owns bounded mustflow intents named `velox_format`,
`velox_lint`, `velox_test`, `velox_build`, `velox_startup_smoke`,
`velox_cpp_build`, `velox_cpp_startup_smoke`, and
`velox_startup_benchmark`. The repository still has no standalone task runner.
Do not infer additional commands from `go.mod` or `pixi.toml`.

## Planned M0 Environment

- Windows x64 development machine or CI runner.
- Go 1.26 toolchain for the pure-Go host candidate.
- Installed Evergreen WebView2 Runtime.
- `github.com/jchv/go-webview2` pinned to commit `56598839c808`.
- Pixi 0.72.2 for the maintainer-only C++23 reference environment.
- Locked Clang 21, CMake 4, lld 21, and Ninja 1.13 through `pixi.lock`.
- Installed Visual Studio C++ headers and Windows SDK 10.0.26100.0.
- Microsoft.Web.WebView2 SDK 1.0.4078.44 for headers and the loader DLL.

Pixi is not a consumer dependency and does not make the reference build fully
self-contained. The current build still discovers system Visual Studio C++
headers and the Windows SDK. A clean CI image must provide those components or
the build must later pin an explicit SDK/toolset bundle.

## Planned Repository Boundaries

    cmd/
      velox/
      velox-host/
    internal/
      manifest/
      buildplan/
      packagefs/
      archive/
      diagnostics/
      windows/
      webview2/
      ipc/
    sdk/
      js/
    schemas/
    examples/
      hello/
    tests/
      conformance/
      e2e/

The tree is a design target. It is not current repository state and may change
through an ADR before source creation.

## Development Rules

- Keep CLI and host as separate executables.
- Keep Windows and WebView2 details out of CLI domain packages.
- Keep packaging code out of the host.
- Avoid CGo in the Go host candidate.
- Do not add application-specific native compilation.
- Do not introduce Node.js for the dependency-free example or bridge.
- Add stable diagnostics before relying on log text.
- Add tests with each public command, contract, or native method.

## Validation Contract

Future executable checks use the stable names in VALIDATION.md:

- format
- lint
- typecheck
- test
- contract
- smoke
- docs
- check

The parent mustflow contract currently implements format, lint, test, build,
and startup smoke. Other checks remain skipped rather than invented.

## Current M0 Limitation

The selected Go binding proves that a CGo-free WebView2 host can build and
reach the two-frame ready marker. It does not expose enough policy surface to
implement the production security contract without a maintained patch or a
lower-level host implementation:

- The spike navigates to `file://` instead of an application-specific virtual
  HTTPS origin.
- The binding does not expose top-level navigation, popup, download, frame, or
  permission policy hooks through its public `WebView` interface.
- The binding enables clipboard-read permission during construction.

Treat the current executable as benchmark evidence only. It is not an alpha
runtime and must not be distributed as a secure application host.

## Production Host Decision

ADR 0005 selects Go for both the CLI and production host. This reduces the
normal product build, test, debugging, and release path to one maintainer
language. The decision does not select the current M0 wrapper. Production work
must introduce a repository-owned pure-Go WebView2 adapter that can enforce the
virtual-origin, navigation, permission, message-origin, and shutdown contracts.

The C++23/Pixi path remains reference-only and is removed or moved after the Go
adapter has a stable pinned-CI lifecycle baseline.

## M0 Completion

M0 development setup is complete only when:

- The Go and C++23 reference hosts build reproducibly.
- The hello fixture launches and emits the ready marker.
- Fresh and warm startup measurements can be repeated.
- Missing WebView2 and invalid configuration fail locally and cleanly.
- The selected command front door is documented here and in VALIDATION.md.

The Go and C++23 hosts build and reach the same two-frame marker locally. The
first repeated comparison is recorded in the performance budget. The C++23
host still has a same-profile immediate-relaunch delay that must be explained
before either host is selected. Missing-runtime tests and production security
controls remain open.
