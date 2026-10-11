# File Notes Native Readiness

## Optional Packaged-App Lifecycle Check

`TestPackagedAppLifecycle` inspects the supplied portable directory before
launching its actual application executable and runtime configuration. Set
`VELOX_PACKAGED_APP_DIRECTORY` to that directory and optionally set
`VELOX_PACKAGED_APP_LIFECYCLE_RESULT` to a JSON receipt path whose parent exists.
Run:

```text
go test ./tests/startup -run '^TestPackagedAppLifecycle$' -count=1 -v -timeout=90s
```

The environment settings are process-local; do not set them globally. With no
directory, this optional native test skips and starts no extra host in ordinary
CI. The app must implement the benchmark-only `dom-2raf` readiness marker,
as File Notes and the generated starters do. This is not a general readiness
claim for arbitrary consumer JavaScript.

Both launches use a newly generated disposable profile under `.cache/profiles`,
never the installed app's default profile. The test uses the unchanged
browser-exit-first helper and its shared ten-second release budget. It also
retains the one-second host-exit and ten-second immediate-readiness guardrails;
failure receipts must not claim success when either guardrail fails.

## Local Packaged File Notes: 2026-10-12

The final File Notes portable app built from source `9443812` passed directory
inspection and a fresh/immediate same-profile native pair. Exact identities:

- Host SHA-256: `2081fe6a393544e49fc8f45b09390104d53db658ebcb0c3faec6fe6d5abadac8`.
- Assets SHA-256: `5512d6af472a3c0ea4d721c50a84108fb1ea9b756bd45aae726101cf4830650e`.
- ZIP SHA-256: `8f0d9448a8a322ab8ac0c69c6af4810c39e2d900c908185bba9daa6412454364`.
- Release string: `0.5.10-alpha.68`; this is a local candidate, not a public
  replacement for the published alpha.68 asset.
- WebView2: `154.0.4258.62`, Windows x64.

First readiness was 959.26 ms and host exit 91.88 ms. Immediate readiness was
881.47 ms and host exit 101.57 ms. Its main browser exited 6,528.98 ms after
host exit; the disposable profile was removed at 6,650.18 ms after host exit,
following both observed browser exits. The native test completed in 8.93 s.
Receipt: `.cache/readiness-20261012-source-e55883b/file-notes-native-lifecycle.json`.

`TestPackagedLifecycleGuardrails` also passed five failure-path cases and
preserved an existing browser-exit failure. The measured native pair predates
that helper extraction but satisfies its unchanged numerical guardrails.

These are actual packaged-app launches and browser/process observations in
benchmark mode, not browser mock results. They do not verify a manual native
save picker, close-confirmation cancellation, installer upgrade or recovery
after forced termination. The historical intermittent ten-second failure is
still unresolved; one passing pair does not establish its cause or fix.
No user profile, document, registry entry, public API, runtime host, permission,
database schema, dependency, version or CI workflow changed.
