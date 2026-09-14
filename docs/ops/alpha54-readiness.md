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

Pending in this follow-up. Unit checks do not substitute for observed
125%/150% display scaling or movement between differently scaled monitors.

## 3. Candidate and Release Decision

Pending in this follow-up. Public alpha.51 remains the recorded public
preview. Alpha.54 and beta have not been published by this work.
