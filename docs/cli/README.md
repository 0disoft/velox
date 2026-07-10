# CLI Tool

- Status: Draft
- Repository type: cli-tool
- Owner: Project maintainer

## Purpose

The Velox CLI is the primary public support surface for developers and CI. It
translates project configuration into validation, an immutable build plan, and
portable output.

No programming SDK, network API, plugin API, or graphical project manager is
part of the MVP.

## Commands

| Command | Purpose | Writes project or output |
| --- | --- | --- |
| init | Create a minimal project skeleton | Project target only |
| validate | Validate configuration and assets | No |
| doctor | Inspect local compatibility | No |
| run | Launch the source project with the generic host | No project writes |
| build | Produce portable output and deterministic ZIP | Output root |
| inspect | Read artifact metadata without execution | No |
| version | Report supported contract versions | No |

Exact options, failure behavior, and deferred commands are owned by
docs/cli/command-contract.md.

## Automation Contract

- Every command is usable without an interactive prompt.
- JSON output emits one versioned document.
- Stable exit codes classify failures broadly.
- Stable diagnostic codes carry detailed failure identity.
- Human output may improve without changing JSON semantics.
- Build and validation do not download dependencies.

## Configuration

The project manifest is velox.json by default. Configuration precedence and
field ownership are defined in docs/cli/configuration.md.

## Output

Human output, JSON envelopes, diagnostics, and exit codes are defined in
docs/cli/output-and-exit-codes.md.

## Compatibility

The first CLI artifact and host target Windows x64. The exact minimum Windows
and WebView2 versions remain UNDECIDED until M0.

Application authors do not install Go or another compiler to use a released
Velox build.

## Deferred Surfaces

- Shell completion.
- Package-manager installation.
- Plugin and extension commands.
- Publishing, update, installer, and signing commands.
- Frontend generation or dependency installation.

Each deferred surface requires a real actor, a compatibility promise, tests,
documentation, and an ADR before becoming public.
