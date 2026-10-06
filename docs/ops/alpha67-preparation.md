# Alpha.67 Local Preparation: 2026-10-06

## Identity

- Source version: `0.5.10-alpha.67` (local candidate; not published).
- Public preview: `0.5.10-alpha.66` (published unsigned prerelease).
- No alpha.67 tag, push, hosted run or publication is authorized in this step.

## Scope

The next preview groups the CLI manifest-change notices for `run --watch`,
their stderr path alongside one JSON stdout envelope, and the new
`init --template tray-app` starter. Manifest edits require a restart; they
do not reload, restart or reconfigure the running host. Normal packaged
apps do not acquire a manifest watcher. The tray starter grants only
`notification.show` and reuses existing native behavior.

Implementation and manual evidence are retained in
[development-watch.md](development-watch.md) and
[tray-app-starter.md](tray-app-starter.md). The prior
[source candidate](source-candidate-9dfcb8b.md) retains its alpha.66 version
string and hashes; those artifacts are not alpha.67 bytes.

## Validation Boundary

Version-dependent buildplan, builder, CLI, inspector and runner tests passed,
along with releasebundle, releaseevidence and repository hygiene tests.
`git diff --check` passed.

This initial record describes version preparation, not a completed build.
The matching alpha.67 CLI, GUI host and GUI Setup will be compiled together;
their artifact identity and local packaging results will be recorded after
the build. Existing public alpha.66 assets and historical receipts stay intact.
No API/IPC, DB/schema, dependency or CI workflow change is included.
Signing, installation, hosted stress, performance comparison and publication
are outside this local preparation.
