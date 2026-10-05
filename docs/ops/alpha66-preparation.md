# Alpha.66 Local Preparation: 2026-10-05

## Identity

This is the historical local preparation record for version
`0.5.10-alpha.66`, written before the tag push. At preparation time it was
not published: there was no alpha.66 tag, release, hosted run, or publication,
and no matching alpha.66 distribution bundle, digests, or hosted evidence
existed yet. The tag and hosted evidence were added afterward; see
[the alpha.66 tag evidence](alpha66-tag-evidence.md). The current public
preview remains the unsigned prerelease `v0.5.10-alpha.65`, and alpha.66 is
still not published.

- Source version: `0.5.10-alpha.66` (not published).
- Public preview: `0.5.10-alpha.65` (published unsigned prerelease).
- At preparation time, no alpha.66 artifacts existed: no release bundle,
  checksums, SBOM, provenance, public download, or hosted evidence.
- Only the version string changed, so a locally compiled CLI reports
  `0.5.10-alpha.66`.

## Prepared Release Scope

The prepared scope is the feature slices implemented after
the alpha.65 publication. Existing generated projects are not upgraded.

- Image and font metadata in `run --watch`: common image and font assets are
  watched by path, size and modification time inside the existing 500 ms
  polling and 500 ms quiet-period detector, without reading contents. Images
  PNG/APNG/JPG/JPEG/GIF/WebP/AVIF/BMP/ICO/SVG and fonts WOFF/WOFF2/TTF/OTF/EOT
  are matched case-insensitively. Same-size edits that preserve the
  modification time are not detected; that is the metadata-only limitation.
- Opt-in `run --debug` JavaScript diagnostics: `run --debug` installs a
  metadata-only listener for `uncaught-error` and `unhandled-rejection`.
  Unknown sources and Promise rejections report `<unknown>:0:0`, and Promise
  locations remain unavailable because stacks and rejection contents are not
  inspected. No error message, stack, rejection reason, console body, document
  content, or network response is read and no raw content is logged. Normal,
  watch-only and packaged defaults install neither the binding nor the
  listener.
- Local IndexedDB draft recovery for `init --template text-editor`: new
  projects store one `drafts/current` record with `schemaVersion`, `name`,
  `text` and `updatedAt`. Text must fit both the 2 MiB character bound and the
  2 MiB UTF-8 byte bound. Typing pauses trigger a 300 ms debounce, and writes
  and clears share one serialized chain. Restore keeps the document dirty with
  no native target, so the first Save chooses a fresh destination. Native
  paths, save tokens, permissions and saved-text baselines are never stored.
  Drafts are local application data, not encrypted backups or remote sync, so
  abrupt exit can lose edits still in the debounce or a pending transaction.

Implementation native and local test details for each slice are recorded in
[development-watch.md](development-watch.md),
[development-diagnostics.md](development-diagnostics.md) and
[text-editor-drafts.md](text-editor-drafts.md). The prior local source
candidate `11efd69` ([source-candidate-11efd69.md](source-candidate-11efd69.md))
carries the same feature commits but uses the `0.5.10-alpha.65` version string,
so its bytes are not alpha.66 artifacts and its record is historical.

## Validation

Locally run results on the shared checkout at the prepared version:

- Version-dependent package tests passed: `buildplan`, `builder`, `cli`,
  `inspector`, `runner`, `hostmeta`,
  `releasebundle`, `releaseevidence`.
- `buildinfo` was checked by the same command but has no test files.
- `go run ./cmd/velox version --json` reported `0.5.10-alpha.66`.
- Scoped vet passed: `go vet ./internal/buildinfo ./internal/buildplan
  ./internal/builder ./internal/cli ./internal/inspector ./internal/runner
  ./internal/hostmeta ./internal/releasebundle ./internal/releaseevidence
  ./tests/hygiene`.
- `go test ./tests/hygiene` passed, retaining the public alpha.65 evidence
  checks separately from the unpublished alpha.66 source checks.
- The publishing script parsed successfully and its release-notes here-string
  contained no control escapes. Workflow triggers, jobs, commands and permissions
  are unchanged; only generated release prose changes.

At preparation time, no alpha.66 distribution bundle, digest, hosted run, or
publication was asserted. The prepared version was exercised only through a
locally compiled CLI, not a shipped alpha.66 distribution. Hosted tag evidence
was added later and is recorded separately.

## Release Boundary

The unsigned-preview caveats are unchanged for any future alpha.66 publication:

- Compatibility is limited to Windows x64: Windows 10 version 1709 build
  16299 or newer clients, Windows Server 2016 build 14393 or newer servers, and
  Evergreen WebView2 Runtime 92.0.902.49 or newer. The executables are not
  Authenticode-signed, so SmartScreen may warn or managed devices may block
  execution; verify `checksums.sha256` before running any archive.
- Immediate same-profile relaunch can take several seconds while the previous
  WebView2 browser process exits. Close an existing app with the same
  identity/profile before a development run.
- Packaged applications keep HTML, CSS, JavaScript and `velox.runtime.json`
  beside the unchanged generic host, so anyone who can modify the application
  directory can change application behavior. This is the accepted mutable
  asset boundary; there are no sealed assets or local tamper resistance.
- The checksum file, SPDX SBOM and provenance statement describe the release
  files. They do not provide a Windows publisher identity, and the provenance
  statement is not an authenticated attestation.
- Optional `build --installer` produces a per-user Windows Setup for one
  account, and a build can stage an optional application icon and version
  resource into its host copy; without branding the generic prebuilt host keeps
  its own icon, version resources and bytes. There is still no automatic
  updater, machine-wide or elevation-requiring install, MSI/MSIX package,
  Authenticode code signing, arbitrary application backend or plugins, or
  non-Windows target.
