# ADR 0004: Keep a C++23 WebView2 host as the M0 reference implementation

- Status: Retired by ADR 0005
- Date: 2026-07-10
- Owner: Project maintainer

## Context

The pure-Go M0 host proves that one Go toolchain can produce a CGo-free Windows
executable, but its selected binding does not expose the virtual-origin and
browser-policy controls required by the product security contract. Startup
cost also needs a lower-level comparison before Actutum commits to a binding
fork, a direct pure-Go COM implementation, or a C++ host.

## Decision

Maintain a minimal Windows x64 C++23 reference executable under
`host/reference-cpp`. It uses Win32, COM, direct WebView2 interfaces, a virtual
HTTPS asset origin, direct web messages, and deny-by-default browser settings.
It is a benchmark and lifecycle reference, not the selected production host.

Use Pixi only to lock the maintainer-side Clang, CMake, lld, and Ninja versions.
The current reference build also requires installed Visual Studio C++ headers
and Windows SDK 10.0.26100.0. Consumer application builds use neither Pixi nor
a native compiler.

## Retirement

ADR 0005 selected Go for both production executables after the bounded M0
comparison. The reference source, Pixi environment, and executable comparison
tests were removed after the Go release-bundle and hosted consumer gates
passed. The measurements below remain historical evidence and are not
reproducible from the current repository checkout.

## Consequences

- The same static fixture can report the same DOM-plus-two-frame ready marker
  through the Go and C++23 hosts.
- The reference host exercises virtual-host mapping and browser policy hooks
  that the current Go wrapper does not expose.
- The executable is tiny, but it distributes `WebView2Loader.dll` beside the
  executable and therefore must not be compared by executable bytes alone.
- Maintainer cold-build and cache costs increase; consumer build and cache
  contracts do not change.
- The C++ source has a manual COM lifetime surface and a no-default-CRT build,
  so warning-clean compilation and repeated lifecycle smoke tests are required.

## Evidence

The first local ten-run comparison found C++23 fresh-profile p50/p95 of
925.82/1001.82 ms and Go fresh-profile p50/p95 of 965.79/1208.99 ms. The same
run found a repeatable approximately seven-second C++23 delay when immediately
relaunching against one warm profile. That warm result is a lifecycle defect or
an invalid benchmark condition, not evidence of a production-ready winner.

The local run is directional evidence only. It did not capture pinned-runner
metadata and must not be published as a product benchmark.

## Exit Criteria

Before selecting the production host:

1. Explain and remove or explicitly budget the C++ same-profile relaunch delay.
2. Repeat fresh, settled-warm, and immediate-relaunch profiles on a pinned CI
   runner with raw results and runtime metadata.
3. Compare total distributed host files, maintainer cold-build/cache cost,
   security hooks, and shutdown behavior rather than executable size alone.
4. Choose the Go fork, direct pure-Go COM, or C++23 path in a superseding ADR.
