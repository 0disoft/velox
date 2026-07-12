# Rollback

- Status: Pre-implementation
- Owner: Project maintainer

## Scope

Velox rollback means selecting a previous immutable release bundle. There is no
service deployment, database migration, account state, or server-side data
rollback.

## Triggers

- Published checksum or provenance mismatch.
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
6. Publish a concise limitation or incident notice when users may be affected.

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

No promotion metadata, release workflow, or rollback automation exists. Exact
operator commands remain UNDECIDED until release tooling is implemented and
tested.
