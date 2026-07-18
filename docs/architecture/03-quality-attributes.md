# Quality Attributes

- Status: Draft
- Owner: Project maintainer

## Priority Order

1. Fast and small consumer build surface.
2. Deterministic, inspectable output.
3. Secure minimum native boundary.
4. Reliable startup and shutdown.
5. Maintainability for a small project.
6. Compatibility breadth.

Later attributes must not silently override earlier ones.

## Build Performance

- Consumer builds install no compiler or frontend package manager.
- Consumer Actions cache upload is zero bytes.
- Build phases are measurable separately.
- Full cross-framework benchmarks run outside the normal pull-request path.

Thresholds and regression policy live in
docs/engineering/03-performance-budget.md.

## Startup Performance

- Ready means usable content after DOMContentLoaded and two animation frames.
- Fresh and warm WebView2 profiles are distinct measurements.
- Startup claims include failures and environment metadata.
- ADR 0009 treats startup as a release guardrail and lifecycle diagnostic, not
  a product headline.
- ADR 0005 fixes the production language unless its explicit safety revisit
  trigger is met.

## Determinism

- Build planning is independent of filesystem enumeration order.
- Archive entry order, metadata, and timestamps are normalized.
- Machine-specific paths, random identifiers, and current time do not affect
  unsigned artifact bytes.
- Inspection reports contract versions and digests.

## Security

- Local web content remains untrusted.
- Native capabilities are absent unless explicitly declared and granted.
- No listening socket, local server, plugin scan, shell, process, or arbitrary
  filesystem API exists in the MVP.
- Path validation and atomic output ownership prevent source escape and unsafe
  cleanup.
- Directory asset tampering is a documented limitation, not a solved property.

## Reliability

- Invalid configuration and unsupported versions fail closed.
- Build failure preserves source and previous successful output.
- Startup failure does not download or repair dependencies implicitly.
- Shutdown releases WebView2 callbacks on the owning thread and is idempotent.

## Maintainability

- CLI and host remain separate executables and ownership boundaries.
- Public contracts have one named source of truth.
- Go COM code must make thread and lifetime ownership explicit.
- Dependencies require a removal path and measurable build or runtime cost.
- Deferred features stay out of core until an actor, contract, and verification
  path exist.

## Compatibility

- Windows x64 and Evergreen WebView2 are the only initial compatibility promise.
- The supported floor is Windows 10 version 1709 x64 or Windows Server 2016
  x64 with Evergreen WebView2 Runtime `92.0.902.49` or newer. This is the
  intersection of the Go runtime floor and the `ICoreWebView2_4` security API
  required to deny downloads.
- Manifest, IPC, host, and build-result versions reject unsupported required
  versions rather than guessing.
- Cross-platform support requires a separate ADR and benchmark.

## Operability

- CLI and host diagnostics are local by default.
- Stable diagnostic codes support CI and troubleshooting.
- No telemetry, crash upload, or update check runs by default.
- Release artifacts publish checksums and compatibility metadata before alpha.

## Evidence Rule

No quality claim is considered met because it appears in this document.
Implementation, tests, raw benchmark results, and release artifacts provide the
evidence.
