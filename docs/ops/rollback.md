# Rollback

- Status: Pre-implementation
- Owner: Project maintainer

## Scope

Velox rollback means selecting a previous immutable release bundle. There is no
service deployment, database migration, account state, or server-side data
rollback.

## Triggers

- Published checksum or provenance mismatch.
- A release claims Authenticode but has a missing, invalid, revoked, or wrongly
  scoped signature.
- Signing record does not bind the published bytes to the verified unsigned
  inputs and approved provider request.
- CLI or host fails the published hello smoke test.
- New release requires an undeclared consumer toolchain.
- Startup, security, or output integrity regresses materially.
- Compatibility metadata is incorrect.
- A critical vulnerability affects the released artifact.

## Decision Flow

1. Stop promotion and mark the affected release as unsuitable.
2. Preserve evidence and do not overwrite release artifacts.
3. Identify the last verified immutable release.
4. Point documentation or distribution metadata to that release only through an
   explicit reviewed change.
5. Re-run download, checksum, inspect, build, and startup verification.
6. For a signed channel, re-run artifact-attestation and Authenticode
   verification. Do not require those checks for an explicitly unsigned
   developer preview.
7. Revoke exposed provider tokens, suspend the signing policy, or contact the
   provider when credential or certificate integrity is involved.
8. Publish a concise limitation or incident notice when users may be affected.

## Consumer Recovery

Consumers pin an exact known-good Velox release and checksum. Velox does not
silently downgrade itself or perform an automatic update.

Artifacts built with the affected version may need rebuilding. The
compatibility decision depends on which contract or host behavior changed.

## Forward Fix

A fixed release receives a new immutable version. Existing release bytes and
tags are never replaced.

## Data Policy

Velox owns no authoritative user data. WebView2 profile and application
business-data recovery belong to the packaged application and are outside this
runbook.

## Current Gap

ADR 0011 and `docs/ops/release.md` define the unsigned developer-preview
boundary. The repository implements a guarded publication job but has not
exercised it against a public tag. ADR 0010 and `docs/ops/signing.md` define the
dormant future signed-channel boundary; no signed artifact or provider
onboarding exists.
