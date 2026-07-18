# ADR 0008: Pass the narrow structural-simplicity gate

- Status: Accepted
- Date: 2026-07-18
- Owner: Project maintainer

## Context

The M3 gate asks whether Actutum is structurally simpler than the closest
compile-free wrapper and meaningfully distinct from a PWA. Fewer features alone
do not satisfy that gate.

Neutralinojs is the closest wrapper comparison. Its documented architecture
serves local resources over HTTP and uses WebSocket messages for native APIs.
Its framework surface includes filesystem, OS, extensions, updater, storage,
server, and window namespaces. The pinned benchmark adapter responsibly
allowlists only `window.setTitle`, but the distributed framework still owns the
server, protocol, extension, and broader API implementation. Actutum instead maps
an external directory to a virtual HTTPS origin, uses direct WebView2 messages,
and exposes six closed application/window methods. Both approaches avoid
bundling a browser engine.

A PWA is the stronger counterargument. Edge can install a web application and
provide offline behavior, local storage, and OS integration. Reliable offline
operation normally adds a service-worker and cache lifecycle, while normal web
distribution uses an HTTPS origin and browser-managed installation.

Evidence sources:

- the repository product, runtime, IPC, and build invariants;
- the pinned Neutralinojs `v6.8.0` benchmark adapter;
- [Neutralinojs architecture](https://neutralino.js.org/docs/contributing/architecture/);
- [Neutralinojs native API overview](https://neutralino.js.org/docs/api/overview/);
- [Microsoft Edge PWA overview](https://learn.microsoft.com/en-us/microsoft-edge/progressive-web-apps/);
- [Microsoft Edge PWA development model](https://learn.microsoft.com/en-us/microsoft-edge/progressive-web-apps/how-to/).

## Decision

Pass the M3 structural-simplicity gate, but only for the following product
boundary:

- Windows x64 static applications;
- portable directory and deterministic ZIP distribution without an application
  web server;
- no consumer compiler, Node.js, package manager, generated binding, or Actions
  cache upload;
- no runtime HTTP listener, WebSocket server, extension process, plugin scan,
  updater, or broad native API;
- one window and the closed IPC v1 method table.

This is a topology decision, not a claim that Actutum is a better or more mature
framework than Neutralinojs.

A PWA remains the default recommendation when HTTPS deployment,
browser-managed installation and updates, and browser capability policy are
acceptable. Actutum is justified only when a portable, offline-distributable,
deterministic, locally inspectable artifact is materially more important than
PWA reach or Neutralinojs capability breadth.

## Comparison Boundary

| Concern | Actutum MVP | Neutralinojs comparison | PWA counterargument |
| --- | --- | --- | --- |
| Consumer native compile | None | None for the prebuilt core path | None |
| Local content transport | WebView2 folder mapping | Embedded HTTP server | HTTPS origin plus browser cache |
| Native messaging | Direct WebView2 message | Local WebSocket | Browser APIs |
| Native surface | Six closed methods | Broad configurable namespaces and extensions | Browser-granted capabilities |
| Distribution | Portable directory and deterministic ZIP | Framework packaging workflow | Web deployment and browser-managed install |
| Platform reach | Windows x64 | Cross-platform | Browser and OS dependent, broadly cross-platform |

## Consequences

- The structural gate is complete without claiming a universal product win.
- Native API, local-server, extension, updater, and plugin requests remain
  outside core because adding them would erase the accepted distinction.
- PWA-suitable products should not adopt Actutum merely to obtain a desktop
  window.
- External user attempts remain required to prove that the narrow portable
  artifact use case has enough value for M5.

## Revisit Triggers

- Actutum adds a local listener, extension process, updater, sidecar, plugin
  system, or broad filesystem/OS namespace.
- Consumer builds require a compiler, Node.js, generated binding, or framework
  cache.
- A PWA packaging path provides the required offline portable artifact under
  the same deployment and inspection constraints.
- User attempts show that portable static applications do not need even the
  current six native methods.

## Synchronized Surfaces

- `README.md`
- `VALIDATION.md`
- `docs/product/00-product-brief.md`
- `docs/product/01-roadmap.md`
- `docs/product/02-spec.md`
- `docs/product/03-risk-register.md`
- `docs/adr/README.md`
