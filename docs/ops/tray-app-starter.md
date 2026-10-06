# Tray App Starter: 2026-10-06

This is a source-only `init --template tray-app` addition, not part of the
published alpha.66 CLI. Basic output and existing generated projects are
unchanged. The CLI embeds five static assets and the existing declaration;
consumer generation/build still require no compiler or frontend dependencies.

## Scope

The generated manifest grants only `notification.show`. It enables the existing
host tray, single-instance behavior, remembered placement and system-themed
native title bar, using 620x480 dimensions and 360x400 logical minimums. No
global shortcut is registered. The host-owned tray menu supplies hide, restore
and quit; no script-level window or clipboard/file permissions are added.

The UI is an unframed notification composer with a kind selector, textarea,
255-UTF-16-unit count and one licensed Lucide bell button. Native notification
requests occur only on submission. Invalid/control-containing input is blocked,
pending requests disable duplicate submissions, and results retain the message.
No browser notification fallback, scheduler, message history/storage or font is
included. Windows can suppress balloon display despite accepting the request.
CommandCode DeepSeek 4.1 Flash/high supplied the checked style draft, adapted
to the existing starter conventions. Icon licenses ship with the generated app.

## Automated Verification

- Initializer/CLI Go tests and scoped vet passed. Cases cover exact permission
  and window defaults, escaped names, seven-file/five-asset inventory, refusal
  to overwrite a conflicting icon, basic-template preservation and CLI argument
  forms. Hygiene/diff checks also passed.
- Four Bun cases passed for explicit submission, all three kinds, literal
  mixed-language text, UTF-16/control bounds, pending duplicates, failures and
  missing native bridge without browser fallback.
- The source CLI from `83e78c1` generated, validated, built and inspected a
  fresh tray app. Generated host bytes match the unchanged public alpha.66
  host. The CLI was built with Go 1.27.1 using
  `-buildvcs=false -trimpath -ldflags='-s -w'`.
- Headless Edge/Playwright checked light/dark at 620x480 and 360x540 over a
  loopback-only fixture server. There was no horizontal overflow; the 40x40
  button stayed fixed, the bell loaded and rendered, Space activated it once,
  focus returned, and literal Korean/tag-shaped text was retained. Screenshots
  were visually inspected. These interactions used a mock native bridge.
- The packaged app ran with its own profile and a loopback CDP port. One real
  `notification.show` request was accepted. A second invocation with the same
  isolated identity exited 0 and retained the first document marker and text.
  Normal page close and cleanup returned 0. No test-owned process remained
  in the subsequent process check. These are automated native/CDP observations,
  not a new maintainer/manual confirmation.

## Exact Artifact Identity

- Local source CLI SHA-256:
  `690d1fd6cf8b14190aaa88720ee293f0f1bbf23c34c40e6f3768313704d98b62`.
- Public host and generated app EXE SHA-256:
  `7cf4c80d614ff1ba4f942c39dbfd42865b59146d5f5c089b4c5fa0be3c850beb`.
- Static assets: 5 files, 8,387 bytes, tree SHA-256
  `2b6636d54ef16b86e4e56b0c49e9d4ff9cac2bccac0fd55f8c78f69b58aaa4fb`.
- Portable app: 8 files, 4,425,522 bytes; ZIP: 2,165,227 bytes, SHA-256
  `e8c6c0c39d9962f947932ab2a90dae1427c49c6af9ee796f3700d5b0fb481990`.
- Accepted receipt: `.cache/tray-starter-1791288580245/result.json`.

These source CLI bytes still report alpha.66 but are not its published CLI.
The unchanged host comparison is a byte-identity check, not a startup or idle
performance benchmark.

## Retained Fixture Failures

The first attempt accepted the native request, then timed out collecting a
native screenshot. Its process tree was force-cleaned with exit 0; that is not
normal-close evidence. Receipt: `.cache/tray-starter-1791288427879/result.json`.

An intermediate run passed native acceptance/activation/close, but its
file-URL previews showed blank CSS-masked bell icons despite decoding the SVG.
Receipt: `.cache/tray-starter-1791288506646/result.json`. That visual result is
not accepted as a rendered-icon pass. Serving only the four known fixture
resources over loopback HTTP fixed the previews without changing product
assets. The final receipt above supplies the accepted UI/native scope.

## Repeat

Place the source CLI, compatible host and matching `velox-host.json` in
`.cache/manifest-watch-bin`, or set `VELOX_TRAY_STARTER_BIN_DIR`. With Playwright
resolvable by Node and Edge available, run:

```sh
node scripts/tray-app-smoke.mjs
```

The script uses timestamped copied/generated fixtures, a private profile,
loopback preview/CDP ports, and its own child processes. It closes the preview
server/browser and cleans the owned native child tree on failure; receipts
and screenshots remain in its `.cache/tray-starter-*` directory.

## Not Repeated

No manual tray hide/restore/quit check, OS balloon visibility confirmation,
installer execution, hosted stress, startup/performance benchmark or external
user attempt was performed. Existing host tray behavior is reused unchanged.
No new host/public IPC, DB/schema, dependency, CI, version, publication or beta
promotion is included.
