# Velox TypeScript Declarations

`velox.d.ts` is a declaration-only mirror of the IPC v1 application-facing
surface. It ships in this repository only. It emits no JavaScript and installs
no runtime guard.

## Using the Declarations

The file is repository-local, so an application opts in by one of:

- copying `velox.d.ts` into its own sources,
- `import type` from the copied path, or
- a `/// <reference path="..." />` or `checkJs` include from a TypeScript or
  JavaScript project.

## Surface Notes

- The declarations export `Method`, `Params`, `Result`, `VeloxError`, and
  `VeloxAPI`.
- `window.velox` is optional; the host injects it only into a trusted top-level
  document, not browser previews or child frames. Check it before use.
- Clipboard reads and connected saves use `cancelled` discriminated unions.
  Narrow `cancelled` before reading their `text` or `target`. File/folder dialog
  results retain the native fields even on cancellation; check before use.
- Strict `catch` values are `unknown`. Bridge-generated errors carry `code`,
  but check the actual caught value before treating it as `VeloxError`.
  No runtime error guard is exported.

## Authority

The declaration is a type-level view only. Native permission, UTF-8 byte-limit,
range, and integer checks remain authoritative, and runtime behavior is
unchanged. The file is not part of the current release archive or `velox init`
output and is not published to npm.

## Checking

```sh
tsc --noEmit -p tests/types/tsconfig.json
```

Validated locally with TypeScript 5.9.3 and 6.0.3.
