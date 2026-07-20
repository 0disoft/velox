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
| R-003 | Pure-Go COM lifecycle code is unsafe or costly | Medium | Critical | Monitoring | The M4 internal review records the pinned vendored binding and missing independent memory-safety review; keep the adapter bounded and reopen ADR 0005 only if required COM lifetime or security controls cannot be represented safely |
| R-004 | Feature requests recreate Tauri or Wails | High | Critical | Open | Enforce project invariants; require an ADR and metric impact for every new native surface |
| R-005 | Unchanged generic host prevents application branding | High | Medium | Accepted | Keep branding out of M0; revisit only after product viability |
| R-006 | Directory assets can be modified locally | High | High | Accepted | State the limitation prominently; defer sealed assets to a separately measured profile |
| R-007 | Hosted-runner noise produces false performance claims | High | High | Monitoring | Run 29569560999 controls the Wails claim with 10 successful samples per framework, one environment, balanced CPU allocation, paired non-overlapping jobs, and published p50 and p95; retain the controls for every future comparison |
| R-008 | Full benchmark CI consumes excessive Actions resources | Medium | Medium | Mitigated | The public benchmark repository limits the full cross-framework zero-cache matrix to weekly schedules, benchmark-candidate tags, or explicit manual dispatch; pull requests run contract checks, and recommended-cache runs use bounded keys with cleanup. Reopen if recurring frequency or retained cache scope expands |
| R-009 | Windows-only success does not transfer to other platforms | High | Medium | Accepted | Make no cross-platform promise before the Windows go-or-kill gate |
| R-010 | External WebView2 policy or runtime availability blocks users | Medium | High | Mitigated | `doctor` enforces Windows 10 1709 or Server 2016 and WebView2 `92.0.902.49`, returns stable prerequisite errors, and the public-download verifier exercised the supported path. Reopen when the compatibility floor or runtime acquisition policy changes |
| R-011 | Velox conflicts with existing products and an executable namespace | High | High | Accepted | The collision review found Meta Velox plus a released Go CLI that already ships `velox.exe`; ADR 0015 records the maintainer's decision to retain Velox, disclose the risk, and revisit before package-manager or commercial distribution |
| R-012 | Benchmark targets become marketing theater | Medium | Critical | Open | Keep setup in headline timing; scope the current decision to the generated Velox-Wails pair artifact and make any future README numeric claim mechanically derived from published evidence |
| R-013 | A downloaded release bundle is tampered with | Low | Critical | Monitoring | Public verifier run 29715002921 downloaded `v0.5.10-alpha.1` without checkout and matched independently recorded ZIP SHA-256 `5df53090e1e67ce54c8639f061ffc7b03b7c3aa38f95a725c29342cfaff73b68` before exercising it. ADR 0011 does not pretend co-located checksums authenticate a compromised release channel; retain the unsigned-channel limitation and revisit authenticated attestations for a broader channel |
| R-014 | Static-only scope has too little user value | Medium | Critical | Open | ADR 0008 limits the answer to offline portable deterministic artifacts; require external user attempts to prove that distinction matters before M5 |
| R-015 | Virtual HTTPS and same-UDF relaunch ownership create a controller-startup tail | High | High | Monitoring | Keep file URL diagnostic-only under ADR 0007; publish the delay, UDF, origin, browser-process, and phase recovery matrix before changing transport or adding a workaround |
| R-016 | Unsigned preview warnings or managed-device policy block adoption | High | High | Accepted | ADR 0011 limits the first release to a developer preview, requires prominent SmartScreen and managed-device warnings, and uses the external-user attempt to decide whether signing becomes necessary |
| R-017 | Future signing obscures the reproducible unsigned lineage | Medium | Critical | Monitoring | `velox.signing-record/v1` and its dry-run verifier bind the unsigned files, signing-input ZIP, provider-output placeholders, final manifest and ZIP, checksums, and SBOM while remaining non-publishable; keep the tooling dormant until an ADR 0011 adoption trigger justifies real provider output and release-mode evidence |
| R-018 | Future provider token, signing policy, or certificate is compromised | Low | Critical | Monitoring | No active provider credential or signing workflow exists. If signing resumes, keep private keys provider-held, scope the API token to a protected environment, separate publication permission, record request identities, and define revocation and consumer notice before activation |

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
