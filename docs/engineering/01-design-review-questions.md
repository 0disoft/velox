# Design Review Questions

- Status: Active
- Owner: Project maintainer

## Product

- Which named user and workflow need this change before the next milestone?
- Does it preserve the static, compile-free product boundary?
- Why is a PWA or existing compile-free wrapper insufficient?
- Is the surface supported now, deferred, explicitly unsupported, or internal?
- What evidence would cause the project to reject or remove it?

## Build

- Does a consumer install or execute a compiler, Node.js, or package manager?
- Does the change add network access, generated source, cache, or intermediate
  output?
- Can failure damage source or previous successful output?
- Are path ownership, staging, promotion, cleanup, and determinism explicit?
- How are release and host bytes pinned and verified?

## Runtime

- Does the host remain generic and byte unchanged?
- Which thread owns Windows, COM, and WebView2 lifecycle state?
- Can a callback or reference outlive its session?
- Does the change open a socket, start a server, scan plugins, or add background
  work?
- Does it change process-to-ready or shutdown behavior?

## Security and Privacy

- Which trust boundary receives new input or authority?
- Can a frame, remote origin, malformed message, or path bypass validation?
- Is the native method table still closed and permission checked?
- Does output expose source, secrets, absolute paths, or native stack traces?
- Does the change add telemetry, update checks, crash upload, or remote storage?

## CLI and Contracts

- Which document or schema is the primary source?
- Are commands, options, JSON, diagnostics, exit codes, examples, and tests
  synchronized?
- Is the change backward compatible, versioned, or explicitly rejected?
- Are defaults visible and deny-by-default?
- Can automation distinguish every expected failure without parsing prose?

## Performance

- Which priority metric is affected?
- Is setup included in an end-to-end claim?
- Are fresh and warm profiles separated?
- Are p50, p95, failures, and environment metadata preserved?
- Does the feature's value justify its cold-build, cache, and startup cost?

## Operations and Recovery

- Can the problem be diagnosed locally without a Actutum service?
- What is the rollback or fallback?
- Does the change create a new release, signing, support, or incident surface?
- Which configured validation proves readiness?
- Which skipped validation and residual risk remain?

## Decision

A design review ends with one of: accept, accept with constraints, experiment
first, defer, reject, or supersede through a new ADR.
