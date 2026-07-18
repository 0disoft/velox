# SignPath Foundation Onboarding

- Status: Application packet ready; maintainer submission required
- Owner: Project maintainer
- Repository: https://github.com/0disoft/velox

## Application Facts

Use these values in the SignPath Foundation application.

| Field | Value |
| --- | --- |
| Project | Velox |
| Repository | `https://github.com/0disoft/velox` |
| Maintainer | `0disoft` |
| License | `MIT OR Apache-2.0` |
| Primary language | Go |
| Target | Windows x64 |
| Build system | GitHub Actions on GitHub-hosted `windows-2025` runners |
| Distribution | Portable ZIP containing a Go CLI, generic Go WebView2 host, schemas, and notices |
| Native signing subjects | `velox.exe`, `velox-host.exe` |
| Telemetry | None in the CLI or runtime host |
| Security policy | `SECURITY.md` |
| Privacy policy | `PRIVACY.md` |
| Private vulnerability reporting | Enabled for `0disoft/velox`, verified 2026-07-18 |

The public repository description and discovery topics are also populated.
These repository-owner setup items are complete and do not need to be repeated
during the SignPath application.

## Application Description

Paste this description where the application asks about the project:

> Velox is an open-source, compile-free Windows desktop application packager
> for static HTML, CSS, and JavaScript. It combines a project manifest and
> static assets with a prebuilt pure-Go WebView2 host. Consumer builds do not
> invoke Go, Rust, C++, Node.js, a package manager, or a GitHub Actions cache.
> The public repository contains deterministic build and benchmark evidence,
> an explicit security model, and a narrow native API. We request Authenticode
> signing for exactly two Windows x64 executables, velox.exe and
> velox-host.exe, produced by GitHub-hosted release jobs.

## Requested SignPath Values

Ask SignPath to use or allow these stable slugs:

| SignPath value | Requested value |
| --- | --- |
| Project slug | `velox` |
| Artifact configuration slug | `windows-x64` |
| Signing policy slug | `release-signing` |

The organization ID and certificate publisher subject are assigned or approved
by SignPath. Do not guess them. Record the exact values after acceptance.

Upload or paste `.signpath/artifact-configuration.xml` as the artifact
configuration. Link the GitHub trusted build system and use
`.signpath/policies/velox/release-signing.yml` as the repository source policy.
The provider input is `velox-signing-input.zip`, which contains exactly the two
PE files at the ZIP root.

## Maintainer-Only Steps

These steps require the repository owner's authenticated account and acceptance
of provider terms. An automation agent must not perform them on the maintainer's
behalf.

1. Confirm that you own or can license all project code under
   `MIT OR Apache-2.0`. This is a maintainer legal assertion, not an automated
   source scan or legal opinion.
2. Open https://signpath.org/apply and submit the application facts and
   description above.
3. Review and accept the SignPath Foundation terms as the project maintainer.
4. After approval, install the SignPath GitHub App with access limited to
   `0disoft/velox` and link the predefined `GitHub.com` trusted build system.
5. Create or confirm the `velox`, `windows-x64`, and `release-signing` provider
   objects using the repository-owned configuration files.
6. Confirm the exact Authenticode publisher subject and timestamp policy with
   SignPath.
7. Create a protected GitHub environment named `alpha-signing`, restrict it to
   release tags, and require maintainer approval.
8. Add `SIGNPATH_API_TOKEN` only as an `alpha-signing` environment secret. Do
   not paste the token into an issue, chat, file, or command output.

## Values to Return

After approval, provide only these non-secret values to the implementation
work:

```text
SignPath organization ID:
Project slug: velox
Artifact configuration slug: windows-x64
Signing policy slug: release-signing
Exact publisher subject:
Timestamp policy or authority requirement:
GitHub environment created: yes/no
SIGNPATH_API_TOKEN environment secret created: yes/no
SignPath GitHub App limited to 0disoft/velox: yes/no
```

Never return the API token value. Once the non-secret values and confirmations
exist, the repository can add the active signing workflow, live-check and pin
all action commits, exercise the real provider output, and build the
release-mode signing record.

## Official References

- https://signpath.org/apply
- https://signpath.org/terms.html
- https://docs.signpath.io/trusted-build-systems/github
- https://docs.signpath.io/artifact-configuration/examples
