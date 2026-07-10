# Code Review Checklist

- Status: Draft
- Owner: Project maintainer

This checklist becomes executable after source exists. Until then it is a
review contract, not evidence that implementation satisfies it.

## Scope and Ownership

- The change belongs to the named milestone and product boundary.
- CLI, host, bridge, schema, benchmark, and application responsibilities remain
  separate.
- Deferred or unsupported features did not enter through a helper or
  dependency.
- Generated and vendor files are not hand edited.

## Correctness

- Inputs are validated once at the owning boundary.
- Errors preserve causes internally and map to stable public diagnostics.
- Failure and cancellation clean only resources owned by the current operation.
- Ordering and output do not depend on unstable filesystem enumeration.
- Edge cases have behavior-focused tests.

## Windows and WebView2

- UI and COM work occurs on the owning OS thread.
- Callback, reference, controller, and shutdown lifetimes are explicit.
- HRESULT and missing-runtime failures are not swallowed.
- No CGo or C++ shim entered the Go host without an ADR.
- Repeated startup and shutdown leave no stale process or lock.

## Security

- Paths remain contained under declared roots.
- Origin, frame, protocol, payload, method, and permission checks occur before
  native dispatch.
- No new shell, process, filesystem, plugin, sidecar, or listening-socket
  capability appeared.
- Diagnostics and logs contain no secrets or source contents.
- Negative tests cover the changed trust boundary.

## CLI and Contracts

- Help, configuration, JSON, exit codes, diagnostics, examples, and docs agree.
- Machine output is deterministic and contains no decorative text.
- New fields or methods have compatibility and version decisions.
- Unsupported versions fail explicitly.
- Human wording is not used as an automation contract.

## Performance and Dependencies

- Consumer toolchain and cache invariants remain true.
- New dependencies pass the admission checklist and have a removal path.
- Hot-path allocations, startup work, file I/O, and archive work are bounded.
- Relevant before-and-after measurements exist or the evidence gap is explicit.
- Benchmark failures and outliers were not removed.

## Verification

- The narrowest relevant configured validations ran.
- First failures were preserved rather than hidden by retry.
- Skipped validations include reason and remaining risk.
- Docs and diagrams synchronized with the primary contract.
- The final report does not claim implementation or performance beyond evidence.
