# Alpha.68 Local Preparation: 2026-10-07

## Identity

- Source version: `0.5.10-alpha.68` (local candidate; not published).
- Public preview: `0.5.10-alpha.67` (published unsigned prerelease).
- This step permits local tests and a candidate build only.
- No push, tag, hosted job, signing or publication is included.
- Candidate root: `dist/candidates/alpha68-local/`.

## Scope

- `templates` lists the four existing starters.
- Expanded CLI/init help and shell-quoted next-step run/build hints.
- Web-only text-editor literal find bar with match-case and wrap navigation.
- Empty-query Escape/X close fix, including IME handling.

Existing generated apps are unchanged. No host/IPC API, permission,
DB/storage schema, dependency or CI workflow change is included.
Product specification and command-contract records retain the distinction
between these source additions and published alpha.67.

## Verification Boundaries

Buildplan, builder, CLI, inspector, runner, releasebundle, releaseevidence and
hygiene Go tests, scoped `go vet` and `git diff --check` passed for the version
change.

Artifact identity and local verification results will follow the version
commit. Local Go 1.27.1 bytes are not a same-toolchain comparison with the
public alpha.67 Go 1.26.0 bundle.

Historical manual find-close confirmation and earlier native tray, watch,
picker and draft receipts are retained, not relabeled as alpha.68 public-byte
evidence. Installer execution, new manual native interaction, hosted stress,
performance comparison and beta promotion are outside this local step.
