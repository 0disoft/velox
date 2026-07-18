# Security Policy

## Supported Versions

Actutum has no public release yet. Development on `main` receives security fixes,
but no published version currently carries a support or response-time promise.

## Reporting a Vulnerability

Use GitHub private vulnerability reporting:

https://github.com/0disoft/actutum/security/advisories/new

Do not open a public issue containing an exploit, credential, private path,
provider token, signing request, or sensitive application data. If the private
reporting form is unavailable, contact the maintainer through the GitHub profile
without including vulnerability details and request a private transfer channel.

Include the affected commit or artifact digest, operating-system and WebView2
versions, reproduction boundary, expected impact, and whether public disclosure
has already occurred. Reports are handled on a best-effort basis until the
first public alpha establishes a formal support policy.

## Release Security Boundary

Signing credentials, certificate private material, raw OIDC tokens, unreleased
provider responses, and embargoed vulnerability details must never be committed,
uploaded as ordinary workflow artifacts, cached, or pasted into public issues.
See `docs/ops/signing.md` for the release trust boundary.
