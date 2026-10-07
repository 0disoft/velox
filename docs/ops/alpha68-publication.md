# Alpha.68 Publication: 2026-10-07

## Identity

- Velox `v0.5.10-alpha.68`, an unsigned prerelease, was published at
  `2026-10-07T13:40:00Z` from `5ffedc8d6e17b16e6e13ca8c57f21c06d76dfe8c`.
- Release ID `405805188`, `draft: false`, `prerelease: true`.
- Annotated tag object: `34f679552d049d6c6d83ac8c835033959a6aff88`.
- [Tag CI 37628664660](https://github.com/0disoft/velox/actions/runs/37628664660)
  attempt 1 passed two independent reproducible producer builds and basic
  checkout-free consumer packaging with Go 1.26.0 on `windows-2025`.
- The same four verified files were reused, without a rebuild, rerun,
  workflow dispatch or tag move. The prior tag-CI publish job stayed skipped.

The [public release](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.68)
was created with `gh release create --prerelease --verify-tag --latest=false`.
Notes preserve prior unsigned-channel disclosures and update the feature
section, CI/source identity and independently recorded ZIP digest.

## Published Scope

- Read-only template discovery for four starters, including JSON/quiet modes.
- Expanded top-level/init help and shell-quoted next-step run/build hints.
- Literal text-editor find with case toggle, match counts and wrap navigation.
- Empty-query Escape/X closing with pending/stale IME state handling.

The editor addition is web-only. Existing generated projects are unchanged.
Manifest-watch notices, tray generation, image/font watch, opt-in debug
diagnostics and draft recovery from earlier releases remain available.

## Public Asset Verification

| Asset | Bytes | SHA-256 |
| --- | --- | --- |
| `velox-windows-x64.zip` | 5,923,846 | `b8e916c41c0d9fa6bc8ce21c6254e8adab8121c37d183da7a9e2cea99ba05bf8` |
| `checksums.sha256` | 279 | `b240f1193217d4bb69a5908cb304931b8cdcb56d1386e517d65b4b82e333fa4c` |
| `velox-windows-x64.spdx.json` | 9,618 | `5072791fdd29421385a79beba827ee84b13dbd5ab5dc12e80f8b6fe6746bc623` |
| `velox-windows-x64.intoto.jsonl` | 744 | `0a185eed5d1c81a2ac0fbf640ebbeaf347ab73a7b7718f6dc83dad05f68b88c3` |

All four unauthenticated public downloads matched the verified CI bytes by
size and SHA-256 and matched GitHub's asset digests. The public annotated
tag resolved to the verified source. No authentication headers were sent to
the public API or asset URLs.

The earlier CI artifact verification covers 16 manifest entries, 17 SPDX
file SHA-256 entries, three checksum entries, source/run/ZIP provenance,
alpha.68 CLI version, Go 1.26.0 clean-source metadata and PE subsystems
(host/Setup GUI 2, CLI console 3); see
[alpha68-tag-evidence.md](alpha68-tag-evidence.md). Host size remains
4,416,000 bytes, equal to public alpha.67, with a different digest. CLI size
is 4,596,736 bytes, 21.5 KiB larger than public alpha.67. These are not the
earlier local Go 1.27.1 candidate bytes.

Receipts under `dist/candidates/alpha68-ci-37628664660/`:
`ci-verification.json`, `public-verification.json`, `release.json` and
`consumer/consumer-clean.json`. Generated assets and receipts are ignored,
not committed. CommandCode DeepSeek 4.1 Flash/high drafted the checked
release-note feature section; its separate publication-summary request
returned no final text, so this record uses verified receipts directly.

## Boundaries

No native UI, startup, picker, tray/balloon, find/draft/watch interaction,
installer execution, hosted public-verifier dispatch, stress or performance
test was added against these published bytes. The CI consumer is a basic
template packaging check, not a new four-template native check. Historical
manual/native receipts remain historical.

Evidence remains `same-repository-public-download` with
`externalUserAttempt: false`. Executables are unsigned; SPDX and provenance
are not authenticated attestations. Beta remains held. No host/IPC API,
database/storage schema, permission, dependency, repository hygiene or CI
workflow/job-scheduling change is included.

The documentation follow-up passed `go test ./tests/hygiene`, scoped
`go vet ./tests/hygiene` and `git diff --check`. The current-channel hygiene
test now checks published alpha.68 and retains the local preparation as
historical; repository hygiene configuration is unchanged.
