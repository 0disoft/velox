# Command Contract

- Status: Draft
- Repository Type: cli-tool
- Owner: Project maintainer

## Interface Principles

- Commands are non-interactive by default in CI.
- Human output is concise; JSON output is stable and versioned.
- Build commands do not access the network.
- A failure never reports success because a feature is unavailable.
- Paths in diagnostics are project-relative when possible.
- Output never includes source file contents, secrets, or environment dumps.

## Implementation Status

`init`, `validate`, `doctor`, `run`, `build`, `inspect`, and `version` are
implemented in the M1 vertical slice. The
release bundle must place the unchanged prebuilt `actutum-host.exe` and its
`actutum-host.json` beside the CLI. The CLI verifies release, target, contract,
size, and digest agreement; there is intentionally no public flag that
substitutes an arbitrary host.

## MVP Commands

### actutum init [directory]

Create a minimal manifest and dependency-free static web example.

- Derive a conservative `dev.actutum.<directory>` application ID and display name
  from the target directory.
- Preflight every planned path and refuse the operation if any generated file
  already exists.
- Remove only files and directories created by the failed invocation.
- Do not install frontend dependencies.
- Do not download a host or runtime.

### actutum validate

Validate manifest syntax and semantics, asset paths, entry point, permissions,
security policy, target support, and host compatibility without creating
output.

### actutum doctor

Report local prerequisites and compatibility, including Windows architecture
and build, WebView2 Runtime availability and version, project configuration,
and bundled host
compatibility. Doctor is read-only.

- Query the installed runtime through the same bundled WebView2 loader used by
  the host instead of inferring availability from registry paths.
- Report platform, Windows build, runtime, project, and host checks in stable
  order.
- Keep the complete check result in JSON on failure while returning the
  corresponding non-zero prerequisite, project, or host exit code.
- Require Windows 10 version 1709 x64 or Windows Server 2016 x64, and Evergreen
  WebView2 Runtime `92.0.902.49` or newer.

### actutum run

Launch the prebuilt host against the source asset directory for a manual smoke
run. It does not start a development server, watcher, bundler, or hot-reload
process.

- Validate the same project, asset, target, and bundled-host contracts as build.
- Create a unique runtime configuration beside the project manifest so relative
  asset containment remains identical to packaged applications.
- Remove the temporary configuration after normal or unsuccessful host exit.
- Close child stdin, wait for the host, and preserve its non-zero exit code.
- Suppress child output in JSON mode so stdout remains one JSON document.
- Do not copy source assets or create build output.

### actutum build

Validate the project and create a portable application directory,
machine-readable build report, and deterministic ZIP through an atomic staging
flow.

The current output names use the complete `app.id`, preventing projects with
the same leaf identifier from overwriting each other in a shared output root:

    dist/<app-id>/<app-id>.exe
    dist/<app-id>/actutum.runtime.json
    dist/<app-id>/web/**
    dist/<app-id>/build-result.json
    dist/<app-id>.zip

The ZIP contains one top-level `<app-id>/` directory. File order, timestamps, and
portable file modes are normalized. The deterministic report contains contract
versions, release version, identity, permissions, host and asset digests, and
counts; it omits
wall-clock timings and absolute paths. Build duration belongs to benchmark
evidence rather than reproducible artifact bytes.

### actutum inspect PATH

Read an output directory or archive and report its Actutum release, contract
versions, target, permissions, application identity, file counts, byte counts,
and digests without executing it.

Inspection recomputes the host and asset-tree SHA-256 values and validates the
runtime configuration against the build result. ZIP inspection rejects unsafe,
duplicate, case-colliding, multi-root, unexpected, or over-limit entries.

### actutum version

Report the CLI version, supported manifest versions, host compatibility range,
IPC versions, and bundled targets.

## Common Options

| Option | Contract |
| --- | --- |
| --config PATH | Project manifest; default is actutum.json |
| --target TARGET | Explicit build target; MVP accepts windows-x64 |
| --out PATH | Output root; default is dist |
| --json | Emit one JSON document and no decorative human output |
| --quiet | Suppress non-error human output |
| --verbose | Add bounded diagnostics without secrets or source contents |
| --help | Print command help and exit successfully |
| --version | Alias the version command |

`--out` is resolved relative to the manifest's project root. The output root
and asset root may not contain each other.

Command-specific options must be added to this document before implementation
is considered stable.

## Configuration Precedence

1. Explicit command-line options.
2. The project manifest.
3. Documented built-in defaults.

Environment variables do not configure application identity, permissions, or
packaging. Development and benchmark-only environment variables must be named,
documented, and ignored by production builds.

The CLI does not search parent directories beyond the resolved project root and
does not merge multiple manifests.

## JSON Envelope

Successful commands return:

    {
      "schemaVersion": 1,
      "ok": true,
      "command": "build",
      "result": {},
      "diagnostics": []
    }

Failed commands return:

    {
      "schemaVersion": 1,
      "ok": false,
      "command": "build",
      "error": {
        "code": "MANIFEST_INVALID",
        "message": "Project manifest is invalid."
      },
      "diagnostics": []
    }

Diagnostics use stable codes, severity, category, project-relative path,
optional line and column, a short message, and structured facts. They do not
contain timestamps, random identifiers, progress events, or absolute local
paths unless no safe relative representation exists.

## Exit Codes

| Code | Meaning |
| ---: | --- |
| 0 | Success |
| 2 | Usage, manifest, or configuration error |
| 3 | Asset or project input error |
| 4 | Host template or contract compatibility error |
| 5 | Runtime prerequisite unavailable |
| 6 | Packaging or filesystem failure |
| 10 | Unexpected internal failure |

Stable diagnostic codes provide detail within these broad process exit codes.

## Failure and Recovery

- validate and doctor do not write project or output files.
- build writes only to an owned staging directory until completion.
- build removes its staging directory after a handled failure.
- build preserves the previous successful output.
- run returns the child host exit reason and cleans benchmark-only resources.
- Cancellation follows the same cleanup boundary as failure.

## Runtime Compatibility

- CLI release artifacts: Windows x64 first.
- Packaged host: Windows x64 first.
- Web runtime: Evergreen WebView2.
- Minimum client: Windows 10 version 1709 x64, build 16299.
- Minimum server: Windows Server 2016 x64, build 14393.
- Minimum WebView2 Runtime: `92.0.902.49`, the first stable runtime that exposes
  the `ICoreWebView2_4` download-denial interface required by the security
  baseline.
- Maintainer implementation language: Go.
- Consumer machine compiler and Node.js requirement: none.

## Deferred Commands

The MVP does not define plugin, add, publish, update, sign, installer, generate,
bind, dev-server, or shell-completion commands.

## Review Blockers

- A command changes without synchronized help, examples, JSON, diagnostics, and
  exit-code tests.
- JSON output exposes source contents, secrets, unbounded logs, or unstable
  process data.
- A build command performs an undeclared network request.
- A consumer command invokes a compiler or frontend package manager.
- A release puts the CLI and host in different directories without defining a
  new immutable host-discovery contract.
