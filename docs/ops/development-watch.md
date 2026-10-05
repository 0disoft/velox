# Development Watch: 2026-10-05

Watch is now published in alpha.64. The exact unauthenticated public download
passed native auto-reload/canceled-input preservation/retry/normal close with
debug off; source/run, public hashes and the native receipt are in
[the release record](release.md#alpha64-published-preview-2026-10-05).

Follow-up: the exact alpha.64 local candidate repeated the native watch test
successfully with debug off, canceled-input preservation, subsequent reload
and normal close. Its current hashes and version boundary are recorded in
[the candidate receipt](alpha64-candidate.md). The earlier hashes below remain
the original source-integration evidence.

## Scope

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
