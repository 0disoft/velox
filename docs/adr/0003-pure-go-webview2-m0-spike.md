# ADR 0003: Use go-webview2 for the M0 feasibility spike only

- Status: Accepted for M0
- Date: 2026-07-10
- Owner: Project maintainer

## Context

Velox must determine whether one Go toolchain can produce a CGo-free generic
Windows host while keeping consumer application builds compile-free. The M0
host needs to load static content and emit a two-frame ready marker before a
larger CLI or packaging implementation is justified.

## Decision

Pin `github.com/jchv/go-webview2` at commit `56598839c808` for the M0 startup
spike. Keep the dependency behind the `velox-host` command boundary. Use an
external runtime JSON file, local static assets, and a benchmark-only named-pipe
ready marker.

This decision does not select the production host implementation.

## Consequences

- A Windows x64 host builds with Go and no CGo or C++ shim.
- The host can reach `DOMContentLoaded` plus two animation frames and report
  readiness without opening a listening socket.
- The same binding is used by Wails, so differentiation must come from the
  unchanged prebuilt host and compile-free consumer build path.
- M0 uses `file://` because the wrapper does not expose virtual-host mapping
  through its public host interface.
- Navigation, popup, download, frame-origin, and permission policy controls
  required by the security baseline are not available at the wrapper boundary.
- The wrapper enables clipboard-read permission during construction, contrary
  to the Velox deny-by-default security contract.

## Exit Criteria

Before alpha, choose one of these paths through a new ADR:

1. Maintain a narrowly reviewed patch or fork exposing the required WebView2
   policy controls without widening native capability.
2. Implement a lower-level pure-Go host directly against Windows and WebView2
   interfaces.
3. Use the measured C++23 reference host if the Go path cannot meet security,
   lifecycle, and startup budgets.

The current M0 binary must not be released as a secure application runtime.
