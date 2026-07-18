# SignPath Configuration

These files are dormant repository copies of a proposed future SignPath
Foundation configuration. ADR 0011 defers provider onboarding until an actual
developer-preview adoption trigger exists.

- Project slug: `actutum`
- Artifact configuration slug: `windows-x64`
- Signing policy slug: `release-signing`
- Artifact configuration: `artifact-configuration.xml`
- GitHub source policy: `policies/actutum/release-signing.yml`

The artifact configuration accepts one ZIP containing exactly
`actutum-host.exe` and `actutum.exe` and applies Authenticode signing to both. The
source policy requires GitHub-hosted runners and rejects workflow reruns.

The provider organization ID, accepted project identity, certificate publisher
subject, API token, and protected GitHub environment are external values. Never
commit those credentials or copy provider responses into this directory.

This directory does not authorize provider submission, a signing workflow, or
a release. Follow
`docs/ops/signpath-onboarding.md` and `docs/ops/signing.md`.
