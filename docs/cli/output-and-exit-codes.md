# Output and Exit Codes

- Status: Draft
- Repository type: cli-tool
- Owner: Project maintainer

## Human Output

Human output is concise, phase oriented, and written to the appropriate stream.

- Normal results and summaries use stdout.
- Errors and actionable diagnostics use stderr.
- Progress output is absent in non-interactive or JSON mode.
- quiet suppresses non-error output.
- verbose adds bounded diagnostic context without secrets or source contents.

Human wording is not a machine contract.

## JSON Output

The json option emits exactly one UTF-8 JSON document to stdout. Decorative
text and progress events are forbidden.

Success envelope:

    {
      "schemaVersion": 1,
      "ok": true,
      "command": "build",
      "result": {},
      "diagnostics": []
    }

Failure envelope:

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

## Diagnostic Shape

A diagnostic contains:

- Stable code.
- Severity.
- Category.
- Short message.
- Project-relative path when available.
- Optional line and column.
- Structured facts needed for repair.
- Optional related locations.

Diagnostics do not contain:

- Source file contents.
- Secrets or environment dumps.
- Absolute paths when a safe relative path exists.
- Timestamps, random identifiers, or progress events.
- Native stack traces in normal output.

## Exit Codes

| Code | Category | Examples |
| ---: | --- | --- |
| 0 | Success | Command completed |
| 2 | Usage or configuration | Unknown option, invalid manifest |
| 3 | Project input | Missing entry point, unsafe asset path |
| 4 | Host compatibility | Unsupported host or protocol version |
| 5 | Runtime prerequisite | WebView2 unavailable |
| 6 | Packaging or filesystem | Copy, staging, archive, promotion failure |
| 10 | Internal failure | Unhandled invariant violation |

Exit codes remain broad. Scripts that need precise classification use the JSON
error or diagnostic code.

## Stable Diagnostic Families

The exact registry is created with implementation. Initial families are:

- CLI usage and option errors.
- Manifest syntax, schema, and semantic errors.
- Asset path and filesystem safety errors.
- Target and host compatibility errors.
- Runtime prerequisite errors.
- Packaging, cleanup, and promotion errors.
- IPC and runtime configuration errors.
- Internal invariant failures.

Codes are added centrally and are never reused for a different meaning.

## Build Result

Successful build JSON should include:

- Velox release and contract versions.
- Target.
- Project-relative input summary.
- Output paths relative to the selected output root.
- File and byte counts.
- Artifact digests.
- Phase durations.
- Cache upload is measured by benchmark tooling, not guessed by build output.

Build output must not claim reproducibility until the reproducibility check
passes.

## Compatibility Policy

- Adding optional JSON fields is compatible within schema version 1.
- Removing, renaming, or changing field meaning requires a new schema version.
- Human output may change without a schema version.
- Exit-code meaning and diagnostic-code meaning do not change silently.
- Unsupported JSON schema versions fail explicitly.

## Review Blockers

- Mixed human and JSON output.
- A success envelope paired with a non-zero exit code.
- A failure envelope paired with exit code zero.
- Unstable filesystem order in diagnostics.
- Source content, secrets, absolute local paths, or native stack traces in
  normal machine output.
