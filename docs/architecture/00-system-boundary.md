# System Boundary

- Status: Draft
- Owner: Project maintainer

## Owned Components

### Velox CLI

A standalone Go executable that owns manifest parsing, validation, path safety,
build planning, output staging, asset copying, deterministic archiving,
inspection, diagnostics, and machine-readable command output.

### Velox Host

A separate prebuilt native executable that owns the Windows window lifecycle,
WebView2 environment and controller lifecycle, virtual-host asset mapping,
navigation policy, direct web-message transport, and a fixed native method
dispatcher.

The first implementation candidate is a pure-Go Windows host. It must not use
CGo or a C++ shim in the consumer artifact. A minimal C++23 reference host may
exist only as benchmark and fallback evidence.

### JavaScript Bridge

A dependency-free script that owns request identifiers, Promise completion,
response validation, timeouts, and the small public invoke API. It does not
generate application bindings.

### Contracts

- Project manifest schema.
- Runtime configuration schema.
- Host compatibility version.
- IPC protocol version.
- Build-result schema.

Each contract is versioned independently in design but released atomically
during the MVP.

## Consumed Components

- Windows x64 desktop APIs, historically named the Win32 API.
- COM ABI required by the native WebView2 interfaces.
- An installed Evergreen WebView2 Runtime.
- Static application-owned HTML, CSS, JavaScript, and other assets.

Velox does not own the browser engine, application network endpoints, or
application business data.

## Build-Time Flow

1. Resolve the project root and manifest.
2. Parse and validate the manifest.
3. Validate the asset tree and entry point.
4. Resolve a pinned host template bundled with the Velox release.
5. Create an immutable build plan.
6. Assemble output in a sibling staging directory.
7. Copy the unchanged host, runtime configuration, and assets.
8. Write a machine-readable build report.
9. Create a deterministic ZIP.
10. Promote the completed output atomically.

No compiler, package manager, local server, code generator, or network lookup
runs in this flow.

## Runtime Flow

1. Start the generic host.
2. Read and validate external runtime configuration.
3. Initialize the Windows UI thread and COM single-threaded apartment.
4. Create the native window and WebView2 environment.
5. Apply security settings and virtual-host asset mapping.
6. Inject the frozen JavaScript bridge.
7. Navigate to the local application origin.
8. Accept bounded messages from the trusted top-level origin.
9. Dispatch only declared and granted native methods.
10. Shut down the WebView and native window cleanly.

## Trust Boundaries

- Manifest and static assets are untrusted build inputs.
- Web content is untrusted runtime input.
- The host release bundle is trusted only after checksum verification.
- WebView2 is an external runtime dependency.
- Application network traffic is outside the Velox trust boundary.
- The output directory is not modified until staging completes.

## Dependency Direction

The CLI may understand authoring contracts and host compatibility metadata. It
must not depend on host implementation details.

The host understands compact runtime configuration and IPC contracts. It must
not parse the authoring manifest or contain packaging logic.

The JavaScript bridge understands IPC messages. It must not expose Windows or
host implementation details.

## Repository Boundary

This repository owns the core CLI, host, bridge, schemas, specifications,
examples, and conformance tests.

Cross-framework benchmark adapters and raw historical benchmark results should
live in a separate public benchmark repository once M0 proves that the product
hypothesis is worth maintaining.

Signing credentials and unreleased vulnerability details must never live in
this repository.

## Recovery Boundary

Build recovery is local and deterministic: remove the current staging
directory, preserve source and the previous successful output, and return a
stable diagnostic and exit code.

Runtime recovery is fail-closed: an invalid configuration, unsupported contract
version, unavailable WebView2 Runtime, or untrusted navigation stops startup
with a local diagnostic. The host does not download missing components.
