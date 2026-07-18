# Risk Register

- Status: Active
- Owner: Project maintainer

## Scale

- Likelihood: low, medium, high.
- Impact: low, medium, high, critical.
- State: open, monitoring, mitigated, accepted, retired.

## Current Risks

| ID | Risk | Likelihood | Impact | State | Response and evidence gate |
| --- | --- | --- | --- | --- | --- |
| R-001 | Velox duplicates an existing compile-free wrapper | High | Critical | Mitigated | ADR 0008 passes only the smaller runtime/build topology and rejects a broad superiority claim; adding a local server, extensions, updater, or broad native API reopens the risk |
| R-002 | WebView2 initialization dominates startup | High | High | Mitigated | ADR 0009 removes startup from the headline while preserving fresh, settled warm, immediate-relaunch, failure, and regression evidence |
| R-003 | Pure-Go COM lifecycle code is unsafe or costly | Medium | Critical | Monitoring | Keep the WebView2 adapter bounded and reopen ADR 0005 only if required COM lifetime or security controls cannot be represented safely |
| R-004 | Feature requests recreate Tauri or Wails | High | Critical | Open | Enforce project invariants; require an ADR and metric impact for every new native surface |
| R-005 | Unchanged generic host prevents application branding | High | Medium | Accepted | Keep branding out of M0; revisit only after product viability |
| R-006 | Directory assets can be modified locally | High | High | Accepted | State the limitation prominently; defer sealed assets to a separately measured profile |
| R-007 | Hosted-runner noise produces false performance claims | High | High | Monitoring | Run 29569560999 controls the Wails claim with 10 successful samples per framework, one environment, balanced CPU allocation, paired non-overlapping jobs, and published p50 and p95; retain the controls for every future comparison |
| R-008 | Full benchmark CI consumes excessive Actions resources | Medium | Medium | Open | Run the cross-framework matrix only on schedule and release candidates |
| R-009 | Windows-only success does not transfer to other platforms | High | Medium | Accepted | Make no cross-platform promise before the Windows go-or-kill gate |
| R-010 | External WebView2 policy or runtime availability blocks users | Medium | High | Open | M0 records runtime versions; doctor must fail with actionable local diagnostics |
| R-011 | Working name conflicts with existing products or namespaces | High | Medium | Open | Treat Velox as a working name and complete naming review before public package release |
| R-012 | Benchmark targets become marketing theater | Medium | Critical | Open | Keep setup in headline timing; scope the current decision to the generated Velox-Wails pair artifact and make any future README numeric claim mechanically derived from published evidence |
| R-013 | A downloaded release bundle is tampered with | Low | Critical | Open | The alpha-evidence workflow emits checksums, SPDX, and unsigned provenance and verifies same-run artifacts without checkout. ADR 0010 separates authenticated final-artifact provenance from Authenticode; keep the risk open until both are implemented and a public download is verified independently |
| R-014 | Static-only scope has too little user value | Medium | Critical | Open | ADR 0008 limits the answer to offline portable deterministic artifacts; require external user attempts to prove that distinction matters before M5 |
| R-015 | Virtual HTTPS and same-UDF relaunch ownership create a controller-startup tail | High | High | Monitoring | Keep file URL diagnostic-only under ADR 0007; publish the delay, UDF, origin, browser-process, and phase recovery matrix before changing transport or adding a workaround |
| R-016 | Signing-provider approval, availability, or publisher identity blocks alpha | Medium | High | Open | Use SignPath Foundation only after project and policy approval; retain Microsoft Artifact Signing as the migration candidate and never fall back to a repository-held private key |
| R-017 | Signing obscures the reproducible unsigned lineage | Medium | Critical | Monitoring | `velox.signing-record/v1` and its dry-run verifier bind the unsigned files, signing-input ZIP, provider-output placeholders, final manifest and ZIP, checksums, and SBOM while remaining non-publishable; keep monitoring until real provider output, Authenticode verification, and final attestations exercise release mode |
| R-018 | Provider token, signing policy, or certificate is compromised | Low | Critical | Open | Keep private keys provider-held, scope the API token to a protected environment, separate publication permission, record request identities, and define token revocation, policy suspension, affected-release discovery, and consumer notice before alpha |

## Review Rules

- Every roadmap milestone reviews open critical risks.
- A release cannot proceed with an unowned critical risk.
- A risk becomes mitigated only when the named evidence exists.
- Deferred functionality does not count as mitigation.
- New native APIs, targets, signing modes, or update paths add a risk entry
  before implementation.

## Source of Truth

Product scope and stop conditions remain in docs/product/02-spec.md. This
register records uncertainty and treatment; it does not expand product scope.
