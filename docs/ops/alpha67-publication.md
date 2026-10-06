# Alpha.67 Publication: 2026-10-06

## Identity

- Velox `v0.5.10-alpha.67`, an unsigned prerelease, was published on 2026-10-06
  from `cb801f5bf3014b31a2a544d97bb0fe95d33ff270`.
- [Tag CI 37471273685](https://github.com/0disoft/velox/actions/runs/37471273685)
  attempt 1 passed two independent producer builds and a basic checkout-free
  consumer smoke with Go 1.26.0 on `windows-2025`.
- Four verified artifacts were published without another hosted build; all
  four unauthenticated public downloads matched CI hashes and GitHub digests.
- Scope: restart-required manifest-watch notices, stderr notices in JSON mode,
  and the opt-in tray-app starter. No beta, native UI, installer execution,
  performance/stress or adoption claim follows from this publication.

## Publication

The owner authorized one atomic push of four prepared commits and annotated
tag `v0.5.10-alpha.67`. Remote main and the peeled tag matched `cb801f5`.
Annotated tag object: `6e898ccc8367ef5f4ebd0777f49633e2a63dbb76`.
The tag triggered one evidence run; its publication job was skipped.

`gh release create --prerelease --verify-tag --latest=false` reused the run's
verified files without a second producer or tag move. Release ID `404753374`,
`draft: false`, `prerelease: true`, published `2026-10-06T13:34:41Z`.
The [public release](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.67)
retains unsigned-channel warnings and describes the three new CLI/template
features. Notes reuse the prior public release's disclosures; the feature
paragraph and independently recorded ZIP digest were updated.

| Public asset | Bytes | SHA-256 |
| --- | --- | --- |
| `velox-windows-x64.zip` | 5,910,481 | `fcb5e807682df028515f95d40146235b29d3ed7de32d41c09f00b1d4cf9afead` |
| `checksums.sha256` | 279 | `d6ac771cbd847f16f6d3b02bbbef3ffc6512ab7dab44fdbaa1759f2566cb312d` |
| `velox-windows-x64.spdx.json` | 9,618 | `b80215dfe455e1a8e119cc3e259c11840aaaff4e15e391f339a81e511f96ef90` |
| `velox-windows-x64.intoto.jsonl` | 744 | `7b6189d024b980ea08d09d6b8cee9ff03894b892f1b1264084d6d2c2bd4062db` |

## Verified Scope

CI consumer evidence invokes only `velox.exe`: init, validate, doctor, two
builds and inspect without checkout. Both app ZIPs hashed
`b1e1529c23a976b9a1e1f7fd41e78bc4a66b2d8f13677aca8ea4cdeeb020e0b9`.
Runner images can contain toolchains; none was invoked by that consumer step.

Local verification downloaded the CI files, checked three checksum entries,
16 release-manifest artifacts, 17 SPDX file SHA-256 entries, version/target,
and provenance source/run `37471273685/1`. The CI-extracted CLI reported
`0.5.10-alpha.67`; `go version -m` confirmed Go 1.26.0 and clean source
`cb801f5`. Public downloads then matched every verified CI file byte-for-byte
by size and SHA-256, including GitHub metadata digests. No authentication
headers were sent to the public API or asset URLs.

The public Go 1.26.0 host is 4,416,000 bytes, the same size as the public
alpha.66 host; its digest differs. The CLI is 4,574,720 bytes. These are not
the prior Go 1.27.1 local candidate bytes; its native/manual evidence remains
historical and is not relabeled as published-byte evidence.

Receipts: `dist/candidates/alpha67-ci-37471273685/ci-verification.json`,
`public-verification.json`, and `consumer/consumer-clean.json`.
CommandCode DeepSeek 4.1 Flash/high drafted the checked summary bullets.

## Boundaries

No native UI, startup, picker, tray/balloon, draft, development-watch,
installer execution, hosted public-verifier dispatch, stress or performance
test was added against these published bytes. The consumer packaging step is
basic-template evidence, not a fresh four-template or native behavior check.
Evidence remains `same-repository-public-download` with
`externalUserAttempt: false`. Executables are unsigned; SPDX and provenance
are not authenticated attestations. Beta remains held. No API/IPC, database,
schema, dependency or CI workflow/job-scheduling change is included.
