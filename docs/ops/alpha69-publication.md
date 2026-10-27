# Alpha.69 Published Preview: 2026-10-28

## Release Identity

Unsigned prerelease
[`v0.5.10-alpha.69`](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.69)
is release ID `425812692`: `draft: false`, `prerelease: true`, published
`2026-10-27T18:05:15Z` (2026-10-28 KST).

- Annotated tag object: `2382e082f13d352c07fee3c2e4a76aac2cbf62d4`.
- Peeled source commit: `d56c3ce5c0e008e7f6566308d2c5e6c6308587ec`.
- Source version: `0.5.10-alpha.69`.
- Signing: `velox.exe`, `velox-host.exe` and `velox-setup.exe` are
  Authenticode `NotSigned`.
- Evidence class: `same-repository-public-download`.
- `externalUserAttempt: false`; this is not beta promotion or human adoption.

Main and the annotated tag were pushed atomically. No tag move, replacement
tag producer or reuse of a different source occurred after release.

## Tag Evidence

[Tag CI 40554795110/1](https://github.com/0disoft/velox/actions/runs/40554795110)
passed two byte-identical producer builds with Go 1.26.0 and the checkout-free
consumer job. The producer took 88 seconds and the checkout-free consumer seven
seconds. Verification covered:

- 16 release-manifest entries and 17 SPDX file hashes.
- Provenance source, tag run and ZIP identity.
- PE subsystem 3 for the CLI and 2 for the host and Setup template.
- CLI version `0.5.10-alpha.69` and `vcs.modified=false`.

The tag ZIP is 6,027,217 bytes with SHA-256
`46160a05bd65c82639878b40a7fceb79d9d1847e94121efb0576145e77b69949`.

## Separate Quick Run

The first separate quick run `40554797712` failed in a Windows fixture before
its path comparison used one spelling for `RUNNER~1` and `runneradmin`. Main
`a8f500cee18f5d5ebc844f1daa23c1e9bf112667` changes four lines in
`internal/cli/build_output_test.go` to normalize the fixture root with
`filepath.EvalSymlinks(root)`. The tag-to-main product source diff is empty.

Follow-up [quick run 40555567736/1](https://github.com/0disoft/velox/actions/runs/40555567736)
passed the full Go tests, pin verification, native startup/security,
file-permission recovery, lifecycle, one consumer and summary checks. It built
its own artifact with different VCS metadata, so neither its native checks nor
its bytes are a GUI or artifact claim for the alpha.69 tag ZIP.

## Public Artifacts

All four files were downloaded without authentication. Their byte sizes and
SHA-256 values matched the tag CI evidence and GitHub release metadata.

| Public asset | Bytes | SHA-256 |
| --- | ---: | --- |
| `velox-windows-x64.zip` | 6,027,217 | `46160a05bd65c82639878b40a7fceb79d9d1847e94121efb0576145e77b69949` |
| `checksums.sha256` | 279 | `1a5bce32998bf4cce77960db3f1d2eae46ecb3c3a6c840e8edc2c614a0a12eef` |
| `velox-windows-x64.spdx.json` | 9,618 | `27269a1051af78e56eb7fef7b78a6885477846df930db37c726f24207f647707` |
| `velox-windows-x64.intoto.jsonl` | 744 | `e67126a6f98b2895de4a3c82cba4d7b55b4f0c33a076cf5211c7e7ffb4714179` |

The ZIP, checksum, SPDX and provenance checks passed, including the public CLI
version and PE subsystem checks. The checksum and provenance files describe the
unsigned artifact; they do not supply a Windows publisher identity or an
authenticated attestation.

## Public CLI Smoke

The extracted public CLI ran outside a source checkout in a temporary directory
and invoked only `velox.exe` for:

```text
init
validate
doctor
build (twice)
inspect
```

Both generated application ZIPs had SHA-256
`e95c0d62150c4c4cb87ff6822e5da2be5f111b30cca19d50e66e9fb245a1f3a3`.
The public CLI reported `0.5.10-alpha.69`, and all three executables remained
unsigned.

## Boundaries

No actual GUI editor, native picker or installer was exercised against the tag
or public bytes. No hosted public-verifier dispatch, fuzz, stress, history or
benchmark repeat ran. The native checks in quick run 40555567736 use its own
test-only-main build and are not upgraded to GUI evidence for the public
artifact.

This publication adds no API, database, permission, dependency,
repository-hygiene configuration or runner/job change. Between the tag and
quick-run source commits, the only change is the four-line test-fixture
normalization above. Beta remains held.
