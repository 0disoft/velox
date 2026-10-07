# Command Contract

- Status: Draft
- Repository Type: cli-tool
- Owner: Project maintainer

## Interface Principles

- Commands are non-interactive by default in CI.
- Human output is concise; JSON output is stable and versioned.
- Build commands do not access the network.
- A failure never reports success because a feature is unavailable.
- Paths in diagnostics are project-relative when possible.
- Output never includes source file contents, secrets, or environment dumps.

## Implementation Status

`init`, `validate`, `doctor`, `run`, `build`, `inspect`, and `version` are
implemented in the M1 vertical slice. The
release bundle must place the unchanged prebuilt `velox-host.exe` and its
`velox-host.json` beside the CLI. The CLI verifies release, target, contract,
size, and digest agreement; there is intentionally no public flag that
substitutes an arbitrary host.

An opt-in `build --installer` flag also packages a per-user Windows Setup
executable. It is a build flag, not a separate command, and the default
portable output is unchanged.

The source CLI additionally implements the read-only `templates` command;
it is not included in the published alpha.67 CLI.

Source CLI `help`, `--help` and `-h` include command descriptions, generation/
run/build examples and a pointer to `velox <command> --help`. Successful
top-level help remains on stdout; missing or unknown commands print usage
on stderr and exit 2. `init --help` adds defaults, a generation example and
`velox templates` discovery before its flag list on stderr. It exits 0
without creating a project. As before, subcommand help requested with
`--json` exits 0 with no output. Expanded help is not in published alpha.67.

## MVP Commands

### velox templates

List built-in starters in stable order: `basic`, `text-editor`,
`folder-browser`, `tray-app`. Each entry shows its purpose, the native
permissions written by `init`, and a ready-to-run generation command.

- No project, host or runtime is required. No files are read or written and
  no network, native API or application is invoked.
- Accept only `--json`, `--quiet` and `--help`; positional arguments and
  unsupported options exit 2 with `USAGE_INVALID`.
- `--quiet` suppresses successful human output; `--json` takes precedence
  over it and emits one schema-version-1 envelope with command `templates`.
- The JSON result is `{ "templates": [...] }`. Each entry has `name`,
  `description`, `permissions` (an array, empty for `basic`), and `initCommand`.
  Descriptions are informational; names and permissions identify the template.
- Listing and project generation share the initializer's permission catalog.
  Listing does not grant any permission or modify template defaults.

### velox init [directory]

Create a minimal manifest and dependency-free static web example.

- `--template basic` is the default and keeps the existing output and empty
  permissions. `--template text-editor` creates the native text-editor starter
  with only `file.open` and `file.save`. `--template folder-browser` creates a
  read-only folder starter with only `folder.read` and `folder.readText`.
  Unknown templates exit 2 with
  `USAGE_INVALID` and write nothing. This option ships in public alpha.65 and
  later, including the current alpha.66; alpha.64 does not include template selection.
- The source CLI also supports `--template tray-app` with only
  `notification.show`. It enables `app.singleInstance`, `window.tray`,
  `window.rememberState` and `window.followSystemTheme`, uses a 620x480 window
  with 360x400 logical minimums, and reserves no activation shortcut.
  This starter is not in the published alpha.66 CLI.
- Derive a conservative `dev.velox.<directory>` application ID and display name
  from the target directory.
- Successful human output also prints run and build commands pointing to the
  generated project's `velox.json`, with forward-slash paths and literal
  quoting for the labeled shell: PowerShell on Windows, POSIX shell elsewhere.
  Apostrophes are escaped for that shell. These are instructions only; no
  command is executed, host downloaded or generated file changed. `--json`
  and `--quiet` output are unchanged. This guidance is source-only and is not
  in public alpha.67.
- Preflight every planned path and refuse the operation if any generated file
  already exists.
- Remove only files and directories created by the failed invocation.
- Write a root `velox.d.ts` byte-identical to the declaration embedded in the
  CLI, beside the manifest and `web/` files. The generated `web/app.js` opens
  with `/// <reference path="../velox.d.ts" />`, a type-checking hint only.
- Keep the declaration at the project root; do not copy it into the `web` asset
  directory or add it to generated application assets.
- Grant no native permission for the basic template, install no TypeScript,
  and compile or run no example. Each native template explicitly declares
  only the permissions listed above in the generated manifest.
- Do not install frontend dependencies.
- Do not download a host or runtime.

The published alpha.67 text-editor starter includes New/Open/Save/Save as, document-scoped native
save-target reuse, a discard dialog, close protection and IME-aware keyboard
actions. The first Save after Open still prompts for a save target. It includes
four local Lucide icons and their license, but no bundled font, find or preview.
Public alpha.65 had no draft storage or recovery; alpha.66 adds it. Its assets
add no runtime dependency or host code.

The source text-editor starter additionally includes a hidden-by-default
find bar: Ctrl+F, Enter/Shift+Enter navigation, Escape to close, match count
and a Match case checkbox. Search is literal, Unicode case-insensitive by
default, non-overlapping and wraps at the ends using original UTF-16 offsets.
Only an open find bar searches text; navigation scrolls the selected match
into view. IME and modal/busy guards remain active. Search state is memory-only
and never changes document text, native save targets or draft records.
The close button also dismisses an empty query or document and cancels pending
find-input composition. Escape closes once its key event is no longer
composing, including legacy key-code-229 events. Late composition-end events
cannot move selection or return focus to a closed find bar.
Four additional licensed icons and `find.js` are included, with no new
permission, dependency, host code, regex UI, replace or preview. This affects
newly generated projects only and is not in public alpha.67.

Published alpha.66 text-editor generation also includes `drafts.js` and a
Restore/Discard dialog. One local IndexedDB record contains only
`schemaVersion`, `name`, `text` and `updatedAt`, with a 2 MiB UTF-8 text limit.
Writes debounce for 300 ms and serialize with clears; success is reported
only after transaction completion. Restored text is dirty and has no save
target, so its first Save reselects a destination. Storage errors preserve
editing and unsaved-change protection. No native path, token, saved-text
baseline or permission is persisted. This affects newly generated projects,
not existing projects; host, IPC and permissions are unchanged.

The folder-browser starter provides folder selection, immediate-entry listing,
explicit refresh, readonly UTF-8 file preview and folder release. It reuses
the existing host bounds and document-scoped folder tokens. Selection
cancellation retains the current view; unsupported reads clear the preview;
expired tokens clear both panes. It includes three local licensed icons but
no subfolder navigation, clipboard, writes, monitoring, bundled font or new
dependency. Folder-browser does not change the basic or text-editor template.

The source-only tray starter includes a message composer, kind selector,
bounded count and explicit Send action. It calls only the existing
`notification.show` method, blocks duplicate submissions, preserves text on
success/failure and displays error codes without raw native details. Its limit
matches the host's 255 UTF-16 units and control-character rules. Hide/restore/
quit use the existing host-owned tray menu; accepted requests do not guarantee
Windows will display the balloon. It includes a local licensed Lucide bell
icon and no browser fallback, message history/storage, timer, font or dependency.
Other starters and the basic output are unchanged.

### velox validate

Validate manifest syntax and semantics, asset paths, entry point, permissions,
security policy, target support, and host compatibility without creating
output.

### velox doctor

Report local prerequisites and compatibility, including Windows architecture
and build, WebView2 Runtime availability and version, project configuration,
and bundled host
compatibility. Doctor is read-only.

- Query the installed runtime through the same bundled WebView2 loader used by
  the host instead of inferring availability from registry paths.
- Report platform, Windows build, runtime, project, and host checks in stable
  order.
- Keep the complete check result in JSON on failure while returning the
  corresponding non-zero prerequisite, project, or host exit code.
- Require Windows 10 version 1709 x64 or Windows Server 2016 x64, and Evergreen
  WebView2 Runtime `92.0.902.49` or newer.

### velox run

Launch the prebuilt host against the source asset directory. Default runs do
not start a watcher; no run starts a development server, bundler or hot module
replacement process.

- Validate the same project, asset, target, and bundled-host contracts as build.
- Create a unique runtime configuration beside the project manifest so relative
  asset containment remains identical to packaged applications.
- Remove the temporary configuration after normal or unsuccessful host exit.
- Close child stdin, wait for the host, and preserve its non-zero exit code.
- Suppress child stdout in JSON mode so stdout remains one JSON document.
  Child stderr is also suppressed unless `--debug` is explicitly enabled.
  Source CLI manifest-watch notices use a separate stderr path; `--watch --json`
  reports them even with debug off, without forwarding ordinary host logs.
- Do not copy source assets or create build output.
- `--debug` explicitly enables development tools and cache bypass.
  Alpha.66 additionally installs top-level metadata-only error
  diagnostics: fixed `uncaught-error`/`unhandled-rejection` categories,
  known startup asset relative path, line and column. Unknown sources and
  Promise rejection locations become `<unknown>:0:0`; no messages, stacks,
  rejection reasons or console bodies are read. Native attempts are capped
  at 20 per host run, including invalid requests. `--debug --json` forwards
  host stderr while retaining one stdout envelope. Normal, watch-only and
  packaged defaults install no diagnostic listeners or binding. This
  diagnostics extension is not in published alpha.65; it ships in alpha.66.
- `--watch` independently enables full-page reload after source HTML, HTM,
  CSS, JS or MJS contents settle for 500 ms, sampled every 500 ms. The entry
  file is always included. Alpha.66 also watches image extensions
  PNG/APNG/JPG/JPEG/GIF/WebP/AVIF/BMP/ICO/SVG and font extensions
  WOFF/WOFF2/TTF/OTF/EOT, case-insensitively, by path, size and modification
  time only. Binary contents are not repeatedly read or counted in the text
  budget. Additions, deletions and renames count; reverted text contents do
  not. Same-size binary edits with preserved modification times cannot be
  detected. Other asset formats remain outside the reload scope.
- The source CLI additionally watches the selected project manifest (including
  a custom `--config` path) by content every 500 ms with a 500 ms quiet period.
  A stable valid edit emits a restart-required notice to stderr; an invalid,
  missing, unreadable, linked or over-1-MiB manifest emits a nonfatal error.
  Repeated unchanged errors are suppressed and subsequent edits are retried.
  Manifest edits never reload, restart or reconfigure the running host; the
  existing asset watcher continues against the original asset directory.
  The CLI stops and joins this additional loop when the host exits. These
  CLI notices/errors reach stderr even in JSON mode; only host stderr retains
  its suppression unless `--debug` is enabled. Stdout stays one JSON envelope.
  This notice-only extension is source-only, not in published alpha.66.
- Watch is default-off and is passed as a host argument, not stored in the
  manifest, runtime configuration, profile, build report or ZIP. It does not
  enable DevTools; cache bypass is confined to the development WebView.
- Watch validates the asset boundary, refuses links/reparse points, and limits
  watched text to 64 MiB and the asset tree to 10,000 files. Unsafe, unreadable
  or missing-entry intermediate states suppress reload and are retried.
- Reload follows normal browser navigation and the application's
  `beforeunload` protection. Canceling preserves the current page and waits
  for a subsequent edit; this is not state-preserving HMR. Navigation resets
  document-scoped native targets exactly as manual reload does.
- Closing the development window stops and joins the watch loop. Existing
  app identity, profile and single-instance behavior are unchanged; close a
  running app with the same identity/profile before starting a watched run.
- Watch is available in published alpha.64 and later and needs its matching
  CLI/host pair. Earlier alpha.63 binaries do not implement this option.
  Image/font watching is not in published alpha.65; it ships in alpha.66.

### velox build

Validate the project and create a portable application directory,
machine-readable build report, and deterministic ZIP through a recoverable staging
flow.

The current output names use the complete `app.id`, preventing projects with
the same leaf identifier from overwriting each other in a shared output root:

    dist/<app-id>/<app-id>.exe
    dist/<app-id>/velox.runtime.json
    dist/<app-id>/web/**
    dist/<app-id>/build-result.json
    dist/<app-id>.zip

With `--installer`, the build also emits `dist/<app-id>-setup.exe`: the
verified portable ZIP appended to the prebuilt `velox-setup.exe` template that
sits beside the CLI. The template is checked against the adjacent
`release-manifest.json` release version, size, and SHA-256. A missing,
duplicated, or mismatched template fails the installer step with
`PACKAGING_FAILED`: the normal portable directory and ZIP are still produced,
but no Setup executable is published and the command does not report a
successful installer. The portable directory and ZIP remain the default output.
See `docs/ops/windows-installer.md`.

The successful `build` result includes an optional `installer` object with
`file`, `bytes`, and `sha256`. A default build without `--installer` omits the
`installer` key entirely.

The Setup executable is unsigned, and the current trailing-footer reader does
not support signing it: an Authenticode signature appends data after the
footer, so a signed Setup needs a separate future payload design rather than
signing the produced file as-is.

The ZIP contains one top-level `<app-id>/` directory. File order, timestamps, and
portable file modes are normalized. The deterministic report contains contract
versions, release version, identity, permissions, host and asset digests, and
counts; it omits
wall-clock timings and absolute paths. Build duration belongs to benchmark
evidence rather than reproducible artifact bytes.

Build and ZIP inspection share limits of 100,000 files, 512 MiB per file,
and 1 GiB of total uncompressed data, including the host and metadata.
Each runtime configuration and build report is limited to 1 MiB. Known source
size and file-count violations fail before staging; streaming checks include
the final metadata bytes and prevent oversized output from being published.
The same compression-ratio check is applied before ZIP publication and during
inspection. Existing outputs are preserved when a limit is exceeded.

### velox inspect PATH

Read an output directory or archive and report its Velox release, contract
versions, target, permissions, application identity, file counts, byte counts,
and digests without executing it.

Inspection recomputes the host and asset-tree SHA-256 values and validates the
runtime configuration against the build result. ZIP inspection rejects unsafe,
duplicate, case-colliding, multi-root, unexpected, or over-limit entries. Every
ZIP entry must be a regular file; symbolic links and other special-file types
are rejected even when their content digests match the report.

### velox version

Report the CLI version, supported manifest versions, host compatibility range,
IPC versions, and bundled targets.

## Common Options

| Option | Contract |
| --- | --- |
| --config PATH | Project manifest; default is velox.json |
| --target TARGET | Explicit build target; MVP accepts windows-x64 |
| --out PATH | Output root; default is dist |
| --json | Emit one JSON document and no decorative human output |
| --quiet | Suppress non-error human output |
| --verbose | Add bounded human-mode diagnostics to stderr without secrets, source contents, absolute paths, timestamps, or timing claims; quiet wins |
| --help | Print command help and exit successfully |
| --version | Alias the version command |

`--out` is resolved relative to the manifest's project root. The output root
and asset root may not contain each other.

Command-specific options must be added to this document before implementation
is considered stable.

`doctor`, `run`, `validate`, and `build` share the project options above.
`init`, `templates`, `inspect`, and `version` expose only their command-specific subset;
unsupported options fail instead of being silently ignored.

## Configuration Precedence

1. Explicit command-line options.
2. The project manifest.
3. Documented built-in defaults.

Environment variables do not configure application identity, permissions, or
packaging. Development and benchmark-only environment variables must be named,
documented, and ignored by production builds.

The CLI does not search parent directories beyond the resolved project root and
does not merge multiple manifests.

## JSON Envelope

Successful commands return:

    {
      "schemaVersion": 1,
      "ok": true,
      "command": "build",
      "result": {},
      "diagnostics": []
    }

Failed commands return:

    {
      "schemaVersion": 1,
      "ok": false,
      "command": "build",
      "error": {
        "code": "MANIFEST_INVALID",
        "message": "Project manifest is invalid."
      },
      "diagnostics": []
    }

Diagnostics use stable codes, severity, category, project-relative path,
optional line and column, a short message, and structured facts. They do not
contain timestamps, random identifiers, progress events, or absolute local
paths unless no safe relative representation exists.

## Exit Codes

| Code | Meaning |
| ---: | --- |
| 0 | Success |
| 2 | Usage, manifest, or configuration error |
| 3 | Asset or project input error |
| 4 | Host template or contract compatibility error |
| 5 | Runtime prerequisite unavailable |
| 6 | Packaging or filesystem failure |
| 10 | Unexpected internal failure |

Stable diagnostic codes provide detail within these broad process exit codes.

From alpha.62, closing the native host window during initial
construction is a normal user cancellation (exit 0, no runtime-error message).
Missing WebView2 or an actual initialization failure is not converted to a
successful cancellation. This does not bypass document close consent after
initialization.

## Failure and Recovery

- validate and doctor do not write project or output files.
- build writes only to an owned staging directory until completion.
- build removes its staging directory after a handled failure.
- build preserves the previous successful output.
- Recovery can resume when directory restoration succeeded but archive restoration
  was interrupted. Windows file locks may block replacement; retained backups
  remain available for retry after the lock is released. This is process-interruption
  recovery, not a power-loss durability guarantee.
- run returns the child host exit reason and cleans benchmark-only resources.
- Cancellation follows the same cleanup boundary as failure.

## Runtime Compatibility

- CLI release artifacts: Windows x64 first.
- Packaged host: Windows x64 first.
- Web runtime: Evergreen WebView2.
- Minimum client: Windows 10 version 1709 x64, build 16299.
- Minimum server: Windows Server 2016 x64, build 14393.
- Minimum WebView2 Runtime: `92.0.902.49`, the first stable runtime that exposes
  the `ICoreWebView2_4` download-denial interface required by the security
  baseline.
- Maintainer implementation language: Go.
- Consumer machine compiler and Node.js requirement: none.

## Deferred Commands

The MVP does not define plugin, add, publish, update, sign, generate, bind,
dev-server, or shell-completion commands. Install packaging is available only
through the opt-in `build --installer` flag; there is no separate installer,
updater, or repair command.

## Review Blockers

- A command changes without synchronized help, examples, JSON, diagnostics, and
  exit-code tests.
- JSON output exposes source contents, secrets, unbounded logs, or unstable
  process data.
- A build command performs an undeclared network request.
- A consumer command invokes a compiler or frontend package manager.
- A release puts the CLI and host in different directories without defining a
  new immutable host-discovery contract.
