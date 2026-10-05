# Alpha.66 Public Native Follow-Up: 2026-10-05

This is a local follow-up to
[the alpha.66 publication record](alpha66-publication.md). It exercises the
exact downloaded public alpha.66 bytes with the existing native smoke scripts
and one manual packaged-window check. It does not reopen that record: the
publication's artifact-identity scope stays accurate as of publication. Beta
remains held and the release remains unsigned.

## Exact Input Bytes

- Source `842fead0c889e9f161c2567a91c8d0fd4c2ca260`; tag CI
  [37317775399](https://github.com/0disoft/velox/actions/runs/37317775399).
- Bundle: `dist/candidates/alpha66-ci-37317775399/public-extracted/velox-windows-x64`,
  the unauthenticated public download performed in the prior step.
- ZIP SHA-256 `3e1bc83cc8ee31e8feb4870a5b4fa26a4f992a5e22754d0625de891394cfacdc`.
- `velox.exe` SHA-256 `e04309c072c3d4ce96c2dae485e338b3dff225b91b352d6474edb4ccbbd8888b`;
  `velox-host.exe` SHA-256 `7cf4c80d614ff1ba4f942c39dbfd42865b59146d5f5c089b4c5fa0be3c850beb`.
- Client day 2026-10-05; local OS Windows 11 Pro 10.0.26200.
- Rehash check on the follow-up day: the packaged host copied from these
  bytes still matches `7cf4c80d...`.

## Automated Native Smokes

The three existing scripts ran once against the exact public bytes in the
prior step and were not rerun. Each exited 0 and cleaned up on its own. These
script runs are separate from the manual packaged-window check below. Their
normal close exit codes do not establish a normal close for that separate check.

| Script | Environment | Result | Receipt |
| --- | --- | --- | --- |
| `scripts/text-editor-draft-smoke.ts` | `VELOX_DRAFT_RELEASE_DIR=<public path>` | write, restore, clear and cleared-relaunch all exit 0 | `.cache/text-editor-draft-1791209915870/result.json` |
| `scripts/dev-reload-smoke.ts` | `VELOX_RELOAD_RELEASE_DIR=<public path>`, `VELOX_RELOAD_WATCH=1`, `VELOX_RELOAD_VISUAL_ASSETS=1` | HTML/CSS/JS two reload cycles plus cancel/retry; image pixel red-to-green; font actual width 203.23989868164062 to 204.51171875; origin and private profile preserved; debug off; normal close exit 0 | `.cache/normal-reload-1791209932385/result.json` |
| `scripts/development-diagnostics-smoke.ts` | `VELOX_DIAGNOSTICS_RELEASE_DIR=<public path>` | normal run installed no binding and emitted 0 records; debug run emitted 19 bounded stderr records; each run retained one successful stdout JSON envelope and exited 0 | `.cache/development-diagnostics-1791209990264/result.json` |

## Public CLI Packaged Editor

The public `velox.exe` generated and packaged one isolated editor fixture under
`.cache/alpha66-public-editor/`:

- `init my-editor --template text-editor --json` emitted the standard template;
  only the fixture manifest `app.id` (`dev.velox.alpha66-public-editor`) and
  `app.name` (`Velox alpha.66 Public Editor`) were changed. Generated web assets
  were left byte-unchanged.
- `build --config manifest --out ../output --json` succeeded: 9 asset files /
  21,307 bytes (tree `2ac3310b7226bf26e095517462a362778ea75ee4a168b70dfa8fc07c879e4f05`),
  12 portable files / 4,438,401 bytes, app ZIP 2,127,409 bytes SHA-256
  `1abc859aaa6437725c79431526b273ba969212f1f74cc138818f0b6542a21d92`. The
  packaged host inside the build is the public host `7cf4c80d...`.
- `inspect` of the app ZIP passed and reported that same host hash.

## Manual Packaged-Window Check

The packaged EXE ran with `VELOX_DATA_DIR` set to
`.cache/alpha66-public-editor/profile`:

- Typing the literal Korean test string `한글 alpha.66 공개판 복구·저장 확인` stored a draft.
- Leaving the window raised the normal unsaved-change warning.
- Relaunch offered the Recover draft dialog; Restore returned the same literal
  text dirty, and Save opened a fresh native destination picker (no stored path
  or write grant).
- The maintainer saved to `.cache/alpha66-public-editor/recovered-draft.txt`; the
  on-disk readback was exactly 47 UTF-8 bytes, SHA-256
  `52123d1929d19a594d6631e01ff3698d897c9e0046c58c9c33320f9963c2f65d`, and the
  window showed Saved to file with No draft.
- Post-save no-resurrection: relaunching the same profile showed a blank
  `Untitled.txt`, Ready and No draft with no recovery dialog. This passed with
  the same profile and the unchanged exact host hash.
- The relaunched packaged profile's WebView2 process resolved to
  `C:\Program Files (x86)\Microsoft\EdgeWebView\Application\154.0.4258.53\msedgewebview2.exe`
  when matched by the exact `VELOX_DATA_DIR` profile. That version is observed
  only for this manual packaged relaunch; no WebView2 version is claimed for
  the scripted smokes.

## Scope Distinctions

- Post-save no-resurrection was observed before any later typing. The
  maintainer then took over the window and typed new unsaved text; that content
  is not part of this record.
- No normal GUI close is claimed, and no "no residual user-owned process"
  claim is made. Two Alt+F4 attempts were rejected by the UI helper because the
  user had changed state, and the window is intentionally left open to preserve
  the maintainer's input. This is a retained test window, not a lifecycle
  failure.
- An earlier physical Esc stopped the automation before its final relaunch, and
  test processes `18644`/`31024` were force-terminated; that is not normal-close
  evidence. An initial `Start-Process -WindowStyle Hidden` launch (PID `51104`)
  exposed no window, so its tree was terminated and direct invocation used; this
  failed launcher attempt is preserved and was not reproduced as a product bug.

## Not Run

Against these public bytes this follow-up did not run: installer execution, the
folder-browser template, a hosted public-verifier dispatch, full stress or
performance measurement, or an external-user attempt. It adds no runtime, API,
IPC, database/schema, dependency or workflow change; it is documentation only.

The manual observations above are maintainer-local and remain
`same-repository` evidence with `externalUserAttempt: false`, not independent
adoption. Beta stays held. Cache receipts are preserved as observed execution
records, not design authority; the manual-observation receipt is
`.cache/alpha66-public-editor/result.json`.
