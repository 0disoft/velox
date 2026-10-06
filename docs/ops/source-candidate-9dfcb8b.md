# Source Candidate `9dfcb8b`: 2026-10-06

## Identity

This local candidate retains the `0.5.10-alpha.66` version string but is not
the published alpha.66 CLI or ZIP. Public alpha.66 remains unchanged.

- Source: `9dfcb8bd62540740c5340553fcb57cfebc38cc90`, clean at build time.
- CLI: compiled once with Go 1.27.1 Windows amd64,
  `-buildvcs=false -trimpath -ldflags='-s -w'`.
- Host and Setup: reused public alpha.66 Go 1.26.0 binaries from source
  `842fead0c889e9f161c2567a91c8d0fd4c2ca260`, checked against its manifest.
- Candidate root: `dist/candidates/source-9dfcb8b/`.
- ZIP: `release/velox-windows-x64.zip`, 6,324,041 bytes, SHA-256
  `bcd8f3f1274843c63eb514eb445fb3e9d55278fa100577843e55c307d90be43a`.

| Binary | Bytes | SHA-256 |
| --- | --- | --- |
| New CLI | 5,062,144 | `690d1fd6cf8b14190aaa88720ee293f0f1bbf23c34c40e6f3768313704d98b62` |
| Reused host | 4,416,000 | `7cf4c80d614ff1ba4f942c39dbfd42865b59146d5f5c089b4c5fa0be3c850beb` |
| Reused Setup | 3,574,784 | `e0b40362dd0c82a8b31ea4922025683f952a372167d3086117fab1b44de86786` |

## Scope

- Manifest-change notices during `run --watch`; edits do not restart the host,
  reload the page or change running permissions/settings.
- CLI notices/errors on stderr in JSON mode, with one stdout result envelope;
  host stderr remains suppressed without `--debug`.
- New `init --template tray-app`, granting only `notification.show` and reusing
  existing tray, single-instance, placement and theme behavior.

## Verification

- Two packaging runs from the same prebuilt inputs produced identical ZIPs.
  This is packaging reproducibility, not two independent recompilations.
- The ZIP was extracted outside the checkout. Only its prebuilt CLI performed
  `init`, `validate`, `build --installer` and `inspect` for basic, text-editor,
  folder-browser and tray-app. All passed; installers were generated, not run.
- All packaged hosts matched the reused public host. Generated declarations
  matched the bundled `types/velox.d.ts`. Tray ZIP packaging repeated identically.
- Extracted release-manifest entries: 16 verified. Sidecar checksum entries:
  3 verified. SPDX 2.3 file SHA-256 entries: 17 verified. The single unsigned
  provenance statement matched the candidate ZIP and source commit.
- Three executables are `NotSigned`. Host and Setup use GUI subsystem 2;
  CLI uses console subsystem 3. Host size and bytes are unchanged.
- `go test ./internal/releasebundle ./internal/releaseevidence ./tests/hygiene`
  passed; focused implementation checks were reused for unchanged inputs.
- CLI/host bytes match the earlier automated tray native evidence. Maintainer
  hide/restore/notification/Quit confirmation is retained separately in
  [tray-app-starter.md](tray-app-starter.md). Manifest-watch native evidence
  remains earlier implementation evidence, not a repeat against this bundle.

Receipts: `dist/candidates/source-9dfcb8b/candidate-result.json` and
`verification-result.json`. Checksum, SPDX and unsigned provenance sidecars
are under `evidence/`. CommandCode DeepSeek 4.1 Flash/high drafted this record;
the facts were checked against local results.

## Boundary

No version bump, tag, push, publication, hosted job, installer execution,
native-watch repeat, startup/performance benchmark or stress run occurred.
No API/IPC, DB/schema, dependency, CI or functional source change is included
in this preparation commit. The initial manual blank-window cause remains
unresolved; the successful replacement check is not a root-cause fix.

Publishing requires a new version decision and separate authorization. The
local unsigned sidecars are not authenticated attestations, and this mixed
new-CLI/reused-host bundle does not claim all binaries were newly compiled.
