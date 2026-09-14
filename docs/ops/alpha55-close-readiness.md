# Alpha.55 Native Close Follow-Up

Issue #16 is separate from the alpha.54 controller HRESULT correction and
the packaging rollback fix merged as `c8e56e484313b3d92f1d317b82312da7a64894f2`.
Alpha.55 is a development candidate, not a public release or beta admission.

## Reproduction

The unmodified host at `c8e56e4` was built locally and exercised with a private
File Notes copy and profile. The fixture deliberately makes IndexedDB opening
fail, so draft persistence cannot conceal an unconditional native close.
CDP input established browser user activation; the fixture then dispatched an
editor input event containing only disposable test text.

Ordinary navigation produced a beforeunload dialog, and cancelling it preserved
the text. A native `WM_CLOSE` posted only to the probe process's named window
instead terminated the host without consent. Evidence:
`.cache/close16/1789384856965/result.json`.

Earlier diagnostic attempts were inconclusive: the first did not establish
editor contents, and a later attempt could not find the hidden probe through
`Process.MainWindowHandle`. The final fixture checks dirty text and activation,
and finds its own hidden window by process ID plus title. These tool failures
are not counted as product failures.

## Repair and Boundaries

User close now calls browser `window.close()`. Only the native
`WindowCloseRequested` callback schedules forced host teardown. A cancelled
dialog leaves the page alive. Initialization cleanup and explicit runtime close
use a distinct internal message and remain bounded by the existing owner.
The new callback shares the existing pinned COM owner, token removal, and
QueryInterface lifetime rules. No filesystem IPC or automatic save is added.

The repaired local host passed the same fixture, including navigation cancel,
native close cancel with exact text retention, and a second native close with
acceptance and host exit. Evidence: `.cache/close16/1789385039680/result.json`.
The maintained probe is `scripts/file-notes-close-smoke.mjs`.

## Final Local Verification

The alpha.55 host SHA-256 is
`82f39c78f2752223b8ae2ed728ec500cbfdd3654ca9459126fb63fc3a56a97a7`.
The maintained probe passed on WebView2 `152.0.4191.66`, including both cancel
paths, accepted native close and renaming its private profile directory.
Evidence: `.cache/close16/1789385415968/result.json`. That result's historical
`profileReleased` field means directory rename succeeded, not proof of browser
process exit; the maintained probe names it `profileRenameSucceeded`.

Focused host, CLI, runner, builder, build-plan, inspector and hygiene tests
passed, followed by all fork window and COM tests. The callback inventory now
covers twelve pinned interfaces; registration failure covers ten stages and
the close callback is tested for ignored late delivery after destruction.

Built-host startup and security-policy smoke passed in 21.54 seconds. This
separate smoke observes browser exit and profile release: the immediate launch
was 6.94 seconds, and browser exit lagged host exit by about 6.4 seconds. Those
delays remain known limitations; this change does not claim to improve them.

No hosted run or full evaluation campaign was repeated for this local change.
Public IPC, DB and permission policy are unchanged. The new native callback and
internal teardown message belong only to the runtime implementation.

This is real local WebView2 evidence with automated consent, not a human visual
check. `WM_CLOSE` covers the common native close handler; physical Alt+F4 input,
display scaling, and mixed-monitor movement are not separately verified here.
It does not claim persistence after forced termination or power loss.

The previous alpha.54 publication hold remains: mixed-DPI visual checks,
candidate-specific native save/restart confirmation and the known same-profile
relaunch delay are separate work. No tag, publication or beta promotion is made.
