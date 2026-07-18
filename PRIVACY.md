# Privacy

## Actutum Tooling

The Actutum CLI and native host do not send telemetry, crash reports, analytics,
or automatic update requests. The build path reads the project manifest,
static assets, and prebuilt host, then writes local build outputs.

## Local Runtime Data

The installed WebView2 Runtime can store cookies, cache, local storage,
IndexedDB, and other browser-profile data in an application-specific local user
data directory. Actutum does not upload that data. Removing a portable app does
not automatically remove its WebView2 profile.

## Packaged Applications

An application packaged with Actutum can implement its own network requests,
analytics, accounts, or data storage. Those flows belong to that application
and require its own privacy disclosure. They are not Actutum data collection.

## Maintainer Services

GitHub Actions processes source and build evidence for repository automation.
The unsigned developer-preview workflow does not contact a signing provider.
If ADR 0011 later reactivates SignPath, it will process only the two release
executables and signing metadata after provider onboarding is approved. These
maintainer services do not receive end-user application data from the Actutum
runtime.
