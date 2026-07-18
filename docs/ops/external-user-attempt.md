# External User Attempt

- Status: Contract ready; no qualifying attempt recorded
- Owner: Project maintainer
- Decision: ADR 0012

## Purpose

M4 needs evidence from a person, account, or repository that is not controlled
by the implementation workflow. The repository-owned public-download workflow
proves GitHub Release acquisition and execution, but it always records
`externalUserAttempt: false` and cannot satisfy this gate by itself.

## Qualifying Attempt

A qualifying attempt:

- starts from the public `v0.6.0-alpha.1` GitHub Release;
- records the exact ZIP SHA-256 before execution;
- follows the public installation and hello-project path without unpublished
  maintainer instructions;
- records Windows and WebView2 versions;
- reports whether SmartScreen warned or managed-device policy blocked launch;
- runs `version`, `init`, `doctor`, `build`, `inspect`, and `run` far enough to
  distinguish acquisition, compatibility, packaging, and startup failures;
- states whether a compiler, Node.js, or package-manager command was required;
- links public evidence when available without exposing tokens, private paths,
  usernames, organization details, or application data.

An attempt can fail and still qualify. Product friction is evidence. A
maintainer repeating the same workflow from another branch does not qualify.

## Preferred Submission

Use the repository's **External user attempt** issue form. Report only:

```text
Release tag:
Release ZIP SHA-256:
Public workflow or reproduction URL, if any:
Windows version:
WebView2 version:
SmartScreen or policy result:
Last successful command:
First failing command and safe error code:
Compiler or package-manager command required: yes/no
Overall outcome: success/failure/blocked
```

Do not paste local absolute paths, environment variables, access tokens,
private repository URLs, crash dumps containing user data, or proprietary test
assets.

## Maintainer Review

Before counting an attempt toward M4, verify:

1. The tag and digest identify an immutable public Actutum release.
2. The reporter is independent from the release workflow.
3. The report covers acquisition through either startup success or a localized
   failure.
4. Any claimed compiler-free path is supported by command or workflow evidence,
   not only by the reporter's recollection.
5. The report contains no sensitive data that needs removal.

Record the accepted issue or evidence URL in `docs/product/01-roadmap.md`.
Do not copy the reporter's machine logs into the repository.

## Non-Qualifying Evidence

- Same-run GitHub Actions artifacts.
- The repository-owned checkout-free consumer job.
- The repository-owned public-download workflow by itself.
- A local maintainer build from `main`.
- A screenshot without tag, digest, environment, and command boundary.
- A report that used unpublished patches or credentials.
