# ADR-0001: Initial Architecture Boundaries

- Status: Superseded by ADR 0005
- Date: 2026-07-10
- Owner: Project maintainer

## Context

Velox exists to reduce end-to-end cold build time, consumer GitHub Actions
cache use, and application startup overhead for small desktop applications
built from static web assets.

Choosing a faster native language does not solve the primary problem if every
consumer application still compiles native code. The main architectural lever
is therefore a prebuilt generic host rather than the implementation language
alone.

The same host language as the CLI reduces repository and release complexity,
but WebView2 exposes native COM interfaces whose threading, callback, lifetime,
and error semantics must be handled correctly.

## Decision

### Platform

M0 and MVP target Windows x64 and the installed Evergreen WebView2 Runtime.
Additional operating systems and architectures are deferred.

### CLI

The CLI is a standalone Go executable.

### Host

The first production candidate is a separate pure-Go generic host executable.
It uses Windows desktop APIs and the COM ABI required by WebView2.

The candidate must:

- Avoid CGo and a C++ compatibility shim.
- Lock UI and COM work to the owning OS thread.
- Make callback and reference lifetimes explicit.
- Expose only the fixed MVP method table.
- Contain no packaging or CLI implementation.

ADR 0005 subsequently made Go the accepted production host language. The
constraints below remain historical M0 gates rather than an automatic language
fallback policy.

### Reference Implementation

M0 may include a minimal C++23 host used only to test startup and WebView2
lifecycle behavior. It is not shipped as the default unless the Go candidate
fails the gate.

The default host changes to C++23 if the Go host:

- Is more than 10 ms or 10% slower at p95 process-to-ready startup.
- Requires CGo or a C++ shim.
- Cannot safely and maintainably represent WebView2 COM lifecycle behavior.
- Produces an unacceptable crash, memory, or debugging surface.

Changing the host language does not change the compile-free consumer build.

### Packaging

The CLI copies an unchanged prebuilt host alongside:

- External runtime configuration.
- A static web asset directory.
- Build metadata.

The CLI does not append configuration to or patch resources inside the host
executable. This preserves the bytes and signature of the generic host.
Application-specific executable naming, icon resources, and signing are
deferred.

### Asset and IPC Model

- Assets remain directory based in the first release.
- WebView2 maps them to an application-specific virtual HTTPS origin.
- The host opens no listening socket.
- IPC uses direct WebView2 web messages and JSON request-response envelopes.
- Generated bindings, streaming, events, and binary transfer are deferred.

## Consequences

### Positive

- Consumer builds are validation, copying, and archiving rather than native
  compilation.
- CLI and candidate host share one maintainer language and toolchain.
- The unchanged host has a stable digest and can preserve vendor signing.
- The runtime has no localhost port, server process, or plugin scan.
- The Go experiment remains reversible before public API commitments.

### Negative

- A generic unpatched host cannot initially provide per-application executable
  branding and resource metadata.
- External assets and configuration are modifiable by an attacker who controls
  the installation directory.
- Pure-Go COM bindings may require more careful lifecycle code than C++.
- WebView2 initialization may dominate startup enough that host-language
  differences are insignificant.
- Windows-first architecture does not prove macOS or Linux feasibility.

## Rejected Alternatives

### Compile each application in Go

Rejected because it requires a consumer toolchain and cache, reproducing the
cost Velox is intended to remove.

### Patch configuration into the executable

Rejected for M0 because it changes host bytes and invalidates an existing
Authenticode signature.

### Local HTTP server and WebSocket IPC

Rejected because it adds startup work, port and token handling, firewall
surface, and another failure mode.

### C++23 host without comparison

Rejected as an automatic default. C++23 remains the boring fallback, but the
maintenance and startup tradeoff must be measured against the Go candidate.

## Validation

M0 must produce raw evidence for:

- Go and C++23 process-to-ready startup.
- Fresh and warm WebView2 profiles.
- Host failures when WebView2 is missing or configuration is invalid.
- Clean shutdown and repeated launch.
- The absence of CGo and listening sockets in the Go candidate.

## Revisit Triggers

- The Go experiment fails a host gate.
- Per-application branding becomes an MVP requirement.
- Directory asset tamper resistance becomes mandatory.
- A second operating system is approved.
- A native capability beyond basic window control is proposed.
