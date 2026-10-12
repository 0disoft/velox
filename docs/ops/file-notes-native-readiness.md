# File Notes Native Readiness

## Packaged Recovery After Shutdown Corrections: 2026-10-12

Source `23f2327` was rebuilt into a new private CLI/GUI host bundle and used to
package the unchanged File Notes example. Directory inspection passed before
and after the recovery check. Host SHA-256:
`c81f03cd06cc26bc9a792bd1c48bb11f92459c2884f424ecba624691c3785a2c`;
assets SHA-256:
`8efbce17c90ea30e0fa94b3dfca489b387f918e50047082aa0ca8f7dd1386b74`.
The local app ZIP SHA-256 is
`a8fd5b377b59790d4431886224adc23bb9ee63ad831e36522985086d87837c1b`.
Its release string remains alpha.68; this is not a public asset replacement.

`scripts/file-notes-packaged-recovery.mjs` drives the actual packaged application
over loopback CDP without replacing its files, bridge, storage or dialogs. It
creates a unique repository-local profile and isolated LocalAppData, enters
synthetic Korean/multiline/literal text, waits for the actual draft transaction,
and reads it through File Notes' own storage module. Native WM_CLOSE produces
the real beforeunload dialog: cancel retains the text and live host; accepting
the repeated close exits normally. Immediate same-profile relaunch starts
before the first browser-exit wait and restores identical committed text,
the dirty state and the recovery notice. The reopened app also closes normally.
Browser process handles acquired while each observed CDP browser PID was live
confirm both browser exits. The fixture and receipt are retained, not deleted.

Run explicitly on Windows with Node/Playwright and PowerShell 7 available:

```text
node scripts/file-notes-packaged-recovery.mjs <velox.exe> <File-Notes-portable-directory>
```

Two actual runs passed in 11.18 s and 10.52 s on WebView2 `154.0.4258.62`.
The final receipt is
`.cache/packaged-recovery-5a4ef7bf-4bd0-4912-aca8-240331e5d50f/result.json`;
the first is
`.cache/packaged-recovery-74c9e075-ac31-4a26-8e2f-45b4c39c3a8b/result.json`.
CDP arguments and process observation instrument this check, so it makes no
ordinary-startup latency claim. It does not cover a native file picker, an
installer upgrade, interrupted transaction or power-loss recovery. Existing
user profiles and documents were not accessed.

The separate benchmark-mode `TestPackagedAppLifecycle` passed on the same
artifact in 14.85 s: first/immediate ready 958.22/7,091.53 ms, host exits
71.91/103.46 ms, immediate browser exit 6,419.49 ms after host, and profile
release 6,501.33 ms after host. Receipt:
`.cache/shutdown-package-20261012/lifecycle.json`. This confirms the existing
guardrails, not a relaunch-delay fix. No new default native CI launches, API,
permissions, database schema, dependency, version, push or publication occurred.

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

## Long Profile Storage: 2026-10-12

A real WebView2 storage probe reproduced `UnknownError: Internal error opening
backing store for indexedDB.open.` with a 146-character profile and the prior
host. The underlying IndexedDB manifest name was 261 characters long. Ordinary
DOM readiness does not cover this failure: the new storage probe reports ready
only after its database transaction completes, followed by two animation frames.

The runtime now gives WebView2 an extended-length path. When an existing local
directory has a shorter Windows short-name alias, the runtime verifies it with
`os.SameFile` before using that spelling. It never creates an alias, relocates
the profile, resets its contents or changes registry settings. The configured
path still identifies single-instance locks and window-state files. UNC and
device namespaces are not queried for short-name aliases.

Extended spelling alone passed write/read at 146 characters, but still failed
at 180 and 218 characters. Those exploratory failures are retained, not counted
as passes. With the verified short-name spelling, the final source-built host
passed four separate write/exit/browser-exit/relaunch/read/browser-exit pairs
with profile lengths 118, 146, 180 and 218. The long-name IndexedDB manifests
were 261, 295 and 333 characters. The fixture checked exact synthetic Korean
text and CRLF data; it used no native file permission or user document. All
successful profiles were removed only after both browser exits were observed.

A fifth compatibility pair wrote through the previous host's ordinary profile
spelling and read through the new host's spelling. It passed without copying or
migrating the database. The four final pairs took 56.37 s; compatibility took
14.02 s. This is controlled Windows x64 / WebView2 `154.0.4258.62` evidence,
not a cross-machine or arbitrary-path-length guarantee.

Native test invocation (process-local environment):

```text
VELOX_STORAGE_PATH_TEST=1
VELOX_BUILT_HOST=<freshly built host.exe>
go test ./tests/startup -run '^TestBuiltHostIndexedDBLongPath$' -count=1 -v -timeout=120s
```

Optionally set `VELOX_STORAGE_LEGACY_HOST` to the previous host and run the
`/legacy-profile` subtest to check retained data across the change. Without the
enable flag, this optional native check skips and consumes no ordinary CI host
launches. Test failures preserve their disposable profiles for diagnosis.

The before/after hosts were built with Go `1.27.1`, `-trimpath`, and
`-ldflags='-s -w -H windowsgui'`. Sizes: 4,876,288 / 4,878,336 bytes (+2 KiB).
SHA-256 before: `0b2ac4f2ee1aad7add3bd1a77a611cd641fdf0d3237d4cb92f64a7d9a33d7241`;
after: `9d6497f556089668242fb6cf69110ff7c630e5643f573177369e7fa2d55e8bc8`.
Related WebView2/host/startup package tests, vet and diff checks passed. The first
probe's invalid asset-root configuration was repaired before storage evidence
was collected; its failure does not establish an IndexedDB defect.

Short aliases are not guaranteed on NTFS and are absent on ReFS. If unavailable,
the runtime retains extended spelling rather than substituting a different
profile. Deeper profiles in that environment remain unverified and the
extended-only failures above remain a limitation. See Microsoft's
[short-path API limitations](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getshortpathnamew)
and [extended-path rules](https://learn.microsoft.com/en-us/windows/win32/fileio/maximum-file-path-limitation).
No public API, permissions, database schema, dependency, version, release or CI
workflow changed. The runtime host implementation changed; no installed app
or user profile was replaced. Installer upgrade and interactive picker checks
were not run in this slice.

## Isolated Interactive Session

`node scripts/packaged-app-session.mjs <portable-directory>` verifies the host's
build-report size/hash, strips inherited benchmark/browser argument overrides,
and launches exactly that packaged executable with a new disposable profile.
Default profiles use a short temporary path; `--long-profile` uses a deeper
repository-local path, and `--extended-profile` uses its extended spelling.
These are alternative fixture modes, not an instruction to alter a user profile.

The launcher preserves the profile and writes an exit receipt under its unique
`.cache/packaged-session-*` directory. Nonzero exit, spawn error or its own
eight-minute forced stop causes a failing exit status. It does not claim browser
exit, durable draft storage or permission success from a normal host exit alone.
It is a local interactive tool, not part of ordinary CI or a consumer bundle.

Earlier actual packaged File Notes session `a286cbed-f6ff-480c-ac9f-0946f9a9007e`
confirmed the close warning, cancellation retaining the synthetic text, Undo to
a clean document and normal host exit. Its draft store failed, so it is not
draft-recovery evidence. Session `b058ea3a-cc62-40c3-bd7e-3b04284fa347` showed
successful draft persistence with extended spelling, but ended nonzero; it is
not a successful lifecycle pair. The native transaction tests above are the
new storage-relaunch proof. Both historical sessions predate the final launcher
and runtime changes; syntax/invalid-input checks alone do not verify a new
interactive run. The launcher's short-profile branch now also creates its
receipt directory before launching rather than failing at receipt write time.
