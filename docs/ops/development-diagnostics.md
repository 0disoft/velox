# Development Diagnostics: 2026-10-05

> Follow-up: opt-in `run --debug` metadata diagnostics now ship in the
> published alpha.66 prerelease. The source-only evidence below is historical;
> its local hashes are not the alpha.66 public release bytes. See
> [the alpha.66 publication record](alpha66-publication.md).

## Source-only Scope

Explicit `run --debug` enables fixed `uncaught-error`/`unhandled-rejection`
categories with known startup asset-relative path and bounded line/column
metadata. Unknown sources and Promise rejections output `<unknown>:0:0`.
The listener and private binding install only for debug mode; normal,
watch-only and packaged defaults install neither. The existing trusted-origin
message gate is unchanged, and shutdown drops further diagnostics. IPC v1,
public method declarations and permissions do not gain a diagnostic API.
Public alpha.65 does not include this source extension.

One metadata-only startup inventory holds at most 10,000 asset names; source
paths must be at most 512 bytes and control-free. Only inventory names are
logged. Newly added or renamed files during a watched run remain unknown
until the next host run. Native requests are at most 1,024 bytes, with strict
allowed fields and bounded integer coordinates. The native reporter handles
at most 20 attempts per host run, including invalid requests; the page
listener sends at most 20 per document. Neither adds timers or background work.

## Privacy And Output

No error message, stack, rejection reason, console body, document content or
network response is read. URL credentials, queries and fragments are omitted,
and unknown paths become a fixed marker. Known relative filenames can reveal
application structure; review them before sharing. This metadata is not an
authenticated crash report or a replacement for DevTools. Promise locations
remain unavailable because stacks/rejection contents are not inspected.
The reporter creates no logfile and sends no external telemetry.

JSON stdout remains one run envelope. Explicit `--debug --json` now forwards
host stderr; without debug, JSON still suppresses child stderr. This is a CLI
output exception, not a new manifest field. Callers may redirect stderr to a
local file, which then becomes caller-owned evidence.

## Local Verification

- Scoped devdiagnostic/WebView2/CLI/runner/host Go tests and vet passed.
- Six Bun cases cover source filtering, untouched private event contents,
  coordinates, frames, missing binding, bridge failure and report bounds.
- Native normal/debug JSON modes passed with separate output streams and
  normal close/cleanup exit 0. Default mode had no private binding or
  diagnostics. Debug reported a real uncaught error from `failure.js` and a
  real unhandled rejection, without their synthetic secret bodies or the
  script URL query/fragment. A forged absolute path became unknown; an
  extra `message` field was rejected, not logged.
- A 50-call storm produced 19 lines because one invalid request consumed
  one of the 20 native attempts. Neither stdout nor stderr contained the
  synthetic `SECRET_` markers. Stdout parsed as one successful JSON envelope.

Native receipt: `.cache/development-diagnostics-1791201076564/result.json`.
CLI SHA-256: `d989f8dc3db4291d9f2b93506fc6f4565c2688094d215578d8989ea1ba867795`.
Host SHA-256: `2b6a0214b2c82eee8790273fb7426de7e405b9afb262ac007e74bea16c1f66d3`.
The binaries use the source alpha.65 version string but are not its public
release bytes. Existing user applications and profiles were not changed.

Same-Go 1.27.1, `-buildvcs=false -trimpath -ldflags='-s -w -H windowsgui'`
host size increased from 4,854,784 bytes at the image/font-watch step to
4,861,440 bytes: +6,656 bytes (6.5 KiB). No production startup/idle benchmark,
hosted stress, installer execution, public release or beta promotion was
repeated for this debug-only addition. Dependencies, schemas, database and CI
workflows are unchanged. CommandCode DeepSeek 4.1 Flash/high supplied the
checked documentation draft.
