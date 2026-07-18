# Security Baseline

- Status: Active
- Owner: Project maintainer

## Security Model

Actutum assumes that manifests, static assets, web content, IPC messages, output
paths, and downloaded release bundles can be malformed or malicious.

Local content is not trusted merely because it is stored beside the host.

## Trust Boundaries

- Build input to CLI.
- CLI staging output to promoted artifact.
- Downloaded Actutum release to local build.
- Web content to native host.
- Host to installed WebView2 Runtime.
- Packaged application to application-owned remote services.

## Threat Model

### Protected assets

- Source assets and manifests that must not be changed by a build.
- Previous successful output that must survive a failed build.
- Host and release integrity metadata.
- Native methods and window state reachable from untrusted web content.
- Local diagnostics, profile data, and configuration values.

### Attacker capabilities

The model includes malformed project files, hostile HTML or JavaScript,
cross-origin and child-frame content, forged IPC messages, path and archive
traversal, linked or redirected filesystem entries, and a tampered downloaded
release bundle. It does not claim to defeat an administrator or local attacker
who can replace installed directory assets or the external runtime config.

### Abuse cases and controls

| Abuse case | Control |
| --- | --- |
| Web content invokes undeclared native behavior | closed method table and permission check |
| A frame or remote page reaches the bridge | top-level virtual-origin enforcement and frame denial |
| IPC exhausts memory or parser work | 64 KiB payload, depth 16, uint32 IDs, and 64 in-flight requests |
| Browser features escape the product boundary | popup, download, permission, and remote-navigation denial |
| Project paths overwrite unrelated files | canonical containment, link/reparse checks, owned staging, atomic promotion |
| A release swaps the generic host | target, contract, size, and SHA-256 verification |
| Unsigned preview publication substitutes different bytes | unsigned equality gate, checkout-free consumer dependency, exact artifact allowlist, checksum verification, immutable tag, and no-replacement publication |
| A future signing step substitutes different bytes | provider request binding, signed manifest, Authenticode verification, and final artifact attestation |
| Production inspection exposes privileged tooling | development tools and default context menus disabled outside debug runs |

### Residual risk

WebView2 and the pinned Go binding remain external attack surfaces. Directory
assets are not sealed. The M2 tests prove the repository contract on the pinned
Windows runner; they are not an independent security audit or a claim of local
tamper resistance.

## Build Controls

- Validate every input path against a canonical root.
- Reject root escape, links and reparse-point escape, reserved names, alternate
  streams, case collisions, and unsafe archive entries.
- Never delete a path not created and owned by the current build.
- Assemble output in staging and promote after complete validation.
- Verify host and release checksums before use.
- Perform no network request during build.
- Never read or copy ambient credential files.

## Runtime Controls

- Use one application-specific virtual HTTPS origin.
- Keep file URL loading and origin rotation out of production; they are
  diagnostic controls and are not security-equivalent fallbacks under ADR
  0007.
- Accept IPC only from the expected top-level origin.
- Do not expose the bridge to frames.
- Deny remote top-level navigation, popups, downloads, and browser permission
  requests by default.
- Dispatch through a closed native method table.
- Require explicit permissions and deny unknown methods.
- Bound payload bytes, nesting, request identifiers, and in-flight requests.
- Disable production development tools unless a development-only path
  explicitly enables them.
- Open no listening socket.

## Excluded Native Capabilities

The MVP does not expose:

- Arbitrary filesystem access.
- Shell or process execution.
- Native network proxying.
- Dynamic plugins.
- Sidecars.
- Registry or credential-store access.
- Clipboard, global shortcut, tray, or unrestricted window APIs.

Adding one requires a threat-model update, ADR, permission contract, negative
tests, and performance impact evidence.

## Privacy

- No telemetry by default.
- No automatic crash upload.
- No automatic update check.
- No Actutum cloud service.
- Logs remain local and redact configuration values not required for repair.
- Application network and WebView2 profile data belong to the application.

## Known Limitation

Directory assets and external runtime configuration can be modified by an
attacker who can write to the installed application directory. The MVP does not
claim tamper resistance for those files.

An embedded or sealed profile is deferred and must be benchmarked separately.

## Release Blockers

- Origin or frame checks are missing.
- A native method bypasses the permission table.
- Untrusted paths can escape their roots.
- Build cleanup can delete pre-existing user files.
- Release artifacts are not checksummed.
- The unsigned developer preview is not labeled prominently, its checksum,
  SBOM, provenance, tag, or release-manifest bytes disagree, or publication can
  replace an existing release.
- A future signed channel claims publisher identity without a valid approved
  signature profile, or claims authenticated provenance without verified
  attestation evidence.
- Signing private material enters the repository, workflow secrets, artifacts,
  caches, or logs.
- Runtime failure exposes secrets, source contents, or native stack traces.
- A security control exists only in prose and has no executable negative test.

## Required Evidence

- Path and archive adversarial tests.
- IPC malformed-input and permission-denial tests.
- Navigation, popup, frame, and browser-permission tests.
- Missing-runtime and invalid-configuration failure tests.
- Dependency and release checksum verification.
- A current threat model before alpha.

Authenticode chain, publisher/profile, digest, timestamp, and authenticated
artifact-attestation verification are required only when a future signed or
authenticated channel makes those claims. The unsigned preview must not
simulate them.

The current bounded source review is recorded in
`docs/engineering/08-m4-security-review.md`. It is not an independent audit.

## Current Runtime Evidence

The Windows startup security fixture triggers and observes native policy blocks
for remote top-level navigation, child-frame navigation, popup creation,
downloads, and browser permission requests. The ordinary ready path proves that
messages from the generated application origin are accepted; unit tests reject
remote, suffix-confused, credential-bearing, port-bearing, non-HTTPS, and
malformed source URLs. The benchmark-only observer records policy names but not
URLs, payloads, or user data.
