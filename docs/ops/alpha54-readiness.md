# Alpha.54 Readiness Follow-Up

This record separates local source tests, visual observations and candidate
distribution checks. None implies beta admission or public release.

## 1. Initialization Cancellation

Source `b9d501f93b5e690a3528560c8d6b7154409c0b0e`, local Windows amd64,
Go `go1.26.4`, WebView2 `152.0.4191.66`.

One invocation of `TestNativeInitializationCancellation` passed all six cases
in 7.22 seconds: three environment-completion cancellations and three
controller-pending cancellations. All callback reference counts reached zero
and all private profiles were released. Late controller browser exits were
observed after 5,465 ms, 177 ms and 170 ms, with no process-wait errors.
Environment-completion cases do not claim browser-exit observation because
they do not schedule a controller.

The earlier intermittent failure did not reproduce. The slow first browser
exit and historical failure remain recorded; this passing run is not proof
that their cause was fixed. No runtime change or repeated run was made to
obtain a pass. Hosted verification is separate.

## 2. DPI and Visual Checks

The focused DPI/host, fork window/COM, version-fixture and hygiene checks
passed on alpha.54. These cover DPI setup and scaling behavior in tests;
they do not substitute for observed 125%/150% display scaling, text sharpness
or movement between differently scaled monitors.

The Windows UI helper could not obtain a targetable Display Settings window
after a launch attempt and refreshed window inventory. The user was asked to
open Display Settings. No display scaling was changed, no screenshot-based
sharpness claim is made, and the physical display matrix remains pending.
Candidate preparation may continue, but this pending check is retained in
the release decision rather than converted into a pass.

## 3. Candidate and Release Decision

Pending in this follow-up. Public alpha.51 remains the recorded public
preview. Alpha.54 and beta have not been published by this work.
