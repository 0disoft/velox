# Testing Standard

- Status: Active
- Owner: Project maintainer

## Principle

Tests prove contracts at their owning boundary. Snapshot volume, a green retry,
or a benchmark average does not replace behavior evidence.

The Go test suite, Windows startup fixture, release smoke, schema validation,
and hosted consumer workflow provide the executable test layers. This document
defines how those layers prove their owning contracts.

## CLI Unit Tests

- Manifest parsing, defaults, and semantic validation.
- Stable diagnostics and exit-code mapping.
- Path normalization and containment.
- Immutable build-plan ordering.
- Atomic staging, cleanup, and promotion.
- Deterministic archive metadata and ordering.
- Host compatibility and checksum verification.

## Host Unit Tests

- Runtime configuration parsing.
- Unsupported version and malformed input handling.
- Permission and method dispatch.
- Origin and top-level-frame validation.
- IPC size, nesting, duplicate, and in-flight limits.
- Idempotent shutdown and callback lifetime ownership.

Logic that does not require WebView2 should remain testable without launching a
window.

## JavaScript Bridge Tests

- Concurrent invocation and request identifier allocation.
- Success, stable error, timeout, and shutdown completion.
- Duplicate and unknown response identifiers.
- Malformed native messages.
- Pending-request cleanup.
- Frozen public namespace.

## Filesystem Adversarial Tests

- Parent traversal and absolute paths.
- Windows reserved names and invalid trailing characters.
- Alternate data streams.
- Case collisions.
- Links, junctions, and reparse points.
- Long paths and non-ASCII names.
- Locked, read-only, and pre-existing output.
- Unsafe ZIP entry names.

## Windows End-to-End Tests

- Build and inspect the dependency-free hello fixture.
- Launch and receive a ready marker.
- Launch a packaged application directly from a non-application working
  directory and resolve runtime configuration beside the executable.
- Exercise basic window lifecycle methods.
- Reject remote navigation and frame IPC.
- Fail cleanly when WebView2 is unavailable or unsupported.
- Repeat launch and shutdown without stale process or profile locks.

## Contract Tests

- Manifest schema and examples.
- Runtime configuration compatibility.
- IPC request, response, and stable errors.
- Build-result JSON.
- CLI JSON envelope and process exit code.
- Release manifest and artifact checksums.
- Signing-record schema, strict decoder, dry-run non-publishability, exact
  unsigned and signed artifact sets, signing-input ZIP contents, final manifest
  and ZIP lineage, checksums, and SBOM archive identity.
- Signing-input preparation determinism, normalized ZIP metadata, exact
  root-level executable set, missing or linked input rejection, and existing
  output preservation.
- Provider-output directory exactness, unexpected-file rejection, shared
  directory ownership, and regular-file enforcement.

## Reproducibility Tests

Equivalent normalized inputs and the same pinned Velox release must produce
byte-identical unsigned ZIP files on independent clean workspaces.

The test compares artifact bytes, not only extracted file contents.

## Performance Tests

Performance evidence follows docs/engineering/03-performance-budget.md.

- Pull requests run a bounded Velox-only smoke sample.
- Scheduled and release-candidate jobs run full cross-framework measurements.
- Raw failures and timeouts remain visible.
- Tests do not pass by silently retrying a failed first run.

## Security Tests

Every MUST rule in docs/engineering/04-security-baseline.md needs a positive or
negative executable test before alpha.

Fuzzing targets runtime configuration and IPC parsing once those parsers exist.

## Validation by Change

| Change | Minimum evidence |
| --- | --- |
| Product or architecture docs | docs and contract review |
| CLI command or option | unit, JSON contract, help, docs |
| Manifest field | schema, parser, defaults, invalid fixture, docs |
| Path or archive handling | unit and adversarial filesystem test |
| Host lifecycle | host unit and Windows end-to-end smoke |
| IPC method | dispatcher, permission denial, bridge, end-to-end |
| Performance-sensitive path | correctness plus before-and-after measurement |
| Release packaging | reproducibility, checksums, smoke, inspect |
| Signing input or record | deterministic archive, schema, semantic lineage, tamper failures, dry-run non-publishability |

## Skipped Evidence

A skipped validation records:

- Name of the skipped validation.
- Why it cannot run.
- What risk remains.
- What milestone or implementation supplies it.

An unavailable runner is a skip, never a fake pass.
