# Project Invariants

- Status: Draft
- Owner: Project maintainer

## Product Invariants

1. A consumer application build never compiles application-specific native
   code.
2. A consumer build does not require Go, Rust, C++, Zig, Node.js, or a frontend
   package manager.
3. The generic host executable is copied without patching its bytes.
4. The default application model is static HTML, CSS, and JavaScript.
5. Windows x64 is the only target until the benchmark and security contracts
   are proven.
6. Missing features do not justify silently becoming a smaller Tauri or Wails.

## Build Invariants

1. A pinned Velox release contains every Velox-owned build dependency.
2. Builds perform no network access after that release is available locally.
3. Consumer GitHub Actions workflows upload zero bytes to actions/cache.
4. Builds do not create generated source or bindings.
5. Builds never modify application source assets.
6. Partial output is assembled in owned staging and is not promoted on failure.
7. Equivalent normalized inputs produce byte-identical unsigned archives.
8. Paths, links, reparse points, reserved names, and archive entries cannot
   escape their declared roots.

## Runtime Invariants

1. The host opens no listening TCP port and runs no local HTTP server.
2. Web content receives no native method that is absent from the closed
   permission table.
3. Filesystem, shell, process, plugin, and sidecar execution are absent from the
   MVP.
4. Native messages are accepted only from the expected top-level application
   origin and are strictly bounded.
5. Remote navigation, popups, downloads, and browser permissions are denied by
   default.
6. Telemetry, crash upload, and automatic update checks are off by default.
7. The host fails closed when runtime configuration or compatibility checks
   fail.

## Architecture Invariants

1. CLI and host are separate executables even when both are implemented in Go.
2. Packaging code does not enter the host process.
3. Windows and WebView2 code does not enter the CLI domain model.
4. The JavaScript bridge depends on protocol contracts, not native
   implementation details.
5. CGo or a C++ shim in the production Go host requires an explicit replacement
   ADR because it changes the toolchain and maintenance story.

## Evidence Invariants

1. Performance claims include tool acquisition and setup when labeled
   end-to-end.
2. Fresh and warm WebView2 profiles are reported separately.
3. p50 and p95 are published with raw successful and failed runs.
4. A blank native window is not counted as application ready.
5. Startup stops being a headline advantage if the measured difference is
   within benchmark noise.
6. No README claim may be stronger than the latest reproducible benchmark.

## Merge Blockers

- An invariant is weakened without an ADR and synchronized product changes.
- A command or contract changes without tests and documentation.
- A performance change lacks before-and-after evidence or an explicit reason
  that measurement is not yet possible.
- A skipped validation is hidden.

## Related Sources

- Product scope: docs/product/02-spec.md
- Architecture: docs/architecture/00-system-boundary.md
- Performance: docs/engineering/03-performance-budget.md
- Validation names: VALIDATION.md
