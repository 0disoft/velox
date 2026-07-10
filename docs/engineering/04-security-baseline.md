# Security Baseline

- Status: Draft
- Owner: Project maintainer

## Security Model

Velox assumes that manifests, static assets, web content, IPC messages, output
paths, and downloaded release bundles can be malformed or malicious.

Local content is not trusted merely because it is stored beside the host.

## Trust Boundaries

- Build input to CLI.
- CLI staging output to promoted artifact.
- Downloaded Velox release to local build.
- Web content to native host.
- Host to installed WebView2 Runtime.
- Packaged application to application-owned remote services.

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
- No Velox cloud service.
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
- Runtime failure exposes secrets, source contents, or native stack traces.
- A security control exists only in prose and has no executable negative test.

## Required Evidence

- Path and archive adversarial tests.
- IPC malformed-input and permission-denial tests.
- Navigation, popup, frame, and browser-permission tests.
- Missing-runtime and invalid-configuration failure tests.
- Dependency and release checksum verification.
- A current threat model before alpha.
