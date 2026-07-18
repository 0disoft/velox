# Domain Model

- Status: Draft
- Owner: Project maintainer

## Build Domain

### Project

The resolved project root, authoring manifest, and static asset root.

Invariant: every project-owned path is represented relative to one canonical
root after validation.

### Manifest

The versioned authoring contract for application identity, assets, target,
window settings, security settings, and declared permissions.

Invariant: unknown required versions fail closed; defaults are applied only by
the CLI and are visible in inspection output.

### AppIdentity

Application identifier, display name, and version.

Invariant: identity is data in external configuration during M0 and does not
patch the generic host executable.

### AssetTree

The validated set of application-owned static files.

Invariant: entries cannot escape the asset root through absolute paths,
parent traversal, links, reparse points, reserved names, alternate streams,
case collisions, or archive path tricks.

### HostTemplate

An immutable prebuilt host selected from the pinned Actutum release.

Invariant: its digest and compatibility version are verified before copying;
consumer builds do not modify its bytes.

### BuildPlan

An immutable ordered description of validation, copy, report, archive, and
promotion operations.

Invariant: wall-clock time, machine-specific absolute paths, random identifiers,
and directory enumeration order do not influence artifact bytes.

### BuildOutput

The portable directory, deterministic ZIP, and machine-readable build report.

Invariant: output is promoted only after every planned operation succeeds.

## Runtime Domain

### RuntimeConfig

Validated application identity, entry point, target, window settings,
application origin, and permission set consumed by the host.

Invariant: the host reads runtime configuration but never interprets the
authoring manifest.

### ApplicationOrigin

The virtual HTTPS origin mapped to the local asset root.

Invariant: native messages are accepted only from the expected top-level
origin.

### CapabilitySet

A closed set of native methods granted to an application.

Invariant: missing or unknown capabilities deny access; no dynamic plugin or
reflection dispatch exists.

### IPCRequest and IPCResponse

Versioned JSON messages that correlate one JavaScript request with one native
result or stable error.

Invariant: message size, nesting, request identifiers, and in-flight counts are
bounded.

### HostSession

One process, UI thread, COM apartment, native window, WebView2 environment, and
controller lifecycle.

Invariant: COM callbacks and references do not outlive their owning session and
shutdown is idempotent.

## Benchmark Domain

### BenchmarkFixture

A pinned static application and framework adapter with a reproducible digest.

### BenchmarkRun

One isolated execution with framework version, runner metadata, profile mode,
phase durations, byte counts, result, and failure reason.

### Measurement

A duration, byte count, file count, or status with a named phase and unit.

Invariant: failed and timed-out runs remain in raw results.

## Ownership

- The CLI owns build-domain validation and transitions.
- The host owns runtime-domain lifecycle and dispatch.
- The benchmark repository owns cross-framework fixtures and measurements.
- Application business state and remote APIs remain outside Actutum.
