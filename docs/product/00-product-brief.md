# Product Brief

- Status: Draft
- Owner: Project maintainer
- Working name: Velox

## Product

Velox is a compile-free Windows desktop application packager for static
HTML, CSS, and JavaScript.

It combines a project manifest, static web assets, and a prebuilt generic
WebView2 host into a portable application directory and deterministic archive.
The application author does not install or run Go, Rust, C++, Zig, Node.js, or
a frontend bundler during an application build.

## Problem

Desktop web wrappers can impose large cold-build costs through native
toolchains, package-manager dependencies, generated files, and CI caches.
That cost is especially visible on clean GitHub Actions runners and for small
applications whose own source is much simpler than the framework build surface.

Velox tests the hypothesis that a deliberately smaller product can provide a
useful desktop boundary while making the build path mostly validation, file
copying, and deterministic packaging.

## Primary Users

- Developers packaging offline or local-first static web applications.
- Maintainers of small internal tools, viewers, dashboards, prototypes, and
  kiosk-style applications.
- Teams for which clean CI time and cache consumption matter more than native
  plugin breadth.

## Headline Metrics

1. End-to-end cold build time.
2. GitHub Actions cache bytes uploaded by a consumer build.

## Guardrail Order

1. Process-to-ready startup and shutdown reliability.
2. Deterministic and inspectable output.
3. Minimal security-sensitive native surface.

Binary size, plugin breadth, deep OS integration, and frontend framework
convenience are secondary.

## Product Boundary

Velox owns:

- Project manifest validation.
- Prebuilt host selection and compatibility checks.
- Static asset validation and copying.
- Portable directory and deterministic archive creation.
- A small, versioned JavaScript-to-host message contract.
- Reproducible benchmark methodology and raw results.

Velox does not initially own:

- A native application backend.
- A frontend bundler or package manager.
- Plugins, sidecars, shell execution, or unrestricted filesystem access.
- Installers, automatic updates, crash upload, or code signing.
- Cross-platform parity.
- A local HTTP server or WebSocket transport.

## Product Hypothesis

The product proceeds only if a clean, public benchmark shows a material
end-to-end cold-build advantage over Wails, no consumer Actions cache upload,
and a simpler operating surface than the nearest compile-free alternatives.
ADR 0008 passes that structural gate only for portable static Windows apps and
keeps PWA as the default when browser-managed deployment is acceptable. ADR
0009 retains startup as a measured release guardrail, not a promised advantage.

## Data and Privacy

The CLI and host must not send telemetry, crash reports, or update checks by
default. Velox owns build inputs and outputs only while processing them.
Application network traffic and WebView2 profile data belong to the packaged
application and must be documented separately by that application.

## Related Sources

- Product specification: docs/product/02-spec.md
- Architecture decision: docs/adr/0001-initial-architecture-boundaries.md
- Performance contract: docs/engineering/03-performance-budget.md
- Roadmap: docs/product/01-roadmap.md
