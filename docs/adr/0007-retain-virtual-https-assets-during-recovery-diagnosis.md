# ADR 0007: Retain virtual HTTPS assets during recovery diagnosis

- Status: Accepted
- Date: 2026-07-15
- Owner: Project maintainer

## Context

Actutum maps its external asset directory to an application-specific virtual
HTTPS origin. The origin is part of the runtime security contract: navigation,
top-level message authorization, frame denial, and secure-context behavior all
depend on it.

Immediate same-profile relaunch evidence has shown long startup tails. A
normalized file URL control has not shown the same tail as consistently as
Actutum and a virtual-host mapping control. That observation narrows the search,
but it does not make file URLs security-equivalent and does not yet identify an
internal WebView2 timer or lock. User-data-folder ownership, browser-process
lifetime, controller creation, origin state, and machine-wide Runtime state can
overlap.

The public benchmark repository now owns a manual recovery experiment that
varies the relaunch delay, UDF, and synthetic virtual hostname while recording
host and browser process IDs plus startup and shutdown phase timelines. Local
smoke results are diagnostic only. Ten complete hosted samples in one pinned
runner and WebView2 environment remain required for a publishable boundary.

## Decision

Actutum retains application-specific virtual HTTPS folder mapping as the only
production asset transport during M3.

- File URL loading remains a benchmark control, not a runtime option or
  fallback.
- Actutum does not add a fixed relaunch sleep, delete or rotate a user's UDF, or
  change the application origin to hide an immediate-relaunch tail.
- Settled warm-start evidence continues to wait for browser exit and profile
  release. Immediate same-profile relaunch remains a separate lifecycle
  diagnostic and is not a startup marketing number.
- A transport change requires a superseding ADR. It must preserve the current
  origin, frame, navigation, bridge, and secure-context contracts or define and
  test an explicit replacement.

## Alternatives

### Use file URLs for production assets

Rejected for now. It removes or changes origin and secure-context assumptions
that currently bound the native bridge. A faster synthetic control is not
enough evidence to accept that security and compatibility change.

### Sleep before every relaunch

Rejected. A fixed delay would move an observed tail out of the measured window
without fixing ownership, and any assumed timer could change across WebView2
Runtime releases.

### Create a fresh UDF on every launch

Rejected. It would discard the stable application profile contract and split
cookies, IndexedDB, local storage, and cache ownership across launches.

### Rotate the virtual hostname on relaunch

Rejected. It changes application origin identity and storage semantics. The
synthetic fresh-origin scenario exists only to classify the current tail.

## Consequences

### Positive

- The current deny-by-default origin and bridge contract stays intact.
- Performance work remains evidence-driven instead of adding a hidden delay or
  destructive profile workaround.
- Immediate relaunch, settled warm start, browser exit, and profile release
  remain distinct measurements.

### Negative

- Some same-profile immediate relaunches may retain a multi-second startup
  tail while diagnosis continues.
- Actutum cannot claim that its startup path is uniformly faster than another
  wrapper from local or incomplete recovery evidence.
- The benchmark suite spends additional hosted runner time to isolate a
  lifecycle issue that may ultimately be owned by WebView2.

## Validation

A superseding transport decision requires all of the following:

1. ten complete recovery samples for every declared delay and isolation
   scenario in one pinned hosted environment;
2. raw process and phase evidence that localizes the delayed interval without
   discarding failures or right-censored browser exits;
3. origin, frame, navigation, popup, download, permission, and IPC negative
   tests for the proposed transport;
4. fresh, settled warm, and immediate-relaunch startup comparisons using the
   same ready boundary; and
5. a documented migration and rollback path for existing application data.

## Rollback or Fallback

There is no runtime migration in this decision. If virtual HTTPS mapping becomes
unusable, a superseding ADR may introduce a separately named transport profile.
The existing profile remains the compatibility fallback until its security and
data contracts can be preserved or deliberately versioned.

## Revisit Triggers

- Publishable hosted recovery evidence attributes the tail to virtual-host
  mapping rather than UDF or process ownership.
- A supported WebView2 Runtime release changes the measured boundary.
- Virtual HTTPS mapping blocks a required platform capability.
- A security review finds that the current mapping cannot enforce the declared
  origin or frame contract.

## Synchronized Surfaces

- `docs/product/02-spec.md`
- `docs/product/03-risk-register.md`
- `docs/engineering/03-performance-budget.md`
- `docs/engineering/04-security-baseline.md`
- `docs/adr/README.md`
