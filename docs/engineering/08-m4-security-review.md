# M4 Internal Security Review

- Status: Completed internal source review for the unsigned developer preview
- Date: 2026-07-18
- Owner: Project maintainer
- Review boundary: `0.5.10-alpha.1` candidate through commit
  `7d39d89b9ecfff339518b065ba78a13d69737160`

## Identity Recovery Refresh

On 2026-07-19 the maintained source, schemas, workflows, examples, and release
contracts were restored to the Velox identity under ADR 0015. The attempted
Actutum transition had changed identifiers and documentation but had not added
a native method, permission, asset transport, process boundary, network sink,
or publication authority. The recovery was therefore rechecked against the
same trust boundaries and findings below; it does not claim a new penetration
test or independent audit.

## Claim Boundary

This is a repository-internal source and contract review. It is not a
penetration test, independent audit, malware analysis, Authenticode review, or
claim that directory assets resist a local attacker. It satisfies the M5 input
requirement for a bounded security review; it does not itself satisfy release
acquisition or independent-use evidence. The repository-owned public-download
gate was later completed by run `29715002921`; the independent external-user
gate remains open.

## Reviewed Flows

### Build input to filesystem output

- Sources: manifest fields, asset paths and bytes, host metadata, output path.
- Transformations: strict JSON decoding, reverse-domain application identity,
  canonical containment, link and reparse rejection, content hashing, immutable
  build plan.
- Sinks: owned sibling staging directory, portable directory, deterministic
  ZIP, build report.
- Controls: `internal/manifest`, `internal/assettree`, `internal/buildplan`,
  `internal/builder`, `internal/archive`, and `internal/inspector` reject root
  escape, linked inputs, case collisions, alternate streams, reserved names,
  unexpected ZIP entries, oversized archives, and changed source bytes.

### Web content to native capability

- Actor: application HTML and JavaScript, treated as untrusted.
- Resource: application identity and one native window.
- Source: WebView2 top-level message from the application-specific virtual
  HTTPS origin.
- Sink: the six-method `internal/ipc.Dispatcher` table.
- Controls: top-level bridge injection, exact trusted host, remote navigation
  denial, frame denial, popup and download denial, global browser-permission
  denial, 64 KiB payload, depth 16, positive uint32 request IDs, duplicate ID
  rejection, 64 in-flight requests, closed permissions, and generic native
  errors.

### Release bytes to consumer execution

- Sources: tagged source checkout, two independently built release bundles,
  workflow artifacts, manual publication input, public GitHub Release assets.
- Sinks: prerelease assets and the downloaded `velox.exe`/host pair executed by
  a consumer job.
- Controls: pinned Actions, disabled checkout credentials, producer digest,
  byte-equality gate, exact artifact allowlist, checksum/SBOM/provenance
  generation, isolated `contents: write` publisher, no replacement, tag-to-
  manifest binding, independently supplied public ZIP digest, and a fixed
  public download host and file list.

## Findings

| ID | Severity | State | Finding and disposition |
| --- | --- | --- | --- |
| SEC-001 | Medium | Resolved | Runtime configuration rejected absolute paths but did not explicitly reject a Windows drive-relative volume such as `C:outside`. `internal/runtimeconfig.containedPath` now rejects every non-empty `filepath.VolumeName`, matching the authoring-manifest boundary, with a Windows regression test. |
| SEC-002 | High | Accepted for preview | External assets and `velox.runtime.json` remain mutable by a local writer. The preview prominently disclaims tamper resistance; sealing remains a separately benchmarked post-M5 decision. |
| SEC-003 | High | Monitoring after public verification | Public verifier run 29715002921 downloaded `v0.5.10-alpha.1` without checkout and matched separately recorded ZIP SHA-256 `5df53090e1e67ce54c8639f061ffc7b03b7c3aa38f95a725c29342cfaff73b68`. The unsigned release channel still does not authenticate publisher identity or protect against replacement of both assets and an out-of-band digest. |
| SEC-004 | Medium | Monitoring | The pure-Go WebView2 adapter owns COM and `unsafe.Pointer` lifetime risk. The binding is pinned and vendored, the production surface is bounded, and Windows lifecycle/security tests exercise it; no independent memory-safety review exists. |
| SEC-005 | Low | Accepted | The host binary exposes an explicit `--debug` launch flag. Web content cannot enable it, normal `velox run` does not pass it, and local process invocation is already inside the accepted local-writer boundary. |
| SEC-006 | Informational | Deferred | Authenticode identity and authenticated build attestation are absent by design for the unsigned preview. They are future-channel trust improvements, not claims made by M4. |

## Negative Evidence Reviewed

- Unknown fields, schema versions, permissions, methods, parameters, duplicate
  JSON keys and request IDs fail closed.
- Remote, suffix-confused, credential-bearing, port-bearing, non-HTTPS and
  malformed message origins are rejected.
- Child-frame navigation, top-level remote navigation, popups, downloads, and
  browser permission requests are exercised by the Windows security fixture.
- Runtime and archive metadata are bounded before allocation or decompression.
- Production runtime source guards reject listening socket APIs in the host,
  IPC, and WebView2 packages.
- Release publication cannot receive `contents: write` outside its isolated job
  and cannot replace an existing release.

## Privacy and Secret Review

The CLI and host have no telemetry, update request, crash upload, credential
store, or Velox service. Runtime diagnostics are local. The public verifier
uses fixed GitHub URLs and accepts only a constrained alpha tag and digest. No
repository secret is required for the unsigned preview. The WebView2 profile
contains application-owned browser data and remains outside release artifacts.

## Release Disposition

No unowned internal security finding blocks the explicitly unsigned developer
preview. SEC-002 is an advertised product limitation, SEC-003 passed its first
public-download control but remains an unsigned-channel monitoring risk, and
SEC-004 remains a maintenance and external-dependency risk for M5.

The release must still stop if current security tests fail, tag and artifact
bytes disagree, publication warnings disappear, a tracked secret is found, or
the public verifier cannot reproduce the candidate. An independent external
user attempt is still required before M4 completes.

## Re-review Triggers

- Any native method, target, asset transport, plugin, updater, sidecar, network
  proxy, or filesystem capability is added.
- The WebView2 binding or minimum supported runtime changes.
- A signed, installer, sealed-asset, beta, or stable channel is proposed.
- External attempts reveal a new path, policy, privacy, or reputation failure.
