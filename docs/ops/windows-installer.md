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

Source-only follow-up (2026-10-11): extraction now returns that completed
inspection along with the directory. Setup reuses its app name/version for
confirmation and success messages instead of immediately reading and hashing
the same tree again. This removes one full UI-only inspection. The returned
metadata is not installation authorization: `Install` still re-inspects the
source at entry, scans it, compares copied sizes/hashes, and verifies the final
staging tree. Payload checksum/ZIP budgets and copy checks are unchanged.
Public alpha.68 and existing Setup executables do not contain this change.

Targeted payload/installer tests cover the returned complete metadata, empty
results on failed tree inspection, refusal after a prior display inspection
becomes stale, and staging refusal after source mutation following install
entry inspection. Existing Windows shortcut/registry/removal and interruption
tests passed in private test locations; scoped vet and a stripped Windows GUI
Setup build passed. Mutation during the copy stream itself was not newly
injected, and actual Setup confirmation, full distribution smoke and timing/
size comparisons were not repeated. No measured installation speedup is claimed.

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
  `UninstallString`, `VeloxInstaller`, `VeloxInstallIdentity`, and
  `NoModify`/`NoRepair`. The entry
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

Source-only registry follow-up (2026-10-12): the ownership schema and target
directory are now written together as the `VeloxInstallIdentity` registry value
before any display metadata. Metadata order is deterministic and `DisplayName`
is written last. Removal recognizes that complete identity even when later
values are missing, while still refusing changed owner/location values,
malformed or mismatched identities, and unowned subkeys. A key with no values or
subkeys is removable after the installation's file record is authenticated:
there is no registry data to erase at the CreateKey/first-value boundary.

Legacy registrations without this new value still require both their original
owner and location fields. Already damaged nonempty legacy registrations are
not automatically reset. Native private-registry regression cases reproduced
the empty-key refusal before the change, and now cover recognized partial
identity, legacy compatibility and preservation on foreign/changed ownership.
These injected partial states are not power-loss or actual Setup UI evidence.
The application API, permissions and `velox.install/v1` file-record schema are
unchanged; the installer adds one registry value. Existing installed applications
and public Setup assets have not been replaced.

The follow-up native process test stops an actual child with `os.Exit` at ten
boundaries: after empty key creation and after each of the nine production
value writes. That bypasses Go defers instead of simulating normal error-return
cleanup. The already-promoted, authenticated install record and the real
HKCU test key are then passed to the normal removal engine. Every boundary
passed removal, fresh reinstall with version 2.0.0, matching Apps registry
version, final removal and preservation of a separate synthetic draft file.
The test uses private `Software\VeloxInstallerTests` keys and temporary native
shortcuts, not the user's Apps entries. Its host/uninstaller files are installer
test fixtures; no consumer application or Setup UI is launched.

`go test ./internal/installer ./internal/setuppayload ./cmd/velox-setup`, scoped
vet, diff checks and a stripped Windows GUI Setup build passed. The Setup
command currently has no own test files, so that package step is compilation,
not new command/UI coverage. These controlled post-call process exits do not
verify interruption inside a registry system call, power-loss durability,
whole-Setup termination, installer confirmation dialogs or real GUI application
upgrade. No full CI, public release or existing installation was changed.

Source-only publication rollback follow-up (2026-10-12): a failed publication
no longer recursively deletes the promoted target. It reuses the owned-file
removal checks, first requiring the on-disk record to match the record committed
by this install. Changed or additional files and altered ownership stop rollback
before file deletion; the error identifies the preserved installation and the
conflict to resolve before removal/retry. Ordinary unchanged installs still roll
back completely. Integration cleanup stays with the publisher, not ordinary
unregister, so a foreign registry entry or shortcut appearing during publication
is not removed as part of filesystem rollback.

Three regression fixtures reproduced loss of added root/nested notes and an
edited web asset before the change. Afterward their bytes and retry record are
preserved, and resolving the conflict permits removal and reinstallation.
An altered ownership-record case also remains intact. Two real private-registry
and native-shortcut publication races verify foreign integrations are preserved
while unchanged owned files roll back. These are deterministic failure-boundary
tests, not concurrent-writer stress or atomic deletion guarantees. A file can
still change after the existing preflight, and the final directory-removal
record ordering is a separate pending check. No API, permissions, record schema,
dependency, version, CI runner or installed app changed.

Source-only late-removal retry follow-up (2026-10-12): empty child directories
are removed while the ownership record is still present. Only afterward is the
record removed before the final root removal. If that final removal returns an
error, a flushed temporary record is restored by a no-overwrite hard link. A
file recreated at the record path is never overwritten. Ownership is re-read
and must match before the final record deletion. Both normal removal and guarded
publication rollback use this ordering.

Before the change, an added file after file preflight and real Windows locks on
the child/root directory each reproduced loss of the retry record. They now
preserve or restore it, and the engine removes the installation on retry after
the conflicting file/lock is resolved. Separate tests preserve late changes to
ownership and verify restoration refuses a new file at the record path without
leaving temporary state files.

This is error-return retry evidence, not atomic removal or forced-process-exit
recovery: the interval between record deletion and final root deletion remains
unprotected against a process stop. Hard-link restoration can fail on an
unsupported filesystem; its error is returned together with the removal error.
The temporary removal helper remains available after ordinary failure, but the
installed `uninstall.exe` and Apps registration may already be gone, so complete
end-user retry through Setup/helper UI still requires a separate check. No power
loss, full Setup UI, concurrent file-replacement stress or new public artifact
was verified here.

Source-only helper retry follow-up (2026-10-12): an interactive removal helper
now offers Retry/Cancel after a removal failure instead of immediately ending
with an OK-only error. Retry invokes the same removal engine in the still-running
temporary helper. Cancellation emits the failure once without a second error
dialog. Silent failure remains exit 6 and includes a quoted command invoking
that helper with `--uninstall <app-id>`, which works even if the installed
uninstaller has already been deleted. Success behavior is unchanged.

The real command entry-point child test reproduced the missing retry command
before the change and passes afterward; its unique app identity is verified
absent before execution, so it changes no installation. Decision tests cover
success, silent refusal without prompting, cancellation, one retry and multiple
retries, preserving the original error cause. These tests exercise callback
decisions and the silent command path, not a manually clicked MessageBox or an
actual installed application's helper removal. No background retry, new flag,
application API, permission, dependency, version or CI runner was added.

Source-only final-removal journal (2026-10-12): before deleting the internal
ownership record, the engine publishes a completed, flushed snapshot at
`%LOCALAPPDATA%\Programs\Velox\.velox-removal-<app-id>.json`. Its contents
use the existing `velox.install/v1` record and still bind the same app identity,
target directory and file paths. Publication refuses to overwrite any existing
entry; an existing journal is reusable only when its validated record matches.

If the internal record is missing, removal can resume from this sibling snapshot.
If the target directory is already gone, it finishes the guarded integration
and journal cleanup. A conflicting or unreadable journal is preserved and blocks
removal before file deletion. New installation also refuses a pending journal
instead of replacing the identity while old removal is unresolved. Successful
removal deletes its journal; returned root-deletion errors keep the journal as
well as attempting the earlier no-overwrite internal-record restoration.

Regressions reproduced missing-record recovery failure and bypass of conflicting
metadata before the change. Actual child exits without Go defers passed at three
boundaries: journal committed, internal record deleted and root deleted. Each
resumed engine removal and allowed fresh installation afterward. A new user file
added after interruption was retained and blocked removal until moved; an outside
synthetic draft was byte-preserved. The final three-cut run took 0.52 s.

These are authenticated installer fixtures and the production final-deletion
routine, not whole Setup termination, a consumer host launch or power-loss proof.
The snapshot file is a new private installer storage artifact; no record schema,
public application API, permission, dependency, version or CI runner changed.
Old interrupted installations lacking a valid internal record and journal are
not automatically deleted. File replacement between preflight and removal and
manual native retry-dialog interaction remain separate checks. Related package
tests, scoped vet and a stripped GUI Setup build passed; nothing was published.

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
