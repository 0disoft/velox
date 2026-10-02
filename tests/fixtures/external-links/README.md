# External links manual fixture

The bounded `velox_external_links_allowed` and `velox_external_links_denied`
intents launch the latest `dist/velox-host.exe` with copied fixture assets and a
unique profile under `.cache/external-links-manual/`. The launcher supplies
`web/velox.png` from the existing shared branding asset. These source fixtures
are not consumer templates and should be launched through the harness.

Neither page requests a link on load. Requests come only from the three buttons.
The granted fixture declares only `external.open`; the denied fixture declares
no permissions. `queued: true` acknowledges confirmation scheduling, never
successful browser launch.

In the allowed window, request HTTPS and cancel: no browser should open. Request
HTTPS again and approve: the default HTTPS handler should open the Velox GitHub
repository. HTTP and File must show `INVALID_PARAMS` without confirmation or a
browser launch. Close this test window. In the denied window, HTTPS must show
`PERMISSION_DENIED` without a prompt or browser; close that test window too.

The harness waits at most eight minutes. It terminates only the host process it
created if the deadline expires. It never terminates browsers or other Velox
apps, changes user profiles, or writes a document. The receipt records fixture
and host hashes, process exit, and timeout, not approval or page-load success.
Human observations must be recorded separately and must not be inferred from a
normal host exit. Existing File Notes data and release outputs remain untouched.
