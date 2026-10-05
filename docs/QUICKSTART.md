# Velox Release Quickstart

- Audience: Windows developers and clean-room coding agents
- Starting point: one immutable public Velox release
- Consumer toolchain: none

This guide starts from published release bytes. It does not require a Velox
source checkout, Go, C++, Rust, Zig, Node.js, Bun, a frontend package manager,
or a build-system cache.

The PowerShell examples below are consumer-facing local commands.

## 1. Record the release identity

Open the [Velox Releases](https://github.com/0disoft/velox/releases) page and
choose one immutable `vX.Y.Z-alpha.N` or `vX.Y.Z-beta.N` release. Do not use a
moving `latest` URL for evaluation evidence.

Download these assets from that exact release:

- `velox-windows-x64.zip`
- `checksums.sha256`

Record the release tag, release URL, and expected ZIP SHA-256 before executing
anything. A checksum downloaded from the same release is an integrity check,
not an independent publisher identity or authenticated attestation.

## 2. Verify the ZIP

In a new empty working directory containing the two downloaded files:

```powershell
$ChecksumMatches = @(Get-Content -LiteralPath .\checksums.sha256 |
  Where-Object { $_ -match '^[0-9A-Fa-f]{64}\s+velox-windows-x64[.]zip$' })
if ($ChecksumMatches.Count -ne 1) {
  throw "Expected exactly one Velox ZIP checksum."
}
$Expected = ($ChecksumMatches[0] -split '\s+')[0].ToLowerInvariant()
$Observed = (Get-FileHash -LiteralPath .\velox-windows-x64.zip -Algorithm SHA256).Hash.ToLowerInvariant()
if ($Expected -ne $Observed) {
  throw "Velox release checksum mismatch."
}
```

For clean-room evaluation, also require `$Observed` to equal the independently
supplied expected digest. Stop before extraction when either comparison fails.

## 3. Extract the release

The Velox CLI tool is distributed as a portable ZIP. Extract the entire
archive before running it; do not run an executable from inside the ZIP or
copy an executable out by itself. Keep the files in each extracted directory
together. No automatic update is provided.

```powershell
Expand-Archive -LiteralPath .\velox-windows-x64.zip -DestinationPath .\tool
$Velox = (Resolve-Path -LiteralPath .\tool\velox-windows-x64\velox.exe).Path
```

Keep `$Velox` as an explicit absolute executable path during the evaluation.
The `velox.exe` name collides with unrelated software and should not be assumed
to resolve safely through `PATH`.

## 4. Exercise the public CLI

Create all project and output files under this clean working directory:

```powershell
& $Velox version --json
& $Velox init .\work\hello --json
& $Velox validate --config .\work\hello\velox.json --json
& $Velox doctor --config .\work\hello\velox.json --out ..\doctor --json
& $Velox build --config .\work\hello\velox.json --out ..\dist --json
& $Velox inspect .\work\dist\dev.velox.hello.zip --json
& $Velox run --config .\work\hello\velox.json --out ..\run --json
```

Relative `--config` and `inspect` paths start from your current working
directory. Relative `--out` paths start from the project directory containing
`velox.json`. Here, `..\dist` therefore selects `work/dist`, not
`work/hello/work/dist`. An absolute `--out` path is also supported.

`run` stays attached to the desktop application. Close the application window
to let the command finish. A visible window alone is not proof of usable
content; verify that the generated page rendered before closing it.

## 5. Confirm the output boundary

The build should produce:

```text
work/dist/dev.velox.hello/
work/dist/dev.velox.hello.zip
```

The portable directory contains the unchanged generic host, runtime
configuration, static web assets, and `build-result.json`. The ZIP is unsigned,
and directory assets are not protected against a local writer. Windows
SmartScreen may warn, and managed Windows policy may block execution.

To use the packaged app directly, keep `dev.velox.hello.exe`,
`velox.runtime.json`, `build-result.json`, and `web/` together in the portable
directory. Double-click the EXE there; moving only the EXE can break startup
or file access. The application ZIP can likewise be extracted into its own
directory before use.

Verify the exact release URL and digest before deciding whether to run an
unsigned executable. A matching checksum does not authenticate its publisher.
Do not disable Windows protection or override a managed-device policy to make
the preview run. If execution is blocked, record the warning and report it.
If `doctor` reports a missing WebView2 Runtime, obtain the Evergreen Runtime
only from [Microsoft's WebView2 download page](https://developer.microsoft.com/en-us/microsoft-edge/webview2/);
the Velox ZIP does not install it.

## 6. Edit and reload

For local development, enable the host's development tools explicitly:

```powershell
& $Velox run --config .\work\hello\velox.json --debug
```

Edit the files under `work/hello/web`, save them in your editor, then use the
WebView2 context menu to reload. Development tools are available for inspecting
errors and disabling the browser cache while editing. This is manual reload,
not a file watcher or hot module replacement; no development server is started.

The app ID and profile location stay the same, so reload and restart do not
deliberately clear IndexedDB or other browser storage. Save in-app edits before
reloading. Close the app before rebuilding its portable output. Without
`--debug`, development tools and default context menus remain disabled; the
flag does not change packaged configuration, native permissions, or origin policy.

## 7. Grant a native permission

Sections 7 and 8 ship in the current published alpha.66 release; only the
historical alpha.62 bundle lacked these features. The first six sections remain
the baseline public-release path.

New bundles' `velox init` writes `velox.json`, three web assets (`web/index.html`,
`web/style.css`, `web/app.js`), and a root `velox.d.ts`. The generated manifest
starts with `"security": { "permissions": [] }`, so the app can call no native
capability until you opt in. Add permissions to the manifest; nothing is
granted implicitly. Replace the existing `security` object's permissions with
the following; keep the rest of your manifest:

```json
{
  "security": { "permissions": ["app.info", "file.save"] }
}
```

The published alpha.66 also offers a native text-editor starter:

```sh
velox init my-editor --template text-editor
velox run --config my-editor/velox.json --watch
velox build --config my-editor/velox.json --installer
```

Use the alpha.66 CLI with its matching release bundle; alpha.65 added template
selection and alpha.64 did not include it. The generated manifest requests only
`file.open` and `file.save`. Initial Save after Open selects a destination;
later Save reuses that page's save target. Omitting `--template` retains the
basic starter.

New text-editor projects generated with alpha.66 store one local IndexedDB
draft after a 300 ms typing pause and offer Restore/Discard on relaunch.
Restore keeps the document unsaved; the next Save selects a new destination
because file paths and save permissions are never persisted. Alpha.65 had no
draft storage or recovery, and existing generated projects are not upgraded.
Draft storage failure is reported without disabling editing or bypassing
unsaved-change protection. Abrupt exit can lose edits not yet committed to
IndexedDB. See [the verification record](ops/text-editor-drafts.md).

For a read-only folder browser, use the second native starter:

```sh
velox init my-browser --template folder-browser
velox run --config my-browser/velox.json --watch
velox build --config my-browser/velox.json
```

Like text-editor, this option ships in the published alpha.66 release. It requests only
`folder.read` and `folder.readText`. Select a folder, refresh its immediate
entries, select a file for a readonly UTF-8 preview, and release the folder
when done. Subdirectories are listed but not navigable. No clipboard, write
or background monitoring permission is added.

`app.info` enables `window.velox.invoke("app.getInfo")`; `file.save` enables
`window.velox.saveText`, `saveTextAs`, and `saveTextTo`. Each permission is
independent and enforced by the host. See the
[configuration guide](cli/configuration.md) and the
[IPC method table](architecture/04-ipc-v1.md#methods).
The save example only needs `file.save`; omit `app.info` if your app does not
read its native identity.

### A click-triggered save

For this minimal example, add one textarea, one button and one output to the
page, then put the JavaScript in `web/app.js`. Keep the existing deferred script
reference in the HTML; the manifest's CSP does not permit inline scripts.

```html
<label for="note">Note</label>
<textarea id="note"></textarea>
<button type="button">Save</button>
<output role="status"></output>
```

```js
const editor = document.querySelector("textarea");
const saveButton = document.querySelector("button");
const saveStatus = document.querySelector("output");
if (!editor || !saveButton || !saveStatus) throw new Error("Save controls missing.");
/** @type {number | undefined} */
let saveTarget;

saveButton.addEventListener("click", async () => {
  const api = window.velox;
  if (!api || typeof api.saveTextAs !== "function") {
    saveStatus.value = "This bundle does not provide native text saving.";
    return;
  }
  if (saveButton.disabled) return;
  saveButton.disabled = true;
  try {
    const saved = saveTarget === undefined
      ? await api.saveTextAs(editor.value, "note.txt")
      : await api.saveTextTo(editor.value, saveTarget);
    if (saved.cancelled) saveStatus.value = "Save cancelled.";
    else {
      saveTarget = saved.target;
      saveStatus.value = "Saved " + saved.name;
    }
  } catch (error) {
    saveTarget = undefined;
    saveStatus.value = "Save failed. Keep your text and retry Save as.";
  } finally {
    saveButton.disabled = false;
  }
});
```

`saveTextAs` opens a native Save-as dialog and returns a document-scoped
`target` only on success; a cancelled dialog returns `{cancelled: true}` and
writes nothing. Later clicks reuse that target through `saveTextTo`. Call
`window.velox.invoke("file.releaseSaveTarget", { target: saveTarget })` before
New in the same page, only when a target is connected. Reset the local target
after release; navigation clears host targets automatically. This example never
clears the editor buffer. `window.velox` is injected only into
a trusted top-level Velox document, so check it before use.

### TypeScript is optional

`velox.d.ts` is a declaration-only mirror of the IPC surface: it ships no
runtime payload and installs no compiler. Plain JavaScript apps need no
TypeScript for `init` or `build`; developers choosing TypeScript compile their
own frontend sources separately and use the declaration for type checking. See the
[type guide](../types/README.md).

## 8. Current host options

Current sources also support window lifecycle and packaging options. This is a
small, non-exhaustive sample; the
[configuration guide](cli/configuration.md) lists every field. Merge these
fields into the existing manifest rather than replacing it:

```json
{
  "app": { "singleInstance": true },
  "branding": { "company": "Rodisoft" },
  "window": { "tray": true, "rememberState": true, "activationShortcut": "Ctrl+Alt+Shift+V" }
}
```

`singleInstance` suppresses a second window; `tray` adds a notification-area
icon; `rememberState` restores window placement across launches;
`activationShortcut` registers a global reveal hotkey; and `branding` writes
Windows version resources during `velox build`. `velox build --installer` also
emits a per-user Setup executable when the release includes its template.
These options are opt-in and default off.

## Failure Boundaries

Stop and preserve the first stable diagnostic when:

- the release or artifact digest differs;
- `doctor` reports an unsupported Windows or WebView2 version;
- a command requests an undeclared compiler, runtime, package manager, network
  dependency, source checkout, or cache;
- inspection disagrees with the expected application identity;
- the application never reaches usable content.

Do not install another toolchain or substitute local source output to make the
trial pass. A localized failure is valid evaluation evidence.

## Report a problem

Use the [Velox bug report](https://github.com/0disoft/velox/issues/new?template=bug-report.md)
for ordinary install, startup, build, or file-access failures. Include the
release tag and ZIP digest, Windows and WebView2 versions, the last successful
step, the first failing step and safe error text, and whether any draft or file
was changed. A policy block is an outcome to report, not a reason to bypass
the policy. Remove private paths, file contents, credentials, tokens, and
proprietary assets from public reports.

For a suspected security vulnerability, use the private reporting route in
[SECURITY.md](../SECURITY.md), not a public issue. Preview support is
best-effort; there is no response-time or backport promise.
