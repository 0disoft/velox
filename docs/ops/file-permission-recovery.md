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

The historical user confirmation proves the one-shot recovery, not manual use
of the new native menu. Human menu-confirmation/cancel and real-file save checks
for the new build remain distinct from automated API evidence. No release tag,
beta promotion, or signing claim is implied.
