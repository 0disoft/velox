# Development

- Status: Pre-implementation
- Owner: Project maintainer

## Current State

This repository does not contain Velox implementation source, a Go module,
build scripts, or an executable validation runner yet.

Do not infer development commands from this design scaffold. M0 must establish
the actual build and test front door before commands are documented.

## Planned M0 Environment

- Windows x64 development machine or CI runner.
- Go toolchain for the CLI and pure-Go host candidate.
- Installed Evergreen WebView2 Runtime.
- WebView2 SDK needed to define the native interface boundary.
- C++23 toolchain only for the minimal reference host.

Exact versions remain UNDECIDED until M0 records a reproducible toolchain.

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

Until a runner is configured, these checks are reported as skipped rather than
invented.

## M0 Completion

M0 development setup is complete only when:

- The Go and C++23 reference hosts build reproducibly.
- The hello fixture launches and emits the ready marker.
- Fresh and warm startup measurements can be repeated.
- Missing WebView2 and invalid configuration fail locally and cleanly.
- The selected command front door is documented here and in VALIDATION.md.
