# M4 Internal Security Review

- Status: Completed internal source review for the unsigned developer preview
- Date: 2026-07-18
- Owner: Project maintainer
- Original review boundary: `0.5.10-alpha.1` working-name candidate through
  commit `7d39d89b9ecfff339518b065ba78a13d69737160`
- Actutum identity refresh: `0.6.0-alpha.1` product source through commit
  `f6e7677f77abf60eb22af2f273d80727aacdc18f`

## Claim Boundary

This is a repository-internal source and contract review. It is not a
penetration test, independent audit, malware analysis, Authenticode review, or
claim that directory assets resist a local attacker. It satisfies the M5 input
requirement for a bounded security review; it does not close the separate M4
public-download or external-user gates.

The Actutum identity refresh rechecked the source and workflow changes from the
original boundary through the named commit. That range changes product,
command, module, manifest, environment, bridge, schema, and release identity,
but adds no native capability, target, asset transport, or trust boundary. The
review below therefore applies to the selected Actutum candidate while the
original maintenance-cost snapshot remains immutable historical evidence.

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
- Sinks: prerelease assets and the downloaded `actutum.exe`/
  `actutum-host.exe` pair executed by a consumer job.
- Controls: pinned Actions, disabled checkout credentials, producer digest,
  byte-equality gate, exact artifact allowlist, checksum/SBOM/provenance
  generation, isolated `contents: write` publisher, no replacement, tag-to-
  manifest binding, independently supplied public ZIP digest, and a fixed
  public download host and file list.

## Findings

| ID | Severity | State | Finding and disposition |
| --- | --- | --- | --- |
| SEC-001 | Medium | Resolved | Runtime configuration rejected absolute paths but did not explicitly reject a Windows drive-relative volume such as `C:outside`. `internal/runtimeconfig.containedPath` now rejects every non-empty `filepath.VolumeName`, matching the authoring-manifest boundary, with a Windows regression test. |
| SEC-002 | High | Accepted for preview | External assets and `actutum.runtime.json` remain mutable by a local writer. The preview prominently disclaims tamper resistance; sealing remains a separately benchmarked post-M5 decision. |
| SEC-003 | High | Open until public verification | An unsigned ZIP and checksum downloaded from the same compromised release channel do not authenticate each other. The manual public verifier requires a separately recorded SHA-256, but that control remains unexercised until the release exists. |
| SEC-004 | Medium | Monitoring | The pure-Go WebView2 adapter owns COM and `unsafe.Pointer` lifetime risk. The binding is pinned and vendored, the production surface is bounded, and Windows lifecycle/security tests exercise it; no independent memory-safety review exists. |
| SEC-005 | Low | Accepted | The host binary exposes an explicit `--debug` launch flag. Web content cannot enable it, normal `actutum run` does not pass it, and local process invocation is already inside the accepted local-writer boundary. |
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
store, or Actutum service. Runtime diagnostics are local. The public verifier
uses fixed GitHub URLs and accepts only a constrained alpha tag and digest. No
repository secret is required for the unsigned preview. The WebView2 profile
contains application-owned browser data and remains outside release artifacts.

## Release Disposition

No unowned internal security finding blocks the explicitly unsigned developer
preview. SEC-002 is an advertised product limitation, SEC-003 remains a public
release verification gate, and SEC-004 remains a maintenance and external-
dependency risk for M5.

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
