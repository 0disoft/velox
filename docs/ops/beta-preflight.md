# Beta Preflight: 2026-09-09

- Status: Public lifecycle repetition passed; qualifying LLM series held.
- Release under test: unsigned `v0.5.10-alpha.40`.
- Scope: bounded native lifecycle evidence and local provider reachability,
  not beta promotion or human adoption evidence.

## Public Windows Lifecycle

[Hosted run 34324755280](https://github.com/0disoft/velox/actions/runs/34324755280)
passed on measurement revision `572d8386eaf5978fb8fba0a060f603ec15ce427f`.
The runner was Windows Server 2025, image `windows-2025-vs2026`, version
`20260824.214.3`. Go 1.26.7 compiled the measurement tests; the application host
was downloaded from the public release, not rebuilt from the checkout.

The downloaded release ZIP matched SHA-256
`771173b6eec2f74d92228e7ac5b52332160b0f9fb4d7baecf01976864a00f8c8`.
The workflow verified the release manifest version and target and recorded the
downloaded host digest separately from the measurement commit.

`TestStartupLifecycleEvidence` passed 50 fresh-profile/immediate-relaunch pairs,
100 host launches, in 557.28 seconds. The workflow required all 50 sample
outcomes to succeed and validated the v3 evidence schema and run identity.
The test checked host exit, browser-process handle signaling, and profile
directory removal within their existing timeouts.

[Artifact 10093624895](https://github.com/0disoft/velox/actions/runs/34324755280/artifacts/10093624895)
contains `release-binding.json` and `lifecycle.json`. Its ZIP digest is
`41f72c752f11c7834ac1c84522edc9f73f812a6f3127ca47c46ef099800da2b1`.
Retention is 90 days; this is not permanent public evaluation evidence.

This does not test initialization cancellation, prove absence of all native
memory leaks, or certify a broader WebView2 runtime matrix. The binding records
`initializationCancellationTested: false`; those claims remain unverified.

## opencodex Evaluation Route

The selected sequence remains Muse / DeepSeek / Muse through the existing
opencodex installation:

- `commandcode/meta-muse-spark-1.3-contributor`
- `opencode-go/deepseek-v4-flash-vision-exp`

Grok 4.6 and Gemini 3.8 Flash are alternatives to choose before a series starts,
not silent fallback targets. No additional provider purchase is required by
this plan. This preflight made no inference request and proves neither model
availability nor actual provider/model identity at inference time.

The native containment regression passed locally in 0.22 seconds. The opt-in
`velox_beta_opencodex_probe` then connected to `127.0.0.1:10100` outside the
AppContainer, but the same TCP connection inside the AppContainer failed with
a connection timeout. Filesystem denial, allowed child-process behavior,
state export, and sandbox cleanup still passed before the diagnostic failed.

This isolates the current failure to sandbox-to-local-proxy reachability, not
credentials, model choice, or an inference response. No loopback exemption,
firewall change, global Hermes configuration, or provider routing change was
applied. No qualifying trial was started or replaced.

## Next Gate

Provide an explicitly scoped evaluator-to-opencodex transport while preserving
filesystem and process containment, and test its teardown and actual model
identity before starting the three-trial series. Do not expose the shared
proxy publicly or run an uncontained evaluator and label it qualifying.
Initialization-cancellation stress remains a separate native test requirement.

## Inherited-Pipe Prototype

The follow-up rejected the Windows loopback-exemption approach. The standard
API configures a list of AppContainer SIDs, not a single permitted local API
path or port, and this host process is not elevated. No exemption was applied.
See [Microsoft's API contract](https://learn.microsoft.com/en-us/windows/win32/api/netfw/nf-netfw-networkisolationsetappcontainerconfig).

A native prototype instead inherits exactly two pipe handles using an explicit
`PROC_THREAD_ATTRIBUTE_HANDLE_LIST`. Non-pipe handles are rejected. The child
cannot select a URL, model, headers, or prompt: the probe broker accepts only
one fixed operation. The host forwards that operation to the existing local
opencodex endpoint. No credential is supplied to the child and no listener,
firewall rule, provider configuration, or loopback exemption is created.

Native allowed-request, denied-request, non-pipe rejection, timeout termination,
pipe closure, outside-file denial, and unchanged default-containment tests
passed in 1.771 seconds. One live Muse request then passed; the full live probe
including deny and timeout cases took 4.624 seconds. The response declared a
Muse Spark 1.3 model and returned `OK`. This is connectivity evidence, not an
independent upstream routing or no-fallback attestation.

The first prototype run failed before process launch because the test omitted
the sandbox's normal private environment. Reusing `prepareEnvironment` fixed
that test setup; it did not change the isolation policy.

Normal `Run` still inherits no handles. The pipe route is not exposed by the
evaluation CLI and emits no qualifying receipt. Hermes HTTP adaptation,
bounded general evaluation requests, model-route attestation, and supervisor
crash coverage remain prerequisites for the three-trial series. Local source
advances to `0.5.10-alpha.41`; the public release and evaluated application
bytes remain `v0.5.10-alpha.40`. No new release was published.

After the local version change, the related Go package and hygiene checks
passed, as did all 39 evaluation-tooling tests. The full product suite and
hosted lifecycle run were not repeated for this transport prototype.

## Earlier Verification Scope

The focused workflow contract test, repository hygiene suite, and native sandbox regression passed.
The explicit loopback diagnostic failed as recorded above. The full product
suite was not repeated in the first preflight: that change added verification only and changed no
application binary, public IPC, database, or release version. Existing CI
workflows remain unchanged; a separate manual lifecycle workflow was added.
Beta and stable promotion remain held.
