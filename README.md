# Velox

- Status: M4 complete; M5 narrow alpha active; beta gated by product workflows with no human adoption claim
- Scope: general
- Repository type: cli-tool

Velox is a compile-free Windows desktop application packager. It is designed
to turn static HTML, CSS, and JavaScript into a
portable WebView2 application without compiling application-specific native
code.

The repository now contains a policy-enforcing pure-Go WebView2 host, a frozen
and permission-checked IPC v1 bridge, manifest
validation, an immutable build plan, recoverable portable-output assembly, a
deterministic ZIP writer, all seven M1 CLI commands, an unsigned deterministic
Windows x64 release bundle, startup fixtures, zero-cache consumer evidence,
and an alpha evidence workflow. The workflow builds the release twice, emits
checksums, a file-level SPDX SBOM, an unsigned in-toto/SLSA provenance
statement, and then exercises it from a checkout-free consumer job. A guarded
manual job publishes those exact files as an explicitly unsigned developer
preview.

[Velox v0.5.10-alpha.1](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.1)
is the first public unsigned developer preview. Tag evidence
[run 29714104653](https://github.com/0disoft/velox/actions/runs/29714104653)
and publication [run 29714173324](https://github.com/0disoft/velox/actions/runs/29714173324)
produced the immutable release from commit
`9f10c545b6bde23d2c3dad5bbb12bffdac513712`. Public-download verification
[run 29715002921](https://github.com/0disoft/velox/actions/runs/29715002921)
then exercised the release without source checkout at verifier commit
`17a91f5c90dcbd58cf8aa20836994097e9c3262b`. The ZIP SHA-256 is
`5df53090e1e67ce54c8639f061ffc7b03b7c3aa38f95a725c29342cfaff73b68`.
The executables remain unsigned and the provenance remains unauthenticated
metadata.

[Velox v0.5.10-alpha.40](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.40)
is a historical unsigned developer preview from commit
`d206fe4ef1be9df198d86809742ef480549344b8`. Tag evidence
[run 34214224962](https://github.com/0disoft/velox/actions/runs/34214224962),
publication [run 34214445883](https://github.com/0disoft/velox/actions/runs/34214445883),
and public-download verification
[run 34215188131](https://github.com/0disoft/velox/actions/runs/34215188131)
passed. The public ZIP SHA-256 is
`771173b6eec2f74d92228e7ac5b52332160b0f9fb4d7baecf01976864a00f8c8`.
This candidate includes retained-callback ownership and partial-initialization
hardening. This release is not a beta promotion or human-adoption claim.
Current channel requirements are in [Product Readiness](docs/ops/product-readiness.md).

Start with the [Velox Release Quickstart](docs/QUICKSTART.md) to verify and use
an immutable public release without a source checkout or consumer toolchain.
Report ordinary failures with the [bug report template](https://github.com/0disoft/velox/issues/new?template=bug-report.md);
report suspected vulnerabilities privately through [SECURITY.md](SECURITY.md).

The current unsigned preview is [v0.5.10-alpha.66](https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.66),
from commit `842fead0c889e9f161c2567a91c8d0fd4c2ca260`. It adds image and font
asset detection to `run --watch`, opt-in metadata-only `run --debug` JavaScript
diagnostics, and bounded local IndexedDB draft recovery for newly generated
`init --template text-editor` projects, while retaining the alpha.65 text-editor
and folder-browser `init --template` starters, opt-in `run --watch` automatic
development reload, EXE branding, per-user installers, native
text/folder/clipboard operations, desktop controls and declaration-only
TypeScript bridge types. Existing generated projects are not upgraded.
Portable static apps remain the default.
[Tag CI 37317775399](https://github.com/0disoft/velox/actions/runs/37317775399)
built reproducible unsigned producer evidence and a checkout-free consumer
smoke for tag `v0.5.10-alpha.66` at source `842fead` with Go 1.26.0. The same
four assets were published without a second producer run. ZIP SHA-256:
`3e1bc83cc8ee31e8feb4870a5b4fa26a4f992a5e22754d0625de891394cfacdc`.
Unauthenticated public downloads matched the CI artifact checksums; the release
manifest, SPDX, provenance and the public CLI version were checked. At
publication, verification covered artifact identity only; no native UI, startup, installer,
watch, draft or public native/picker interaction ran against these published
bytes. The prior alpha.65 unsigned preview remains a historical record, and its
exact-public-native watch and starter-picker evidence is not upgraded to
alpha.66. The earlier alpha.62 public-download verification run `35079337819`
and its native lifecycle evidence also remain separate historical records. See
the [release record](docs/ops/release.md) for exact scope and retained failures.
A later local follow-up ran the existing native smoke scripts against these
exact downloaded bytes; see
[the alpha.66 public native follow-up](docs/ops/alpha66-public-native.md). It
adds no installer, folder-browser, hosted-verifier, stress or adoption evidence.
Beta remains held under the product workflow checklist.

Source version: `0.5.10-alpha.67` (local candidate; not published). Local
preparation is recorded in [alpha67-preparation.md](docs/ops/alpha67-preparation.md).
The public preview remains alpha.66; its four asset identities and scope are in
[the alpha.66 publication record](docs/ops/alpha66-publication.md).

## Headline Metrics

1. End-to-end cold build time.
2. Consumer GitHub Actions cache upload.

Process-to-ready startup is a release guardrail and lifecycle diagnostic, not a
headline advantage. Velox intentionally trades native feature breadth for a
smaller build and runtime surface.

## Proposed Shape

- A standalone Go CLI validates and packages projects.
- A separate prebuilt generic host opens static assets through WebView2.
- The production host is pure Go with no CGo or C++ shim.
- The retired C++23 M0 comparison remains available as historical ADR and
  performance evidence, not as an active build target.
- Consumer builds copy an unchanged host by default, along with external
  configuration and assets.
- Consumer builds require no compiler, Node.js, or frontend package manager.

## Current Product Boundary

Supported in the published alpha.66 preview:

- Windows x64.
- Static web assets.
- One top-level window.
- Portable directory and deterministic ZIP output.
- Permission-checked JSON IPC for application information, window controls,
  selected text files/folders, clipboard text and confirmed HTTPS links.
- Non-interactive CLI operation and machine-readable output.
- Optional per-application executable branding: an icon and a version
  resource written into a staged host copy on build; without it the build
  keeps the release host's own icon and version resources.
- An optional per-user Windows Setup executable from `velox build --installer`
  for a single Windows account; the portable directory and deterministic ZIP
  remain the default output and the Setup is an additional, opt-in artifact.

Explicitly deferred:

- Native application backends and plugins.
- Arbitrary-path filesystem access, recursive directory access or monitoring,
  shell, process, and sidecar APIs. Selected text-file and immediate-folder
  access are supported through the opt-in capabilities below.
- Frontend bundling, state-preserving hot module replacement, and a development server.
- Automatic updates and repair, machine-wide or elevation-requiring installs,
  MSI/MSIX packaging, and code signing automation.
- macOS, Linux, ARM64, and multi-window support.

## Capabilities and Permissions

The table covers the published alpha.66 preview. Native script capabilities
are separate opt-ins in `security.permissions`; manifest settings and CLI
flags need no script permission.

| Capability | Permission or setting | Example |
| --- | --- | --- |
| App information and basic window lifecycle | `app.info` / `window.basic` | [Deskboard](examples/deskboard) |
| Native window caption | `window.title` | [File Notes](examples/file-notes) |
| Open/save selected UTF-8 files, up to 2 MiB; session-only save target | `file.open` / `file.save` | [Reader](examples/file-reader), [Saver](examples/file-saver), [File Notes](examples/file-notes) |
| Immediate entries in a selected local folder | `folder.read`; also `folder.readText` for UTF-8 contents | [Folder browser](examples/folder-browser), also opts into `clipboard.write` |
| Plain-text clipboard; native approval per read request | Independent `clipboard.write` / `clipboard.read` | [Clipboard](examples/clipboard) |
| Confirmed external HTTPS links | `external.open` | [IPC contract](docs/architecture/04-ipc-v1.md) |
| Taskbar flashing | `window.attention` | [Window attention](examples/window-attention) |
| Taskbar progress | `window.progress` | [Taskbar progress](examples/taskbar-progress) |
| Transient tray balloon; Windows may suppress display | `notification.show` and `window.tray: true` | [Tray notification](examples/tray-notification) |
| Single instance per app identity | `app.singleInstance: true` | [File Notes](examples/file-notes) |
| Tray show/hide/quit and remembered placement | `window.tray: true` / `window.rememberState: true` | [File Notes](examples/file-notes) |
| Activation shortcut for the existing window | `window.activationShortcut` | [Activation shortcut](examples/window-activation-shortcut) |
| Minimum window size | `window.minWidth` / `window.minHeight` | [File Notes](examples/file-notes) |
| Topmost, fixed-size and system-themed native title bar | `window.alwaysOnTop` / `window.resizable: false` / `window.followSystemTheme: true` | [Topmost](examples/window-always-on-top), [Fixed size](examples/window-fixed-size), [System theme](examples/window-system-theme) |
| App icon/EXE metadata and optional per-user installer | `branding` / `build --installer` | [Manifest schema](schema/velox-v1.schema.json), [Installer guide](docs/ops/windows-installer.md) |

The generated `text-editor` starter requests only `file.open`/`file.save`;
the `folder-browser` starter requests only `folder.read`/`folder.readText`.
Repository examples may opt into additional permissions. These selected-file
and folder capabilities do not grant arbitrary paths, restart-persistent file
access, recursive traversal or folder writes. Clipboard history and monitoring
are not provided.

## Documentation

- Product scope: docs/product/02-spec.md
- Product brief: docs/product/00-product-brief.md
- Roadmap: docs/product/01-roadmap.md
- Risk register: docs/product/03-risk-register.md
- Architecture: ARCHITECTURE.md
- Initial decision: docs/adr/0001-initial-architecture-boundaries.md
- CLI contract: docs/cli/command-contract.md
- Performance budget: docs/engineering/03-performance-budget.md
- Security policy: SECURITY.md
- Privacy policy: PRIVACY.md
- Unsigned preview decision: docs/adr/0011-publish-unsigned-developer-preview-before-signing.md
- Preview identity decision: docs/adr/0012-bind-preview-version-and-public-download-evidence.md
- Public-name decision: docs/adr/0015-retain-velox-public-identity.md
- Distribution/adoption boundary: docs/adr/0016-separate-technical-distribution-from-independent-adoption.md
- M5 product decision: docs/adr/0017-continue-as-a-narrow-static-packager.md
- Agent-evaluation decision: docs/adr/0018-use-clean-room-llm-agent-evaluation.md
- Branding decision: docs/adr/0020-optional-compiler-free-executable-branding.md
- Windows installer: docs/ops/windows-installer.md
- Deferred SignPath onboarding: docs/ops/signpath-onboarding.md
- External user attempt: docs/ops/external-user-attempt.md
- Clean-room LLM agent evaluation: docs/ops/llm-agent-evaluation.md
- Public release Quickstart: docs/QUICKSTART.md

## Current CLI Slice

The CLI expects an unchanged prebuilt `velox-host.exe` and strict
`velox-host.json` beside `velox.exe` in a release bundle. It verifies release,
target, host-contract, runtime-contract, IPC-contract, file-size, and SHA-256 agreement before
building. Consumer builds never invoke Go, C++, Node.js, Pixi, or a package
manager.

```powershell
velox init .\hello --json
velox validate --config .\velox.json --json
velox doctor --config .\velox.json --out .\dist --json
velox run --config .\velox.json --out .\.velox-run --json
velox build --config .\velox.json --out .\dist --json
velox inspect .\dist\dev.velox.hello.zip --json
velox version --json
```

The published alpha.66 includes the two opt-in native starters introduced in
alpha.65:

```sh
velox init my-editor --template text-editor
velox init my-browser --template folder-browser
```

Text-editor requests only `file.open`/`file.save`. Folder-browser requests only
`folder.read`/`folder.readText` for immediate listing and readonly UTF-8
preview, with no recursion or writes. Both use local licensed icons, without a
bundled font or new host dependency. Omitting `--template` keeps the
permission-free basic starter. Alpha.65 had no draft recovery; it ships in
alpha.66.

Published alpha.66 includes local IndexedDB draft recovery for newly generated
text editors. Restore recovers unsaved text, not file paths or save
permissions; the next Save selects a destination again. Existing generated
projects are not upgraded. See the
[draft recovery record](docs/ops/text-editor-drafts.md) for bounds and checks.

The source CLI also provides an opt-in tray starter, not yet in the public
alpha.66 download:

```sh
velox init my-tray --template tray-app
```

It grants only `notification.show` and enables the existing host tray menu,
single-instance behavior, remembered placement and system-themed title bar.
The composer sends info/warning/error messages only on explicit submission;
Windows can suppress balloon display. It includes one local licensed bell
icon, no bundled font, history, scheduler or additional host code. The basic
starter and existing generated projects are unchanged.
Scope and verification are in the [tray starter record](docs/ops/tray-app-starter.md).

The published alpha.66 supports `velox run --watch --config velox.json`:
stable HTML/CSS/JavaScript saves and common image/font path, size and
modification-time changes trigger full-page reloads without enabling
DevTools. Keep the downloaded CLI/host pair together. App `beforeunload`
handlers can cancel a reload. Normal `run` and packaged apps do not start a
watcher. Alpha.65 watched only text, not fonts/images or manifest changes;
alpha.66 adds metadata-only image/font watching. Same-size binary edits with a
preserved modification time are not detected; manifest changes and unlisted
asset formats remain outside the watch scope. Published alpha.66 `run --debug`
also emits bounded JavaScript error metadata to local stderr, including with
`--json`; messages and rejection contents are not collected. Default runs
install no diagnostic channel. Scope, privacy and native receipts are in
[Development Diagnostics](docs/ops/development-diagnostics.md).

The source CLI additionally detects stable edits to the selected manifest in
`run --watch`. Valid edits print a restart-required notice; invalid or unreadable
settings print a nonfatal error. The running app, permissions and original
asset directory stay unchanged until a manual restart. Normal runs and packaged
apps start no manifest watcher. Manifest notices and nonfatal validation errors
also reach stderr with `--json`, while stdout remains one JSON envelope.
Host stderr stays suppressed in JSON mode unless `--debug` is explicitly
enabled. This extension is not in the public alpha.66 download;
see [Development Watch](docs/ops/development-watch.md#manifest-notices-2026-10-06).

Local candidate verification and extraction instructions are in
[the alpha.64 candidate receipt](docs/ops/alpha64-candidate.md); it is not a
published release download. Use the public release linked above for published bytes.

`build` produces `dist/<app-id>/`, `dist/<app-id>.zip`, and a deterministic
`build-result.json` inside the portable directory and archive. The host bytes
are copied unchanged by default. Output assembly occurs in an owned sibling
staging path; an occupied staging or recovery path fails closed instead of
deleting it.

With a `branding` object, the build writes the icon and version resource into
a staged copy of the host and reports the edited host size and SHA-256 in
`build-result.json`; the released host template is never modified.

See `examples/hello/velox.json` and `schema/velox-v1.schema.json` for the v1
authoring contract.

## Functional Test App

`examples/deskboard` is a complete local-first task board built from static
HTML, CSS, and JavaScript. It persists versioned task data in the application
WebView2 profile, exercises the supported app and window IPC methods, and ships
without a frontend dependency or bundler. Its model tests and packaging/startup
smoke prove a more realistic application path than the minimal hello fixture.

`examples/capability-probe` reports which browser-owned storage, file-picker,
clipboard, drag-and-drop, and permission surfaces are exposed by the current
WebView2 environment. It keeps every operation user-initiated and does not add
or imply a Velox native capability.

`examples/file-notes` is a UTF-8 Markdown editor using bounded native dialogs
under `file.open` and `file.save`, plus `window.title` for its native caption,
with session-only connected saving,
external-change conflict protection, IndexedDB draft recovery and unsaved-change
protection. Restored drafts require fresh save selection; no file path or write
grant is persisted.

`examples/folder-browser` uses opt-in `folder.read` for native folder selection
and bounded immediate-entry listing, plus separate `folder.readText` for
on-click, read-only UTF-8 previews. It exposes no full paths and adds no write
grant, directory watcher, recursive scan or persisted grant.

[`examples/clipboard`](examples/clipboard/README.md) uses independent clipboard
write/read permissions for explicit Copy and host-confirmed Paste actions.
It displays plain text only and adds no storage, history or monitoring.

[`examples/window-attention`](examples/window-attention/README.md) uses only
`window.attention` for bounded taskbar flashing and explicit cancellation.

[`examples/taskbar-progress`](examples/taskbar-progress/README.md) uses only
`window.progress` for application-supplied taskbar progress and explicit clearing.
Its cancellable delay permits a manual background-window check; the host adds
no timer, dependency, notification or foreground activation.

[`examples/window-always-on-top`](examples/window-always-on-top/README.md) is
an unsaved scratchpad with `window.alwaysOnTop: true` and no native permissions.
The host-owned system menu lets users toggle topmost; restarting reapplies the
manifest default. File Notes keeps the default off.

[`examples/window-fixed-size`](examples/window-fixed-size/README.md) demonstrates
`window.resizable: false` with saved-position restoration and no native
permissions. Its scratchpad text is unsaved; only the window position persists.

[`examples/window-system-theme`](examples/window-system-theme/README.md) is an
unsaved scratchpad with `window.followSystemTheme: true` and no native
permissions. The host follows the Windows app light/dark setting for the native
title bar where supported; application CSS stays app-owned.

## Development State

M0 selected the pure-Go WebView2 host, M1 completed the compile-free packaging
slice, and M2 closed the minimum runtime security contract. M3 has passed its
publishable Wails cold-build gate and its narrowly defined structural-
simplicity gate. Startup has been removed from the headline and retained as a
release guardrail. M3's public benchmark deliverables and hosted evidence are
complete. M4 has local and hosted unsigned alpha evidence, a published public
developer preview, same-repository public-download verification, and a
separate public clean-room consumer repository.
Deterministic signing-input, lineage, and
fail-closed Authenticode verification tooling remain dormant for a future
signed channel. ADR 0016 closes M4 on technical distribution evidence, ADR
0017 continues the narrow alpha, and ADR 0019 replaces mandatory AI evaluation
with product workflow gates. AI evaluation is optional and cannot authorize
beta promotion. Passing product checks does not claim human adoption. Provider-approved signing and authenticated
provenance are not M4 gates. The current published preview is
`0.5.10-alpha.66`.
Neither same-repository verification nor the maintainer-controlled consumer
repository counts as independent adoption.

The now-archived public
[`0disoft/velox-consumer-smoke`](https://github.com/0disoft/velox-consumer-smoke)
repository consumed only the pinned public release once in hosted
[run 29736140250](https://github.com/0disoft/velox-consumer-smoke/actions/runs/29736140250).
It invoked no consumer compiler, Node.js, package manager, or Actions cache and
passed release, deterministic-build, inspection, and startup checks. Its schema
fixes `maintainerControlled: true` and `externalUserAttempt: false`.
The repository is retained read-only as historical evidence; ongoing public-
release verification remains in Velox itself.

The bounded maintenance-cost snapshots and internal M4 security review are now
complete M5 inputs. The immutable v1 record preserves the former weekly hosted-
job ceiling; the current v2 record captures the manual-only scheduling boundary
and zero recurring jobs. They also record the maintained WebView2 fork,
unsigned-channel trust limit, and accepted mutable-asset boundary. They are not
person-hour estimates or an independent audit.

ADR 0015 retains Velox as the maintainer-approved product, command, module,
schema, and release identity. The collision review still records Meta's
established project and an existing Go CLI that ships the exact `velox` command
and `velox.exe`; those are accepted discovery and command risks rather than a
replacement-name publication gate.

Consumer release packaging is published as an unsigned developer preview.
`init` creates a
dependency-free starter, `doctor` checks the current Windows, WebView2, project,
and bundled-host compatibility, `run` launches source assets through the
prebuilt host without a development server, and `inspect` validates both
portable directories and ZIPs
without executing them. The parent workspace exposes bounded
maintainer-only release-bundle, compiler-free consumer smoke, host smoke, and
benchmark intents documented in `DEVELOPMENT.md` and `VALIDATION.md`.
The repository-owned consumer workflow keeps maintainer compilation in a
producer job, measures isolated consumer jobs from artifact acquisition through
portable ZIP inspection, and publishes raw and aggregated result contracts.

## Repository Workflow

- Agent instructions: AGENTS.md
- Validation names: VALIDATION.md
- Checklist router: CHECKLIST.md
- Documentation index: docs/README.md
- Scaffold state: .ssealed/manifest.json

## License

Velox is licensed under `MIT OR Apache-2.0`, at your option. See `LICENSE-MIT`
and `LICENSE-APACHE`. Third-party attributions are listed in
`THIRD_PARTY_NOTICES.md`.
