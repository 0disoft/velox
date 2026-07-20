# Development

- Status: Go runtime boundary active
- Owner: Project maintainer

## Current State

This repository contains the Windows-only Go CLI and production host, a strict
external runtime configuration parser, a dependency-free hello fixture, and a
named-pipe startup smoke harness.

The parent workspace owns bounded mustflow intents named `velox_format`,
`velox_lint`, `velox_test`, `velox_build`, and `velox_startup_smoke`. The
repository still has no standalone task runner. Do not infer additional
commands from `go.mod`.

## Planned M0 Environment

- Windows x64 development machine or CI runner.
- Go 1.26 toolchain for the pure-Go host candidate.
- Installed Evergreen WebView2 Runtime.
- A narrow local fork of `github.com/jchv/go-webview2`, pinned to commit
  `56598839c808` under `third_party/go-webview2`.

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

## Current Runtime Limitation

The selected Go host proves that a CGo-free WebView2 runtime can build, enforce
the first native security boundary, and reach the two-frame ready marker:

- The host now maps assets to an application-specific virtual HTTPS origin.
- The fork denies all WebView2 permission requests by default.
- The adapter accepts messages and top-level navigation only from the generated
  application origin, denies every child-frame navigation, and blocks popups
  and downloads at their native WebView2 events.

Treat the current executable as benchmark evidence only. It is not an alpha
runtime and must not be distributed as a secure application host.

## Production Host Decision

ADR 0005 selects Go for both the CLI and production host. This reduces the
normal product build, test, debugging, and release path to one maintainer
language. The repository now owns a narrow fork behind the pure-Go WebView2
adapter. Navigation, message-origin, popup, download, permission, and shutdown
contracts are enforced at that boundary and exercised by the startup security
fixture.

The retired C++23/Pixi comparison is retained only as historical evidence in
ADR 0004 and the performance budget.

The repository-owned adapter boundary lives in `internal/webview2`.
`cmd/velox-host` does not import the fork directly. The adapter reports virtual
HTTPS assets, trusted-origin messaging, navigation, frame, popup, download,
permission, and clean host shutdown controls as implemented.

The fork explicitly closes and releases controller, webview, queried extension,
and environment COM interfaces. The startup smoke reaches the ready marker,
exits normally, and no longer exceeds the ten-second profile cleanup window.
`CleanShutdown` means that the controller is synchronously closed, COM
interfaces are released in ownership order, and the host process exits within
one second. It does not claim that WebView2 has released the user-data folder.
That browser-process lifecycle is measured separately: repeated local smoke
shows same-profile immediate relaunch and final profile release taking about
seven seconds.

Benchmark-only host controls are ignored unless `VELOX_BENCH_MODE=1` is set.
This explicit mode prevents accidental benchmark configuration from affecting
ordinary application launches; it is not a security boundary against a parent
process that already controls the complete child environment and command line.
When both the mode and `VELOX_BENCH_PIPE` are set, the host emits one versioned startup
timeline after the ready marker. It records host entry, configuration loading,
WebView2 environment and controller creation, navigation dispatch, and the
DOM-plus-two-animation-frame boundary with a process-local monotonic clock.
The recorder is disabled during ordinary application execution, carries no
application data, and does not alter runtime policy decisions.

The benchmark path also emits a separate shutdown timeline after the native
message loop exits. It starts at the first runtime close request and records
dispatcher closure, queued destruction, event-handler removal, controller
close, WebView/controller/environment release, window destruction, and message
loop exit. These host-local phases do not claim that the browser process or
user-data folder has already been released.

## M0 Completion

M0 development setup is complete only when:

- The Go CLI and host build reproducibly.
- The hello fixture launches and emits the ready marker.
- Fresh and warm startup measurements can be repeated.
- Missing WebView2 and invalid configuration fail locally and cleanly.
- The selected command front door is documented here and in VALIDATION.md.

The retired Go/C++23 comparison is recorded in the performance budget. The Go
security controls now have executable navigation, frame, popup, permission, and download
evidence. A missing or invalid fixed WebView2 Runtime exits with code 5 and an
actionable local diagnostic. The browser-process relaunch delay remains open.
