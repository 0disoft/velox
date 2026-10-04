# ADR 0037: Taskbar Progress

- Status: Accepted
- Date: 2026-10-04
- Owner: Runtime host

## Context and Decision

Allow one app-controlled taskbar progress indicator under independent,
default-disabled `window.progress`. Add only `window.setProgress` to IPC v1:
`none` and `indeterminate` accept only `state`; `normal`, `error` and
`paused` require an integer `value` from 0 to 100. Reject unknown fields,
states, fractions and null. Basic, title and attention permissions neither
grant progress nor are required by it. Return `null` when accepted, including
when cached before taskbar readiness; native HRESULTs remain internal.

## Native Lifetime

Use the existing UI/COM thread. Without the permission, install no progress
hook, handler or taskbar-message registration; create no progress COM object
and make no progress COM calls. Opted-in hosts install a narrow `BeforeShow`
hook in vendored `WindowOptions`, attaching the handler before show so the first
`TaskbarButtonCreated` is not missed.

Cache only the latest requested state until that real readiness message;
call no `ITaskbarList3` methods beforehand. Once ready, lazily create the COM
object and call `HrInit` only for a non-`none` request. Release a failed
initialization without calling progress methods. For determinate requests use
`SetProgressValue(value, 100)` before `SetProgressState`, so transitions from
indeterminate to error/paused retain the requested percentage. Deduplicate
successfully applied repeats.

Each `TaskbarButtonCreated` resets the applied-state cache, releases the old
COM object and restores the latest request with a newly initialized object
when needed. `WM_NCDESTROY` best-effort clears an existing ready indicator and
releases each owned interface exactly once, including reentrant destruction.
Cleanup must not create a COM object or call progress methods before readiness.

Microsoft documents the readiness boundary in
[ITaskbarList3](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nn-shobjidl_core-itaskbarlist3),
initialization in
[HrInit](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nf-shobjidl_core-itaskbarlist-hrinit),
and transitions in
[SetProgressValue](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nf-shobjidl_core-itaskbarlist3-setprogressvalue)
and [SetProgressState](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nf-shobjidl_core-itaskbarlist3-setprogressstate).

## Scope and Alternatives

Polling readiness or creating COM at startup is rejected: the shell message
supplies readiness and the opt-out path needs no progress machinery. A general
taskbar API, progress getter, notification or automatic application-operation
tracking is outside this decision. Apps supply progress and explicitly clear
it with `none`; retain an in-window indication because Windows can suppress
taskbar progress in high contrast or choose another grouped window's progress.

This adds one permission and method plus an opt-in lifetime handler, with no
activation, polling, timer, worker, new dependency, file I/O, DB/state-format
or version change. File Notes is unchanged. Native presentation remains
shell-controlled; acceptance does not establish visible progress.

## Verification and Rollback

Required coverage: strict state/value grammar, independent permission and
packaged propagation, pre-show installation, pre-ready latest-state caching
without native calls, lazy initialization/failure cleanup, value-before-state,
repeat deduplication, Explorer button recreation and exactly-once release under
reentry/destruction. Visible progress and Explorer recovery need separate
native evidence. No validation, size or operation-count result is claimed here.
Remove `window.progress` to disable the capability; no profile migration is
needed. Revisit if more taskbar methods or persisted progress are requested.

Synchronized surfaces: IPC v1, CLI configuration, product specification, ADR
index and validation requirements.
