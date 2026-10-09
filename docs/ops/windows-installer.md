# Windows Installer

- Status: Locally verified for beta.3 (all four installer intents); not published
- Owner: Project maintainer
- Decision: ADR 0020

## Scope

The install path is an opt-in, per-user Windows Setup executable. It is an
additional artifact:

- The portable directory and deterministic ZIP remain the default `velox build`
  output and do not change.
- `velox build --installer` also emits `<app-id>-setup.exe` beside them.
- No updater, repair, service, elevation, machine-wide scope, or runtime
  download is implemented.

An install does not expand the application runtime. The static-only asset model,
the closed IPC v1 method table, and the no-consumer-compiler boundary are
unchanged.

## Setup Template

`velox-setup.exe` is a standalone, unsigned, prebuilt Windows x64 GUI
executable built from `cmd/velox-setup`. It is not per-application; the Setup
build appends one application payload to it.

- The release bundle includes the template only when
  `velox-release --setup <path>` is passed; the ordinary bundle is unchanged.
- Before packaging, the CLI verifies the adjacent `release-manifest.json` and
  requires exactly one `velox-setup.exe` artifact whose release version, byte
  size, and SHA-256 match the template on disk. A missing, duplicated, or
  mismatched template fails the build with `PACKAGING_FAILED`.
- The template is copied into the release bundle unchanged.

## Payload Format

`<app-id>-setup.exe` is the setup template with the verified portable ZIP
appended, followed by a 64-byte footer:

- bytes 0-15: the `VeloxSetupV1` magic with trailing NUL padding;
- bytes 16-23: template length in little-endian;
- bytes 24-31: payload length in little-endian;
- bytes 32-63: SHA-256 of the appended payload.

The template must be a Windows x64 GUI executable. A template with a non-empty
security data directory (an already signed image) is refused, because an
appended payload would invalidate the signature.

On open, the footer bounds are validated and the payload SHA-256 is recomputed.
Extraction accepts only canonical, regular ZIP entries under one `<app-id>/`
root, rejects duplicate or case-colliding names, applies the shared
file-count, per-file, total-size, and compression-ratio budgets, and then
re-runs the standard portable-directory inspector on the extracted tree.

## Install Layout

A per-user install needs no elevation:

- `%LOCALAPPDATA%\Programs\Velox\<app-id>\app\` holds the portable output.
- `%LOCALAPPDATA%\Programs\Velox\<app-id>\uninstall.exe` is a copy of the
  unsigned setup template (`velox-setup.exe`) only, without the appended
  application payload.
- `%LOCALAPPDATA%\Programs\Velox\<app-id>\velox-install.json` records
  ownership (`velox.install/v1`): app identity, the install directory, the
  relative path, byte size, and SHA-256 of every installed file, and the
  shortcut hash.
- The Start Menu shortcut is created through native `IShellLink` at
  `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Velox\<app-id>.lnk`.
- The Apps uninstall entry is written under
  `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Velox.<app-id>`
  with `DisplayName`, `DisplayVersion`, `InstallLocation`, `DisplayIcon`,
  `UninstallString`, `VeloxInstaller`, and `NoModify`/`NoRepair`. The entry
  is per-user and is removed with the install.

This is a fresh install only. If the install directory, the shortcut, or the
uninstall registration already exists, installation is refused instead of
overwritten. There is no updater: an update is an uninstall followed by a
reinstall.

Source-only interruption handling (2026-10-10): Setup prepares the native
shortcut in a private temporary folder first. Its exact SHA-256 is committed
with the staged install's ownership record before promoting the install
directory and publishing the final shortcut. There is no post-publication
ownership update, so stopping immediately after shortcut publication leaves
a recognizable installation that can be uninstalled. Ownership writes use a
completed, flushed same-directory temporary file followed by replacement,
not in-place truncation; Windows replacement failure preserves the old record.
This is not in public alpha.68 and does not repair existing damaged records.
Changed shortcuts are still preserved. The native test injects a stop at this
boundary; it is not a forced-process-kill or power-loss durability test.
Interruption during partial registry value creation remains outside this fix.

## Removal

`uninstall.exe --uninstall <app-id>` performs a guarded removal:

- It re-reads the ownership record and refuses to run when the registry entry
  or the Start Menu shortcut is not owned by this installation or its hash
  changed.
- It preflights every file against the recorded size and SHA-256 and refuses
  while any file is changed or unowned, or while any installed file is held open
  by a running application. The application must be closed first.
- It removes only recorded files, the Start Menu shortcut, and the registry key,
  then the now-empty directories. Missing recorded files are tolerated so a
  partial removal can be retried.
- Files outside the install tree are never touched. The WebView2 profile and
  recovery data at `%LOCALAPPDATA%\Velox\profiles\<app-id>` and the user's
  documents are preserved.

Because the running uninstaller holds its own executable open, it copies itself
to a private `%TEMP%\velox-uninstall-*` folder and starts that copy with
`--remove-helper --wait-pid`. The helper waits for the original process to exit
(30-second bound) before removing the install tree, the shortcut, and the
registry key.

One small helper executable and its private temporary folder remain in `%TEMP%`
after a normal removal until Windows or the user clears temporary files. The
uninstaller does not delete its own helper folder. This is a known, accepted
residue rather than a completed cleanup.

## Options

- `--silent`: install or uninstall without confirmation dialogs. Used by the
  isolated smoke check.
- `--uninstall <app-id>`: remove that application.
- `--remove-helper`, `--wait-pid`: internal removal-helper controls; they are
  not part of the user interface.
- Without `--silent`, Setup shows native confirmation and result dialogs.

## Not Chosen

- MSI and MSIX packaging are not used. The installer is a small,
  repository-built Go executable with no external installer toolchain.
- There is no installer wizard, elevation, runtime download, repair, or
  automatic update.
- The Setup executable is unsigned, so Windows SmartScreen and managed-device
  warnings can appear. The signing decision is unchanged and remains separate
  under ADR 0011 and ADR 0010.
- The current trailing-footer reader does not support a signed Setup: an
  Authenticode signature appends data after the footer. Signing the produced
  Setup needs a separate future payload design rather than signing the file
  as-is.

## Verification

The install/removal engine and the Setup payload have Go unit tests under
`internal/installer` and `internal/setuppayload`, and the full `velox_test` Go
suite plus `go vet` passed. An installer-enabled beta.3 release bundle was
built locally and the installer intents are now configured:

- `velox_installer_test`: ownership and native isolated-registry tests. Passed.
- `velox_installer_bundle`: build all three executables and an
  installer-enabled release ZIP. Passed.
- `velox_installer_smoke`: passed locally. It built with the unique app ID
  `dev.velox.installer-smoke-3876` (Setup SHA-256
  `46a9da4efc5289fca00207a89f35b0aad669ca49f365c944100f7b307d705ef3`),
  produced identical bytes across two builds, checked the installed tree plus
  the real Start Menu `.lnk` and registry entries, started the installed GUI
  executable and reached the two-render-frame JavaScript readiness marker, and
  ran the actual helper uninstall, which removed the install tree, Start Menu
  shortcut, and registry key while the disposable user test document remained.
  The script cleaned up its temporary helper. Evidence:
  `.cache/installer-smoke-3876/result.json`.
- `velox_file_notes_installer`: passed. It produced
  `dist/examples/file-notes/dev.velox.filenotes-setup.exe` (12,361,886 bytes,
  SHA-256
  `3d9d82c63cd4952995edaa74509a4db7182deb6e85cd2dfec2dbc38c50507368`), and
  the optional build result reports the Setup `file`, `bytes`, and `sha256`.

The final source passed `go vet`, and workflow YAML parsing passed. The
source-free consumer compilation boundary was also verified by the release CLI
smoke: the Setup path consumes the packaged application without a consumer Go or
C toolchain.

This is local harness evidence. It is not hosted CI, a push, a release, or a
manual end-user install. The first two smoke startup attempts timed out because
the harness initializer lacked a readiness callback and then because
`windowsHide` suppressed render frames; those were harness-only corrections, and
the real Setup install and uninstall ran and cleaned up on each failed attempt.
The normal-uninstall `%TEMP%` helper residue described above is unchanged.

## See Also

- ADR 0020: docs/adr/0020-optional-compiler-free-executable-branding.md
- Build contract: docs/cli/command-contract.md
- Release contents: docs/ops/release.md
