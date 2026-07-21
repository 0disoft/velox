# Runtime Flow

- Status: Draft
- Owner: Project maintainer

## Build Flow

1. Resolve the project root and explicit or default manifest path.
2. Parse the supported manifest schema.
3. Normalize defaults and validate semantic constraints.
4. Walk and validate the static asset tree.
5. Select the pinned Windows x64 host template.
6. Create an immutable build plan.
7. Create a sibling staging directory owned by the current run.
8. Copy the unchanged host, runtime configuration, and assets.
9. Write the build-result document.
10. Create the deterministic ZIP.
11. Verify planned output counts and digests.
12. Atomically promote completed output.

### Build Failure

- Parsing and validation fail before output mutation.
- Copy and archive failures remove only the current staging directory.
- Previous successful output and project source remain untouched.
- Errors return a stable exit code and structured diagnostic.
- Cancellation follows the same cleanup path.

## Host Startup Flow

1. The operating system starts the generic host.
2. The host resolves and validates `velox.runtime.json` beside its own
   executable, independent of the process working directory. An explicit
   `--config` path remains available to the source-run path.
3. The host initializes its Windows UI thread and COM apartment.
4. The host creates the native window without declaring the application ready.
5. The host creates the WebView2 environment and controller.
6. Security settings and browser permission handlers are registered.
7. The asset root is mapped to the expected virtual HTTPS origin.
8. The frozen JavaScript bridge is injected for the top-level document.
9. The host navigates to the configured entry point.
10. The page reaches DOMContentLoaded and two animation frames.
11. A benchmark build may emit the benchmark-only ready marker.

### Startup Failure

- Missing or unsupported WebView2 produces a local actionable error.
- Invalid runtime configuration or contract versions fail before navigation.
- An invalid asset root or entry point fails before WebView creation when
  possible.
- Remote or untrusted navigation is canceled.
- No missing dependency is downloaded automatically.

## IPC Flow

1. Application JavaScript invokes a method through the frozen bridge.
2. The bridge allocates a bounded request identifier.
3. The bridge posts a versioned JSON request.
4. The host checks source origin, frame ownership, payload limits, protocol
   version, method existence, and permission.
5. The fixed dispatcher performs the native operation.
6. The host returns exactly one success or stable error response.
7. The bridge resolves or rejects the matching Promise and releases state.

Unknown response identifiers, duplicate completions, malformed messages, and
requests above limits fail safely without invoking native behavior.

## Shutdown Flow

1. Application or operating-system close begins one idempotent shutdown.
2. New IPC requests are rejected.
3. Pending requests complete with a stable shutdown error.
4. WebView2 event handlers and callbacks are detached.
5. Controller and environment references are released on the owning thread.
6. The native window closes and the process exits.

The host does not upload crash data or preserve a background process.

## Data Flow Boundary

Velox processes local configuration, assets, build output, WebView2 messages,
and local diagnostics. Application network requests and business data do not
flow through a Velox service.
