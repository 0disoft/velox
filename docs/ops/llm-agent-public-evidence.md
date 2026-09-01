# Public Clean-Room LLM Agent Evidence

- Status: Active publication contract
- Owner: Project maintainer
- Decision: ADR 0018
- Schema: `schema/llm-agent-public-evidence-v1.schema.json`
- Verifier: `scripts/verify-llm-agent-public-evidence.ts`

## Purpose

A qualifying clean-room series is produced from private local trial roots,
external attestations, sandbox receipts, and a temporary isolated evaluator
database. None of those private locations should be required to verify the
public claim.

This contract reduces a passing series to one public-only JSON packet that
preserves the release, task, model, sequence, sandbox, and outcome identities
needed to review beta technical readiness. It does not publish raw sessions,
prompts, transcripts, tool payloads, local paths, credentials, environment
values, databases, screenshots, or hidden reasoning. It also keeps
`humanAdoptionClaim` fixed to `false`.

## Canonical Location

Publish exactly one packet as an asset on the same immutable GitHub Release that
was evaluated. The canonical asset name is:

```text
velox-llm-agent-evidence-<series-id>-<payload-sha256>.json
```

The packet's `release.url` must name that release tag exactly. A mirror, issue
attachment, workflow artifact, branch file, gist, or local directory is not the
canonical publication.

The asset has permanent retention. Do not delete it, replace it, or upload new
bytes under the same name. A correction starts a new series and produces a new
asset while the old asset remains available. The readiness record created after
publication must preserve the release tag, asset browser URL, GitHub asset ID,
publication timestamp, and packet SHA-256 so deletion and re-upload cannot be
mistaken for continuity.

## Immutable Identity

`payloadSha256` is SHA-256 over the packet's `payload` encoded as canonical JSON:
object keys sorted lexicographically, arrays kept in order, no insignificant
whitespace, and ordinary JSON scalar encoding. The digest appears in the asset
name together with the series ID.

The verifier recomputes the digest and derives the expected filename. Altering a
public field without updating the digest fails. Updating the digest changes the
asset identity and therefore cannot silently replace the recorded packet.

## Public Packet Contents

The top level contains only the schema version, canonical asset name, payload
digest, and payload. The payload contains:

1. UTC publication time and permanent-retention declaration.
2. Series ID, passing outcome, true beta technical gate, and false human
   adoption claim.
3. Exact Velox repository, release tag, release URL, and release ZIP SHA-256.
4. Public task schema, immutable source commit, raw task URL, and task SHA-256.
5. Required v2 attestation, sandbox receipt, sandbox policy, AppContainer
   filesystem boundary, no-breakaway Job Object boundary, and network
   capability identities.
6. Exactly three unique trial IDs and sequences 1 through 3, each with provider,
   model, passing outcome, result digest, attestation digest, and sandbox receipt
   digest.
7. At least two distinct provider and model pairs.

The public packet deliberately excludes session identifiers even when hashed.
Hashed session IDs remain local verification bindings and are not needed to
review the published series identity.

## Forbidden Material

The verifier rejects unknown fields and recursively rejects private field names
or values associated with prompts, transcripts, session IDs, local and workspace
paths, trial roots, evaluator databases, credentials, API tokens, private
reasoning, chain of thought, attestation projections, environment variables, and
command lines. It also rejects recognizable absolute paths and credential
shapes wherever they appear as strings.

A packet that needs any private trial root or attestation directory to verify is
not a public packet.

## Compatibility

Version 1 accepts only:

```text
velox.llm-agent-public-evidence/v1
velox.llm-agent-task/v1
velox.llm-agent-evaluation-attestation/v2
velox.eval-sandbox-receipt/v1
velox.eval-sandbox-policy/v1
```

An older v1 session-log-only attestation cannot qualify. A future incompatible
schema gets a new public packet version and explicit verifier support; the v1
verifier remains fail-closed rather than guessing an upgrade.

## Publication Procedure

First verify the private three-trial series with the existing trial and series
verifier. Then construct the public packet from verified identities and digests,
validate it against the JSON schema, and run:

```text
bun scripts/verify-llm-agent-public-evidence.ts <public-packet.json>
```

Upload the exact verified file once to the evaluated release. Download that
release asset through its public URL into a clean directory and run the same
command again without access to the private series root, attestation directory,
evaluator database, or maintainer credentials. Record the returned series ID,
release tag, asset name, payload digest, model identities, release asset ID, and
public URL in the beta readiness decision.

Publication is incomplete until the public download passes. A successful local
file does not prove that the release asset contains the same bytes.

## Rejection Contract

The verifier fails on malformed or oversized JSON, missing required fields,
unknown fields, incompatible schema identities, invalid release or task URLs,
invalid digests, duplicate trial IDs, duplicate or incomplete sequences, fewer
than two model identities, non-passing outcomes, non-permanent retention,
forbidden private material, payload mutation, and filename mismatch.

`tests/fixtures/llm-agent-public-evidence/valid.json` is synthetic contract data,
not release evidence. The sibling negative fixtures independently exercise
missing, altered, duplicate, incompatible, and forbidden-private-field
rejections.
