# ADR 0017: Continue as a narrow static desktop packager

- Status: Accepted
- Date: 2026-07-21
- Owner: Project maintainer
- Superseded in part by: ADR 0018 replaces the beta and stable admission evidence clause

## Context

M4 proved that a clean Windows runner can acquire a Velox release and package
an application without checking out Velox source or invoking a consumer
compiler, frontend package manager, or GitHub Actions cache. M3 also recorded a
material cold-build advantage over the controlled Wails fixture and ADR 0008
accepted only the smaller portable static-app topology.

The remaining M5 question is a product decision, not another build-system
check. Velox still has zero qualifying independent-user attempts. That is
negative market evidence. Maintainer-created examples and hosted workflows can
prove capability, repeatability, and maintenance boundaries, but cannot prove
demand or documentation usability for an unrelated developer.

Two new maintainer-owned dogfood surfaces narrow the technical uncertainty:

- `examples/capability-probe` records browser API availability separately from
  actual operation outcomes such as passed, canceled, blocked, or failed.
- `examples/file-notes` uses browser-owned file pickers and IndexedDB with an
  empty native permission set. Its document state has one owner, saved state is
  a baseline rather than a second editable copy, and draft restoration
  completes before the startup-ready marker.

Local model tests and deterministic startup smokes passed for both examples.
Hosted Alpha release evidence
[run 29806946109](https://github.com/0disoft/velox/actions/runs/29806946109)
then passed for commit
`d8495b8aa2a399505b583a8ed881b5bc7fa9f304`: the reproducible unsigned
release job and checkout-free consumer job succeeded, while publication was
disabled and skipped. These results prove the current narrow path still works.
They do not create an independent user.

## Decision

Continue Velox toward a stable **narrow static desktop packager**, but keep the
current lifecycle at alpha until the beta admission gate below is satisfied.

The approved product boundary remains:

- static HTML, CSS, and JavaScript;
- an unchanged prebuilt Go host and Go CLI;
- browser-owned storage and file workflows where the installed WebView2 runtime
  exposes them;
- the existing closed native IPC table for application information and basic
  window lifecycle;
- portable directory and deterministic ZIP output on Windows x64.

This decision does **not** approve an application-specific Go backend, native
filesystem, shell, process, sidecar, plugin, local-server, updater, installer,
asset-sealing, new-platform, or broad IPC surface. A request for any of those
must open a new product and threat-model ADR before implementation.

Beta or stable admission requires either:

1. at least one qualifying independent-user attempt under the existing
   external-attempt contract; or
2. a later ADR that explicitly accepts the zero-adoption risk and names the
   evidence that justifies shipping anyway.

Until then, maintainer dogfooding may improve correctness, examples,
documentation, deterministic packaging, and browser-owned workflows inside the
approved boundary. It may not manufacture adoption evidence or widen the native
surface to make the product look more complete.

## Alternatives

### Add a Go application backend now

Rejected. Application-specific native compilation would erase the central
consumer-build advantage. A prebuilt generic backend with broad methods would
recreate the capability, permission, and maintenance surface Velox deliberately
removed.

### Reposition immediately as a general Tauri or Wails replacement

Rejected. The evidence supports a much smaller topology, not ecosystem breadth.
The same WebView2 runtime also limits any claim of a universal startup
advantage.

### Reposition only around CI packaging and reproducibility

Retained as a fallback. If independent attempts value release assembly but not
the runtime, a later ADR may split or reposition the product around that proven
surface.

### Stop the project now

Rejected for this decision horizon. Technical distribution, deterministic
packaging, and browser-owned local workflows are working, and the next alpha
experiments remain small and reversible. Zero independent use still blocks a
stronger channel decision.

### Treat maintainer dogfood as adoption

Rejected. It is valuable product evidence but has the same owner, assumptions,
and incentives as the implementation.

## Consequences

### Positive

- Product work can continue without waiting for an uncontrolled social event.
- The native attack and maintenance surface stays closed and measurable.
- File-oriented apps can be tested through browser-owned capabilities before
  any native API is proposed.
- Beta wording remains honest about the absence of outside demand evidence.

### Negative

- Apps that require a Go backend or deep OS integration remain unsupported.
- Browser file APIs vary by runtime policy and still require real
  user-gesture testing.
- The narrow target market may remain too small; R-014 stays open.
- Alpha work can continue longer without resolving whether another developer
  will choose the product.

## Validation

- `velox_test`, `velox_lint`, and `velox_format` pass on the decision commit.
- `velox_capability_probe_model_test` and
  `velox_capability_probe_smoke` pass.
- `velox_file_notes_model_test` and `velox_file_notes_smoke` pass.
- Hosted run `29806946109` remains successful for exact source commit
  `d8495b8aa2a399505b583a8ed881b5bc7fa9f304` with publication skipped.
- Hygiene tests require this decision, the zero-independent-user boundary, and
  the native-scope prohibition to remain synchronized.

Manual file open, write, reopen, permission-denial, and picker-cancellation
checks remain user-gesture evidence. Automated startup smoke does not claim
those operations passed.

## Rollback or Fallback

If the examples require native capabilities to provide their claimed workflows,
remove the unsupported example behavior and return to the last supported static
surface. Do not add a hidden native bridge as a repair.

If maintenance cost or independent feedback rejects the runtime while valuing
the release path, propose the CI packaging and reproducibility repositioning in
a new ADR. If neither surface attracts a qualifying attempt, stop feature work
and archive the project without weakening the evidence record.

## Revisit Triggers

- A qualifying independent-user attempt is recorded.
- Beta, stable, package-manager, or commercial distribution is proposed.
- A Go application backend or any new native capability is proposed.
- Browser-owned file workflows fail on the supported WebView2 floor.
- Maintenance cost materially exceeds the bounded M5 record.
- The portable runtime no longer has a meaningful distinction from a PWA or an
  existing compile-free wrapper.

## Synchronized Surfaces

- `README.md`
- `VALIDATION.md`
- `docs/README.md`
- `docs/adr/README.md`
- `docs/ops/00-operational-contract.md`
- `docs/product/01-roadmap.md`
- `docs/product/02-spec.md`
- `docs/product/03-risk-register.md`
- hygiene tests
