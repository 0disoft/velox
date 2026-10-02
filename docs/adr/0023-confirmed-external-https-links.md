# ADR 0023: Confirmed external HTTPS links

- Status: Accepted
- Date: 2026-10-02
- Owner: Project maintainer
- Amends: ADR 0017 for one explicit native capability only

## Context

The maintainer approved optional desktop conveniences that preserve the
compiler-free consumer build and avoid background services or heavyweight
dependencies. Opening help or project links externally is useful without
requiring native file APIs or a generic process backend. The existing WebView
navigation and popup policies must not become automatic external launch paths.

## Decision

Add the opt-in permission and IPC method `external.open`. The IPC envelope
remains version 1; the closed method table grows from six to seven methods.
Existing permissions and default-empty permission lists are unchanged.

The method accepts only `{url: string}` and validates before scheduling work.
Only absolute HTTPS URIs up to 2,048 ASCII bytes are supported. Host names must
be ASCII DNS/punycode, IPv4-style names, or valid IPv6 literals without zones;
ports must be in 1-65535. International path/query text must be percent encoded.
Reject credentials, malformed escapes, control characters, backslashes, raw
whitespace, and other schemes. This is a conservative parser boundary, not a
safe-site policy or a destination allowlist.

The source remains the expected top-level application's WebView message; the
native dispatcher independently enforces permission, request validation,
shutdown, and limits. The transport cannot attest a browser click. Therefore
the host requires explicit Yes/No confirmation for each request, defaulting
to No. One request may be pending or confirming per host; concurrent requests
fail with `TOO_MANY_REQUESTS`.

Return `{queued:true}` once the UI work has been scheduled. It does not claim
approval, browser startup, navigation success, or a loaded remote page. Show
the modal confirmation only after the WebView event handler returns, following
the [WebView2 threading model](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/threading-model).
Cancel does nothing. Pending work is skipped on shutdown; if accepted work
fails, show a generic native error without reporting the URL to logs or IPC.
There is no additional completion event in this first version.

After approval use [ShellExecuteW](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shellexecutew)
on the UI/COM thread with a fixed `open` verb, validated URI as the object,
no arguments, and no supplied working directory. The OS-registered HTTPS
handler controls what runs. Velox does not fetch the URL or control remote
redirects, external browser cookies, or handler registration. Host confirmation
is not protection against a local attacker who can change installed assets or
the user's protocol handlers.

No generic shell method, executable path, command arguments, custom protocol,
file delivery, domain allowlist, updater, backend, socket, timer, background
worker, or dependency is introduced. No example's existing permissions are
silently expanded. Ordinary navigation/popups remain denied.

## Alternatives

- Launch directly from the binding: rejected because it provides no native
  evidence of human approval and modal UI inside WebView callbacks risks
  unsupported reentrancy.
- Trust a JavaScript user-activation boolean: rejected because application code
  can forge it or invoke the internal binding directly.
- Automatically open blocked navigation or popups: rejected because it widens
  unrelated behavior and permits unexpected browser launches.
- Add a generic shell or process API: rejected because it exposes substantially
  more authority than the approved HTTPS-only feature.
- Add an asynchronous IPC completion protocol now: deferred. A queued response
  and host-owned result UI cover this first slice without a new message path.

## Validation

Focused tests cover accepted/rejected URIs, permission denial, strict params,
normalization, queue bounds, cancellation, shutdown suppression, failure
redaction, and manifest-to-runtime/build-result permission propagation.
Native no-owner tests must return without displaying UI. Automatic tests use
fake confirmation and launch callbacks, never the real browser or network.
Real confirmation and default-handler launch are a separate manual check;
passing unit tests does not claim that check completed.

Measure host executable growth after the same release-style build. Existing
startup/security smoke remains applicable. No unchanged-performance claim is
made solely from code inspection or a single startup sample.

## Consequences and Rollback

Apps can explicitly open HTTPS links while the default runtime remains denied.
Native confirmation adds an extra interaction; long URLs can be cumbersome in
the standard Windows dialog. ASCII-only URI syntax requires international URL
normalization by the caller, for example `new URL(value).href`.

Remove `external.open` from permissions to disable the capability without data
migration. Revisit asynchronous completion, less intrusive consent, or domain
policies only when an actual consuming app needs them. Any broader capability
requires its own scope and threat-model decision.
