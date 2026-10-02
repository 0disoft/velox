# ADR 0022: Opt-in single-instance host

- Status: Accepted
- Date: 2026-10-02
- Owner: Project maintainer
- Amends: ADR 0017 (narrow host-to-host lifecycle amendment)

## Context

ADR 0017 kept the host's native surface closed to basic window lifecycle and
required a new product and threat-model ADR before any wider native capability.
ADR 0021 amended that boundary for opt-in window-state persistence only.

Launching a packaged Velox application twice creates two independent host
processes with two WebView2 windows over the same application profile. For a
single-window desktop app that is usually the wrong result: the user expects
the already-running window to come forward, not a second copy. Windows already
provides an atomic, kernel-owned primitive for this in the named mutex, which
is scoped to the user and session and closed by the OS on process exit.

This ADR approves a narrow, opt-in single-instance launch behavior. It does not
approve a broader native API, argument or URL delivery, a local server, or any
application-visible native surface.

## Decision

Add one optional manifest field, `app.singleInstance`.

- Type: boolean. Default `false`; omitted values behave as `false`.
- When `false` or omitted, the launch path is unchanged: the host acquires no
  mutex, queries no activation window, and installs no subclass.

When `app.singleInstance` is `true`, the host acquires one named mutex before
opening WebView2. The mutex identity combines:

- the current user SID;
- the Windows local session;
- `app.id`;
- the canonical absolute application profile path.

Profile canonicalization resolves the existing ancestor with
`GetFinalPathNameByHandle` and then appends the missing path tail, so 8.3
short names, junctions, and case differences that alias the same location share
one identity. Different application, profile, user, or session values produce
different identities and run independently. Host version and `app.version`
are not part of the identity, so updating the application keeps the same
identity.

A duplicate launch that opens an already-existing mutex becomes a secondary instance:

- it creates no additional WebView and no window;
- it locates the primary window through a window property;
- it posts a payload-less registered Windows message asking the primary to
  restore from minimized and request foreground;
- it exits `0`.

If the primary is still initializing, or has published no window yet, the
duplicate is still suppressed (`0` exit) but no activation is attempted. There
is no activation queue, timer, or poll wait.

`SetForegroundWindow` is subject to the Windows foreground policy, so focus is
not guaranteed. When the OS refuses, a bounded taskbar flash is allowed as the
fallback.

The mutex uses the default DACL and a non-inherited handle. The OS closes the
handle on normal or forced termination, so there is no lock file to clean up.

It adds no file, URL, or argument delivery; no socket or named-pipe server; no
background goroutine; no new dependency; and no public web IPC method.

On a mutex or attach failure that cannot be resolved, the host fails startup
with exit `6` instead of silently launching a duplicate.

### Scope and security posture

Single-instance is a convenience mechanism, not a security or authentication
boundary. A same-user malicious process can precreate the mutex or send the
activation message. This decision does not attempt to defend against that, and
the documentation must not claim it does.

`app.singleInstance: false` does not prevent multiple same-profile WebView2
instances from existing; that remains the current behavior. The known
same-profile immediate-relaunch delay is unrelated to this decision.

### Amendment to ADR 0017

ADR 0017 required a new product and threat-model ADR before any native
capability beyond basic window lifecycle. This ADR is that review, and it is
deliberately narrow: it approves opt-in single-instance host coordination only.
The application-specific backend, native filesystem, shell, process, sidecar,
plugin, local-server, updater, asset-sealing, new-platform, and broad IPC
prohibitions in ADR 0017 remain in force and are unchanged.

## Alternatives

### Track instances with a lock file

Rejected. A lock file needs stale-lock detection and cleanup plus its own
races, while a named mutex is atomic and closed by the OS on process exit.

### Use a named pipe or local socket for activation

Rejected. It adds a server, a listening surface, and a message protocol where a
registered window message and a window property are sufficient.

### Deliver the new command line, arguments, or URL to the primary

Rejected. It expands the surface to payload parsing and argument injection and
is not needed to bring the existing window forward.

### Queue or poll until the primary publishes its window

Rejected. It adds a wait loop and a timer to the secondary launch. Suppressing
the duplicate without activation is simpler and bounded.

### Make single-instance the default

Rejected. It would change the current multi-instance behavior and add a mutex
and an activation window to every launch. The field stays opt-in.

### Use the mutex as a security boundary

Rejected. A same-user process can precreate the mutex or spoof the activation
message, so it cannot enforce trust. It stays a convenience mechanism.

## Consequences

### Positive

- A second launch reuses the visible primary window instead of creating a
  competing window.
- The default path adds no mutex, no query, and no subclass.
- The mechanism needs no new dependency, server, socket, or background process.
- The identity is per application, profile, user, and session, so unrelated
  applications stay independent.

### Negative

- Foreground focus is best-effort; the OS may allow only a taskbar flash.
- A duplicate during initialization is suppressed without activation.
- Correct grouping depends on profile canonicalization; an unusual profile
  path could misgroup or split instances.
- It is not a security boundary against same-user processes.
- One more opt-in field widens the manifest surface and must stay synchronized.

## Validation

An accepted decision is not proof that tests pass. These checks are the
acceptance plan; none are claimed as passed here, and no measured comparison is
claimed until the parent supplies measurements.

- Schema and parser accept `app.singleInstance` as an optional boolean with a
  `false` default, and reject a non-boolean value.
- With the field absent or `false`, no mutex is acquired, no activation window
  is queried, and no subclass is installed.
- A second launch with the same identity creates no second WebView or window
  and exits `0`.
- Different application, profile, user, or session values run independently.
- Changing the host version or `app.version` keeps the same identity.
- A minimized primary is restored and asked to foreground with a payload-less
  message; an OS-refused foreground falls back to a bounded taskbar flash.
- A mutex or attach failure exits `6` instead of silently duplicating.
- No socket, named-pipe server, payload delivery, or background goroutine is
  introduced.

The File Notes example source opts in with `app.singleInstance: true`, but this
ADR claims no manually rebuilt executable, push, or release.

## Rollback or Fallback

- Removing `singleInstance` or setting it to `false` restores the current
  launch behavior; no migration is needed.
- The mutex is released by the OS on process exit, so there is no file to clean
  up.
- If activation proves unreliable, keep the mutex suppression and drop the
  foreground request, or return to the default multi-instance behavior.

## Revisit Triggers

- A broader native capability, backend, or IPC method is proposed.
- Argument, URL, or file delivery to the primary instance is proposed.
- Single-instance is claimed as a security or authentication boundary.
- Single-instance becomes the default, or applies to non-Windows targets.
- A measured startup or activation claim is requested.

## Synchronized Surfaces

- `docs/adr/README.md` (ADR table for 0017 and 0022).
- `docs/cli/configuration.md` (`app.singleInstance` field).
- `docs/product/02-spec.md` (runtime single-instance behavior).
- `schema/velox-v1.schema.json` (manifest `app.singleInstance`).
- `schema/runtime-config-v1.schema.json` and
  `internal/runtimeconfig/config.go` (runtime-config parse and host
  passthrough).
- `internal/singleinstance/` and `cmd/velox-host/main.go` (Windows lifecycle).
