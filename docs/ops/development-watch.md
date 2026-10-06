# Development Watch: 2026-10-05

> Follow-up: metadata-only image/font watching now ships in the published
> alpha.66 prerelease. The source-only evidence below is historical; its local
> hashes are not the alpha.66 public release bytes. See
> [the alpha.66 publication record](alpha66-publication.md).

Watch is now published in alpha.64. The exact unauthenticated public download
passed native auto-reload/canceled-input preservation/retry/normal close with
debug off; source/run, public hashes and the native receipt are in
[the release record](release.md#alpha64-published-preview-2026-10-05).

Follow-up: the exact alpha.64 local candidate repeated the native watch test
successfully with debug off, canceled-input preservation, subsequent reload
and normal close. Its current hashes and version boundary are recorded in
[the candidate receipt](alpha64-candidate.md). The earlier hashes below remain
the original source-integration evidence.

## Manifest Notices: 2026-10-06

> Follow-up: manifest notices and their JSON-mode stderr path ship in alpha.67.
> The source-CLI/native receipts below are historical, not exact-public-alpha.67
> repetitions. See [alpha67-publication.md](alpha67-publication.md).

The source CLI now observes the selected manifest during `run --watch`, including
custom `--config` paths. It polls content every 500 ms and waits for a 500 ms
quiet period before parsing the captured bytes with the existing manifest
validator. Valid edits print a restart-required notice. Invalid edits and
missing/unreadable files print a nonfatal error; repeated unchanged errors are
suppressed. Links/reparse points and reads exceeding 1 MiB are refused.
Same-size/time content edits are detected; reverting a pending edit cancels it.

No manifest edit triggers page reload, host restart or runtime reconfiguration.
The original asset watcher and permissions continue unchanged until the user
restarts. The CLI serializes host and manifest stderr writes and cancels/joins
its manifest loop on host exit, including failure. CLI manifest notices/errors
also reach stderr in JSON mode; host stderr remains suppressed unless debug
is enabled. Normal runs and packaged apps have no additional
manifest loop; no host source, IPC, DB/schema, dependency or CI change is made.
This extension is not in published alpha.66 and includes no version bump.

Native follow-up used the source CLI from `5b7dda3` built once with Go 1.27.1,
`-buildvcs=false -trimpath -ldflags='-s -w'`, alongside the unchanged public
alpha.66 host and its matching metadata. The isolated File Notes project and
private profile passed with `--watch`, without `--debug` or `--json`, so normal
stderr notices were visible. The test used CDP through a loopback-only port;
no test-side reload/navigation or cache override was issued.

The three manifest phases were a valid app name/window width/permission edit,
invalid JSON, and corrected valid settings. Exactly two restart notices and
one error appeared, with no repeated diagnostics during each observation
window. Every phase retained the same document marker, exact unsaved text
`manifest watch unsaved text`, dirty status, origin, and runtime-config digest
`442ffd5e757427eeb0e2f510dbadcec2e5ee2db9eb322637dcc92b454a579a7f`.
No browser dialog opened. Clearing only the isolated editor for normal close
returned CLI and cleanup exit 0; the temporary runtime file was removed.
This is automated native/CDP evidence, not a new manual confirmation.

The first attempt attached before editor initialization and failed its input
baseline; it cleaned up with exit 0. Adding an explicit readiness wait fixed
the fixture without changing production code.
Failed receipt: `.cache/manifest-watch-1791277219369/result.json`.
Passed receipt: `.cache/manifest-watch-1791277257382/result.json`.
Local source CLI SHA-256:
`c43b15fe70c37f20d04e5bccc1daa12d8b7fb942a3d118b94b36ebe926e2b54e`.
Unchanged public host SHA-256:
`7cf4c80d614ff1ba4f942c39dbfd42865b59146d5f5c089b4c5fa0be3c850beb`.
The CLI is not a public alpha.66 release artifact even though its version
string remains alpha.66. No installer, hosted stress or performance/size
benchmark was repeated for this verification-only follow-up.

To repeat, place the source CLI, compatible host and matching `velox-host.json`
in `.cache/manifest-watch-bin`, or select the directory with
`VELOX_MANIFEST_WATCH_BIN_DIR`, then run:

```sh
bun run scripts/manifest-watch-smoke.ts
```

The script owns only its copied fixture, private profile and child process,
and force-cleans the child process tree if normal close fails. Results and
logs remain under its timestamped `.cache/manifest-watch-*` directory.

## JSON Output Follow-Up: 2026-10-06

The source runner separates CLI manifest-watch output from child stderr.
`run --watch --json` emits restart notices and nonfatal manifest errors to
stderr with debug off; stdout remains one result envelope. Ordinary host
stderr is still discarded unless `--debug` is enabled. The two output paths
share a lock when directed to the same writer, and watcher cancellation/join
still completes before the result is emitted. Host options and runtime config
are unchanged. This is source-only, not a public alpha.66 release change.

Focused runner/CLI tests and vet passed. CLI tests cover debug off/on, invalid
manifest recovery, exact JSON parsing, suppressed child stdout and host
diagnostics visible only with debug. Existing normal JSON cases remain covered.

The updated native smoke passed with the unchanged public alpha.66 host:

```sh
bun run scripts/manifest-watch-smoke.ts --json
```

Valid manifest edits, invalid JSON and corrected settings retained the same
document marker, exact unsaved editor text, origin and runtime-config digest.
Exactly two restart notices and one error reached stderr. Stdout parsed as
one successful `run` envelope, no browser dialog opened, and normal close and
cleanup both returned 0 with the temporary config removed. Debug was off.
This native run did not deliberately generate a host diagnostic; host-log
suppression is proven by the CLI launcher tests, not upgraded from that run.

Receipt: `.cache/manifest-watch-1791280485300/result.json`.
Local source CLI SHA-256:
`254e23959defc1312a1681b10e16b56766e0841fe8981b1b0467b407537b908e`.
The public host remains
`7cf4c80d614ff1ba4f942c39dbfd42865b59146d5f5c089b4c5fa0be3c850beb`.
The earlier non-JSON receipt and hashes above are preserved as a separate
source snapshot. No host/public IPC, DB/schema, dependency, CI or version
change is included. No release/installer, hosted stress or performance/size
benchmark was repeated for this CLI-only output change.

## Image/Font Source Extension: 2026-10-05

The source checkout now includes common image/font metadata in the existing
500 ms polling and 500 ms quiet-period detector. Images PNG/APNG/JPG/JPEG/
GIF/WebP/AVIF/BMP/ICO/SVG and fonts WOFF/WOFF2/TTF/OTF/EOT are matched
case-insensitively. The existing safe tree traversal contributes only path,
size and modification time for those assets, with no binary content read,
new cache or dependency. The 64 MiB content budget still applies only to the
entry and supported text files. Additions, deletions and renames count.
Reverting text contents cancels a pending notification; binary edits that
preserve size and modification time are intentionally not detected. Manifest
edits and other formats remain excluded. No watcher starts by default.
This extension is not in published alpha.65; the earlier release receipts
above and original scope below remain historical.

Scoped devwatch/runner/CLI/WebView2 tests and vet passed, including every
extension, uppercase forms, same-size changed-time writes, debounce,
addition/rename/removal, a sparse font larger than 64 MiB, same-size/time
limitations and symlink rejection without skips. Native/CDP verification
used a matching local source CLI/host, copied File Notes assets and a private
profile. Without changing HTML/CSS/JS, replacing the SVG changed decoded
canvas pixels from `[255,0,0,255]` to `[0,255,0,255]`; replacing the font at
the same URL from Noto Sans KR to local Windows Arial changed measured text
width from `203.23989868164062` to `204.51171875` with a loaded FontFace.
Each operation produced a new document and retained the HTTPS origin.
Existing text reload, real beforeunload cancellation preserving input,
subsequent retry and normal close/cleanup also passed with exit 0. Debug was
off; no test-side navigation or cache override was used.

The first visual fixture used inline CSS/JS and failed its baseline under
the existing CSP. It cleaned up with exit 0; changing the fixture to local
external files passed without changing CSP or runtime behavior.
Failed receipt: `.cache/normal-reload-1791199787277/result.json`.
Passed receipt: `.cache/normal-reload-1791199834636/result.json`.
Local CLI SHA-256: `1e45ae85492ae725b2aecb44625c233c71705d687af1d56079d12eff77751602`.
Local host SHA-256: `1d62f8f2dc8f5a5888bfa9e1cb5a5cb2666c56ed0d2282e61b7dc0d3df3f0fc7`.
These source-build bytes report alpha.65 but are not its public release bytes.

Compared with an archive of source `bd0ad1e` using Go 1.27.1 and identical
`-buildvcs=false -trimpath -ldflags='-s -w -H windowsgui'` options, the host
grew from 4,853,760 to 4,854,784 bytes: +1,024 bytes. This is a size check,
not a production performance benchmark. No hosted release/stress, installer
execution, dependency/IPC/DB/schema/CI/version change, public publication or
beta promotion was performed. Existing user apps and profiles were untouched.
CommandCode DeepSeek 4.1 Flash/high supplied the checked scope draft.

## Original Text-only Scope

`velox run --watch` samples entry/HTML/HTM/CSS/JS/MJS content every
500 ms and emits once after a 500 ms quiet period. It uses bounded content
hashing, shared safe asset traversal and no external dependency. The host
owns the loop only during an explicit watched run and joins it on exit.
Normal runs and generated runtime configurations do not enable watch.
Full-page reload uses browser navigation, not window destruction or HMR;
application `beforeunload` can cancel, and a subsequent edit can retry.
The debug flag remains independent. Only development cache bypass is enabled.

## Local Verification

- Detector tests passed: same-size/same-time content edits, burst coalescing,
  deletion, reverted edits, ignored fonts, missing entry recovery, linked
  assets and cancellation. Symlink rejection ran without a skip on Windows.
- Runner/CLI tests cover default-off, debug-only, watch-only and both flags,
  unchanged temporary configurations and child output/exit handling.
- Runtime tests cover opt-in dispatch, coalescing, no forced destruction and
  dropping queued reload after watch stops.
- The native smoke used the matching local candidate CLI/host and a copied
  File Notes fixture with a private profile. No existing app was altered.
  Watch ran without `--debug` or a test-side `Page.reload`/cache override.
- Two consecutive HTML/CSS/JS edits were observed in the rendered DOM with
  unchanged virtual HTTPS origin. Trusted input then made File Notes dirty:
  an automatic reload opened `beforeunload`, cancellation preserved the
  exact input, and a subsequent clean edit successfully auto-reloaded.
- Normal page close returned CLI exit 0 and cleanup status 0. This is an
  automated native/CDP result, not a new maintainer/manual confirmation.

Native receipt: `.cache/normal-reload-1791186251684/result.json`.
CLI SHA-256: `1f3d745e56d00f8f3ca862dc4cbb78db7c8efd9c1ee694df201301fb1617f842`.
Host SHA-256: `25e6ccf4417a299501fcf240d5dfb559f23ce27059d0939ebec947aa69976c48`.
These hashes identify the integration-test binaries before the subsequent
stderr-only watch-warning wording adjustment. No executable behavior changed
after the native run beyond that diagnostic wording.

The Go 1.27.1 GUI host measured 4,853,760 bytes versus the same-toolchain,
same-flags alpha.63 candidate's 4,736,000 bytes: +117,760 bytes (115 KiB,
about 2.5%). Host sources in that baseline are unchanged through the detector
commit `4cdfef6`. No production idle CPU or startup performance comparison
was performed; production allocates no watch loop/ticker by default.

No new IPC method/permission, DB/config schema, dependency, CI workflow,
version bump, installer, public release or beta promotion is included.
The public alpha.63 artifacts and previous File Notes package remain intact.
