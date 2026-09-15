# File Permission Recovery

## Incident Evidence

The 2026-09-15 File Notes investigation observed NotAllowedError during Explorer
launches while tool-launched sessions could save. Click activation, focus,
secure context, and top-level document checks were true. The live WebView2 API
reported the expected Default profile, no InPrivate mode, and FileReadWrite
state Deny for the exact application origin. Disk preference inspection alone
had not revealed that effective state.

With the user's explicit approval, a one-shot diagnostic reset only this
origin/kind to Default. API completion succeeded and readback no longer listed
the denial. The user saved aa.md, then verified Save and Save as after the
ordinary executable was restored and relaunched from Explorer. This identifies
the immediate blocker, not who or what originally saved Deny. The ambient loader
and GUI-subsystem fixes alone had not resolved this incident.

The diagnostic and recovery executable was removed from the active app path.
The subsequent alpha.59 icon build was also visually confirmed by the user.
Private diagnostic data stays outside version control; no user document paths,
contents, or profile files are bundled with the product.

## Product Behavior

Development alpha.60 distinguishes canceled operations, denied file access,
security-context failures, and other errors without changing dirty state.
Development alpha.61 adds native window-menu **File access...** maintenance.
It uses the current WebView's actual profile and only the host-derived trusted
origin. No page-supplied origin, path, arbitrary permission kind, or Allow state
is accepted. No new public IPC method or manifest permission is added.

The read-only query offers a confirmation only for Deny; No is the default.
An accepted reset rechecks the denial, sets only FileReadWrite to Default, and
reads it back. Default and Allow are left alone. No files are written and no
automatic save is attempted. A new explicit picker/save gesture is still needed.
Policy restrictions or runtime failures may remain after a reset.

The two completion callbacks use the existing pinned COM owner and reference
counting. Only one operation per WebView is pending at a time. Shutdown releases
the profile reference and ignores late callbacks. Modal UI is queued onto the
native event loop rather than entered from a WebView2 COM callback.

## Validation

- File Notes: 14 unit tests, including denied picker, denied handle, cancellation,
  invalid context, retained drafts, and successful later writes.
- COM unit tests: allowed-origin restriction, Deny-only mutation, readback before
  completion, unchanged unrelated state, cancellation, callback lifetime, and
  QueryInterface ownership.
- Installed WebView2: an isolated profile retained Deny after restart, changed
  to Default after explicit reset, retained Default after a second restart, and
  left an unrelated origin denied. All three browser processes exited and the
  callback owner released every reference.
- The consumer-evidence workflow includes the same native recovery test.

### Installed Alpha.61 Manual Check

On 2026-09-15 the user supplied a screenshot of the ordinary File Notes build
with the Velox window icon, a document marked "Saved to file", and the native
File access dialog reporting that no stored block was found and no settings
were changed. This confirms the menu's read-only, no-change branch. The user
also confirmed that the installed icon was visible.

This does not prove manual acceptance or cancellation of a Deny-reset prompt.
Those branches remain separate from the automated temporary-profile API test
and the earlier user-approved one-shot recovery. The screenshot is not evidence
of an independently downloaded GitHub binary; public artifact verification
remains a separate release gate. No user document or profile is included here.

### Hosted Source Validation

Consumer evidence run
[34944803867](https://github.com/0disoft/velox/actions/runs/34944803867)
completed successfully for source
`7393e2fdf1bd0b5b3b74d04a1f0d5a350d453a94` (development alpha.61).
The workflow includes native startup/security checks and the isolated-profile
permission recovery test. Hosted workflow success does not establish a local
Explorer-launched Save as result for downloaded artifacts.

This record adds validation evidence only; it changes no runtime, public
contract, or release version. No release tag, beta promotion, or signing claim
is implied.

### Downloaded CI Artifact Check

The release artifact `10386792060` from that exact successful run was downloaded
on 2026-09-15. Its inner `velox-windows-x64.zip` SHA-256 is
`3132dec0e7e0ddb5a3f80dddd29706996518a0c473499c0a145c741836fe1969`.
The extracted host SHA-256 is
`6c710ff8c0676c5735c12087be095b7311be65c61dd7fdc7c7bccfd7c0cb30f0`.
These are CI bytes, not the previously installed local build.

The downloaded host passed GUI subsystem, both icon groups, native startup,
shutdown, and security-policy tests. Its CLI produced byte-identical File Notes
packages in two builds. The isolated app then completed two ready-exit launches
with the same test profile. The first verification helper had timed out while
launching a hidden window; rerunning with the existing asynchronous launch
pattern passed without changing product source. That timeout is not classified
as an established product defect.

The test application uses a distinct ID and leaves the installed app and its
profile untouched. Manual Explorer-launched Save as, restart, and Save checks
for this CI package are pending. Publication stays held until that check passes;
the consumer artifact is not itself a public release or publication provenance.
