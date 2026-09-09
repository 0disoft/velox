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

## Earlier Evaluation Gate

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

The results below predate the Hermes provider-client follow-up.

The focused workflow contract test, repository hygiene suite, and native sandbox regression passed.
The explicit loopback diagnostic failed as recorded above. The full product
suite was not repeated in the first preflight: that change added verification only and changed no
application binary, public IPC, database, or release version. Existing CI
workflows remain unchanged; a separate manual lifecycle workflow was added.
Beta and stable promotion remain held.

## Hermes Provider-Client Pipe Follow-up

The diagnostic now stages the installed Python 3.11 runtime, an explicit SDK
package list, and the installed Hermes `agent/process_bootstrap.py` and
`utils.py` helpers inside its temporary tool root. It loads the real helper's
`OpenAI` factory with an explicit `httpx.BaseTransport` backed by inherited
pipes. It does not construct `AIAgent`, start the Hermes CLI, or create a Hermes
session database. This is a provider-client integration probe, not a full
agent evaluation or a qualifying trial.

The first native attempt used temporary read/execute grants on the installed
runtime and SDK roots. Python exited before sending a request with Windows
status `0xc0000022` (access denied); cleanup revoked those grants. The final
implementation grants access only to the staged tool and private trial roots.
Copies are bounded to 200 MiB, reject symbolic links and nonregular files, and
exclude bytecode caches and tests. Python runs with `-I -B`; it does not load
user site packages or write installation bytecode. The staging path never
copies personal Hermes configuration, credentials, memories, or state.

The host independently admits exactly one canonical `POST /v1/chat/completions`
request for the selected Muse model and fixed probe prompt. The transport
limits request frames to 8 KiB and response frames to 64 KiB. The host owns
the localhost destination, forbids redirects, supplies no authorization header,
and bounds provider time to 30 seconds and output to 1,024 tokens. Child-supplied
headers, tools, other models, paths, prompts, streaming and token limits are
rejected. A second SDK call must fail locally without another provider request.
The sandbox has a 60-second process deadline; existing Job Object teardown
still applies. No listener, loopback exemption or global configuration change
is introduced.

The staged provider-client test and existing native boundary tests initially
passed in 17.035 seconds; the final rerun passed in 11.891 seconds. Related Go
version fixtures and hygiene checks passed, as did all 39 evaluation-tooling
tests. The full product suite and hosted lifecycle stress were not repeated
for this diagnostic-only change. Two initial live attempts at the earlier 128-token cap failed
the exact-output assertion; the second diagnostic identified `finish=length`,
zero content characters, and no tool calls. Increasing the fixed cap to 1,024
produced `OK` with `finish=stop` and no tool calls. That live test passed in
15.911 seconds including staging and cleanup. Three live requests were made
in this follow-up; none was a qualifying trial. Response content is not logged.
Response-declared Muse identity is checked, but independent routing and
no-fallback evidence remain absent.

The dedicated intents are `velox_hermes_pipe_test` (deterministic response, no
inference) and `velox_hermes_pipe_live` (one live request per invocation).
The regular Go suite skips the installed-runtime probe unless explicitly opted
in, while keeping the host request-policy tests active. Local development is
`0.5.10-alpha.42`; public release bytes remain alpha.40. Public IPC, database
schemas, normal evaluator execution, receipt schema and CI workflows are
unchanged. Full Hermes execution, session evidence, route attestation and
supervisor-crash coverage remain separate gates before the three-trial series.

## Current Direction: Native Cancellation Before Evaluator Work

Hermes-specific integration is paused by maintainer direction. Hermes is not
required to access opencodex models; it was the existing evaluation-record
adapter, not a model-provider requirement. The earlier pipe probes remain
historical diagnostics. No Hermes execution or model request was made in this
native cancellation follow-up. Whether Codex/opencodex records can satisfy the
existing session, tool-budget and containment evidence contract is not yet
verified; ordinary Codex execution must not be called a qualifying trial.

`TestNativeInitializationCancellation` uses the real installed WebView2 loader
and runtime through the repository's fork, a hidden window, and a fresh profile
for every case. It runs three pairs of:

- Cancellation inside the `environment-created` phase callback, before a
  controller is scheduled. The browser must remain destroyed, no controller
  completion may occur, callback roots must reach zero, and the profile must
  be removable.
- A queued message-loop exit after the environment callback, while controller
  creation is pending. A successful native controller completion must arrive
  after `Destroy`; the browser must not be revived. The test observes that
  browser process's handle signaling, callback roots reaching zero, and profile
  removal. It does not substitute a synthetic COM `Invoke`.

Two exploratory runs failed their expected late-environment ordering. Even
starting the native request directly and destroying before the subsequent
message pump did not make the environment callback late on this installation.
The final environment case therefore checks cancellation at actual delivery,
not a manufactured claim of late native delivery. Late-environment rejection
continues to have the existing synthetic unit coverage only; a real delayed
environment completion is not proven by this test.

The final six-case run passed in 2.011 seconds on Windows amd64 with WebView2
`152.0.4191.66` and Go `1.26.4`. All three controller cases observed browser
process exit; environment-completion cases do not claim a process-exit
observation because no controller was created.

The existing fork COM suite, related Go version/hygiene checks and all 39
offline evaluation-tooling tests passed. The full product suite and hosted
100-launch lifecycle run were not repeated for this test-only coverage change.

The new test is opt-in via `velox_native_cancellation_test`, with three pairs,
a 15-second Embed watchdog and bounded callback/process/profile waits. It logs
the installed WebView2, Go version and architecture. This is local source-fork
evidence, not a run against the public alpha ZIP or a hosted Windows runtime
matrix. The public lifecycle binding's `initializationCancellationTested: false`
remains unchanged. Hosted cancellation validation remains pending.

Only native test coverage, this record and local development version fixtures
changed. Local version is `0.5.10-alpha.43`; public alpha.40, public IPC, DB,
production runtime behavior and existing CI workflows are unchanged. No release
or beta promotion was performed.
