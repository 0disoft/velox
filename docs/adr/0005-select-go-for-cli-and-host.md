# ADR 0005: Select Go for both the CLI and production host

- Status: Accepted
- Date: 2026-07-10
- Owner: Project maintainer

## Context

M0 compared a pure-Go WebView2 spike with a direct C++23 reference host. The
C++ reference produced a smaller native artifact and slightly lower local
fresh-profile startup measurements, but it added a second maintainer language,
toolchain, dependency lock, SDK discovery path, and release build surface. The
startup difference was not large enough or stable enough to outweigh that
operating cost, and the C++ reference also exposed an unresolved immediate
same-profile relaunch delay.

Velox primarily wins by shipping an unchanged prebuilt host and avoiding
consumer-side native compilation. The host implementation language is not the
product moat.

## Decision

Use Go for both the CLI and the production Windows host.

The production host must remain pure Go from the repository's perspective:

- no CGo;
- no application-specific native compilation;
- no required C++ shim in consumer or maintainer Go builds;
- UI and COM operations remain on the owning OS thread;
- COM ownership, callback lifetime, HRESULT handling, and shutdown behavior
  are explicit and covered by focused tests;
- WebView2 remains behind a repository-owned adapter boundary.

This decision selects the implementation language. It does not promote the
current `github.com/jchv/go-webview2` M0 wrapper to the production security
boundary. The project must separately choose between a narrow maintained fork
and a lower-level repository-owned pure-Go WebView2 adapter.

## Consequences

- CLI and host share one language, formatter, test toolchain, module graph,
  debugging model, and release build path.
- Maintainer CI does not need a C++ toolchain for normal product builds.
- The generic host can still be built once and copied without a Go toolchain in
  consumer application builds.
- The project accepts a larger host executable than the C++ reference unless
  later measurements justify size-specific work.
- COM and WebView2 interfaces require careful pure-Go ABI code and cannot be
  hidden behind an underpowered convenience wrapper.
- C++23 and Pixi remain M0 reference infrastructure only until their benchmark
  evidence is moved or retired; they are not product build dependencies.

## Migration

1. Keep the current Go wrapper confined to the M0 executable.
2. Define the minimum WebView2 adapter interface from the security and runtime
   contracts, not from the wrapper's existing API.
3. Implement one vertical slice covering virtual HTTPS assets, trusted-origin
   messaging, denied permissions, denied remote navigation, and clean shutdown.
4. Run the existing fresh, warm, and immediate-relaunch fixtures against the
   new adapter.
5. Remove the M0 wrapper after the adapter passes those fixtures.
6. Move or retire C++23/Pixi reference infrastructure after the Go lifecycle
   baseline is stable on pinned CI.

Rollback before the adapter becomes a public runtime contract means returning
to the isolated M0 wrapper while fixing the repository-owned adapter. Rollback
does not change the selected production language.

## Revisit Triggers

Revisit this decision only if a pure-Go implementation cannot safely represent
required COM lifetimes or required WebView2 security controls after a bounded
adapter spike. Artifact size or a small startup difference alone is not enough
to reopen the language choice.
