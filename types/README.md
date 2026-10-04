# Velox TypeScript Declarations

`velox.d.ts` is a declaration-only mirror of the IPC v1 application-facing
surface. It emits no JavaScript and installs no runtime guard. `README.md` and
`example.ts` sit beside it.

## Where the Files Ship

- A release archive built from the current sources includes `types/velox.d.ts`,
  `types/README.md`, and `types/example.ts`. Releases published before this
  addition do not contain them.
- `velox init` writes a root `velox.d.ts` byte-identical to the declaration the
  CLI embeds, beside the four generated project files. The generated
  `web/app.js` opens with `/// <reference path="../velox.d.ts" />`.
- The declaration stays at the project root. `init` does not copy it into the
  default `web` asset root, so it is not part of generated application assets.
- The declarations are not published to npm.

## Using the Declarations

An application opts in by one of:

- copying `velox.d.ts` into its own sources,
- a type-only `import` from the copied path, or
- a `/// <reference path="..." />` or `checkJs` include from a TypeScript or
  JavaScript project.

Type-only opt-in stays in the type layer and emits no JavaScript:

```ts
import type { VeloxAPI } from "./velox";
```

```js
/// <reference path="./velox.d.ts" />
```

## Example

`example.ts` exports two named helpers: `applicationInfo` calls `app.getInfo`
under `app.info`, and `saveNote` uses `file.save` through
`saveTextAs`/`saveTextTo`. Call them from an explicit application action.
Cancellation and error outcomes leave the caller's
buffer, and a connected save `target` is reused only inside the document that
created the connection. The type check does not execute the example. It
enables no native permission, installs no TypeScript, and is not compiled or
run by `velox init` or `velox build`. TypeScript developers may adapt or
compile it; plain JavaScript users need no compiler for `init` or `build`.

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
unchanged. Shipping the declaration, or running `velox init`, grants no native
permission and installs no TypeScript. The declarations are not published to
npm.

## Checking

```sh
tsc --noEmit -p tests/types/tsconfig.json
```

The check compiles `velox.test.ts` and `example.ts`. `example.ts` is
type-checked only by this command. Validated locally with TypeScript
5.9.3 and 6.0.3.
