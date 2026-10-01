# ADR 0020: Optional compiler-free executable branding

- Status: Accepted (branding and per-user installer)
- Date: 2026-10-02
- Owner: Project maintainer
- Amends: ADR 0017 (narrow branding and per-user installer approval)
- Installer follow-up: Accepted (per-user, no updater)

## Context

ADR 0001 deferred application-specific executable naming, icon resources, and
signing from M0. ADR 0017 kept the boundary closed and required a new product
and threat-model ADR before implementation. Risk R-005 records "unchanged
generic host prevents application branding" as accepted and to be revisited
only after product viability.

The product now has a working narrow alpha: a prebuilt generic Windows host, an
external `velox.json` manifest with `app.id`, `app.name`, and `app.version`,
and a deterministic portable build. Two presentation facts already hold before
this decision:

- The portable executable is named after `app.id` (`<app.id>.exe`), not the
  generic `velox-host.exe`.
- The WebView2 window title uses `app.name` (`<app.name>`).

What was still missing was per-application Windows resource branding. The
default build copies the prebuilt host byte-for-byte without applying
application identity to its resources, so the packaged executable keeps the
release host's own icon and version resources. `docs/cli/configuration.md`
also stated that `app` values "do not patch host executable resources."

This ADR approves optional, compiler-free executable branding for the Windows
build. It narrowly amends ADR 0017 for branding. An optional per-user Windows
install package was recorded as a proposed follow-up at the original decision
and is now accepted within the same narrow boundary; it is implemented and its
verification is tracked in `VALIDATION.md`.

## Decision

### Optional branding manifest

Add an optional top-level `branding` object to the project manifest. It is
opt-in; when it is absent, the build output is byte-for-byte unchanged from the
shared-host path.

- `icon`: a project-relative path to an `.ico` file.
- `company`, `description`, `copyright`: optional strings written into the
  version resource.

Application identity stays single-sourced from `app`:

- `app.name` supplies the product name and the default file description.
- `app.version` supplies the numeric file and product version and the retained
  version string.

Field constraints are enforced by the manifest parser and the resource writer:

- Text values must be valid UTF-8, at most 256 UTF-16 code units, and free of
  control characters.
- `app.version` must have a numeric core of one to four dot-separated
  components, each from 0 to 65535. An optional `-` or `+` suffix is stripped
  only for the numeric fields and is retained verbatim in the version string.
- `icon` must resolve inside the project root, be at most 2 MiB, and contain
  one to 32 images, each either PNG or `BITMAPINFOHEADER` with dimensions that
  match the ICO directory entry.

Branding is Windows-only. The resource writer is not built for other targets.

### Compiler-free resource editing

Branding edits a staged copy of the prebuilt generic host, not the host source
and not application-native code:

1. Verify the host template before editing: the release metadata
   (`velox-host.json`, ADR 0006) must agree on release, target, host and runtime
   contracts, file size, and SHA-256. A host that fails the check is not edited.
2. Copy the verified host to an owned staging path.
3. Edit resources on the staged copy through the Win32 resource-update API:
   `BeginUpdateResourceW`, one or more `UpdateResourceW` calls, then
   `EndUpdateResourceW`.
4. Replace `RT_VERSION` id `1` and the icon resources: group icons `1` (large)
   and `11` (small), with their `RT_ICON` entries. Existing icon-group languages
   are discovered first so the replacement stays consistent with the template.
5. Preserve executable code and every other resource. The update merges into
   the existing resource section; it does not replace it.
6. Record the final branded host size and SHA-256 in the build report
   (`build-result.json` `host.bytes` and `host.sha256`).

The edit is a pure byte operation. It must not require a resource compiler, a C
toolchain, or a network fetch.

### Reject signed templates

If the host template is a PE32+ image whose security data directory is
non-empty, branding is refused with a stable diagnostic rather than applied.
Editing resources invalidates an Authenticode signature, and silently
invalidating a publisher signature would break the trust and distribution
contract. A branded artifact must be signed in a separately approved signing
channel.

### Default output and preview stay unchanged

- With no `branding` object, the host is copied as-is; its digest equals the
  released host digest and existing determinism, checksum, SBOM, and provenance
  evidence remains valid.
- `velox run` launches the prebuilt generic host directly, so the development
  preview keeps the shared Velox icon. Branding applies only to `velox build`.

### Installer follow-up (accepted)

The original decision recorded an optional Windows install package as proposed
but unbuilt:

- A per-user install into `%LocalAppData%` that needs no elevation.
- A Start Menu shortcut and an uninstall entry; a Desktop shortcut is optional.
- The portable directory and deterministic ZIP remain the default output; the
  install package is an additional, optional output.

That follow-up is now accepted for one narrow implementation:

- A fresh per-user install under
  `%LOCALAPPDATA%\Programs\Velox\<app-id>` with `uninstall.exe` and an
  ownership record, a Start Menu shortcut, and a User-visible Apps uninstall
  entry under `HKCU`.
- The packaging technology is a small, repository-built Go Setup executable,
  not MSIX or a WiX/MSI bundle. No external installer toolchain is required.
- The Setup is an opt-in `velox build --installer` output; the portable
  directory and deterministic ZIP stay the default and are unchanged.
- There is no updater, repair, elevation, machine-wide scope, or runtime
  download. An update is an uninstall followed by a reinstall.
- Removal refuses changed or unowned files, requires the application to be
  closed, and preserves user documents and the WebView2 profile and recovery
  data, which live outside the install tree.

The behavior, layout, payload format, and remaining limitations are documented
in `docs/ops/windows-installer.md`.

### Unchanged boundaries

Branding and the installer do not change:

- The static-only asset model and the unchanged generic backend.
- The closed IPC v1 method table in `docs/architecture/04-ipc-v1.md`.
- The prohibition on application-native compilation during the consumer build.

No new application-runtime native capability, backend, or IPC method is
approved here. The deployment-only shell-link and registry operations of the
per-user installer are approved under this ADR and stay outside the application
runtime.

### Amendment to ADR 0017

ADR 0017 prohibited per-application branding and required a new product and
threat-model ADR for branding-adjacent surfaces. This ADR is that review for
branding and the narrow per-user installer follow-up. It permits optional
compiler-free resource branding under the constraints above and the per-user
install package described above, and leaves updater and signing decisions
untouched. Risk R-005 is addressed by the optional branded path.

## Alternatives

### Keep the host unchanged and never brand

Retained as the default, rejected as the only option. It forces every packaged
application to present with the shared Velox icon and no application-specific
version metadata, which is a real adoption obstacle already tracked as R-005.

### Patch resources into the source host in place

Rejected. In-place edits change the shipped host bytes, cannot preserve the
released digest across repeat builds, and make rollback and verification harder.
Staged-copy editing keeps the source host immutable.

### Brand a signed host and re-sign immediately

Rejected for now. It couples branding to a signing provider and leaves the
artifact unsigned between the edit and the re-sign step. Until a signed channel
is approved, signed templates are refused instead.

### Compile a per-application host with baked-in resources

Rejected. It reintroduces application-specific native compilation, which is the
cost Velox exists to remove.

### Ship an installer as the default output now

Rejected. Portable remains the default per ADR 0017 and ADR 0008; an install
package is an optional addition, not the default output.

## Consequences

### Positive

- Packaged applications can show their own icon, product name, version, and
  legal text without a consumer compiler.
- The default path stays byte-identical, so existing determinism and
  distribution evidence is preserved.
- Resource editing is bounded, local, and inspectable.
- Refusing signed templates keeps signature trust honest.

### Negative

- The build gains a Windows-specific resource-edit path that needs careful,
  deterministic handling and ongoing tests.
- A branded host digest differs from the released host digest; release
  metadata, the build report, and any consumer verifier must account for the
  branded artifact identity.
- A signature-invalidating edit means branded output cannot carry a publisher
  signature until a signing channel is approved.
- An installer adds uninstall, shortcut, and update-lifecycle surface with its
  own maintenance and support cost.

## Validation

The implementation lives in `internal/pebranding`, `internal/buildplan`, and
`internal/builder`. An accepted decision is not proof that tests pass. Required
evidence for the branded path:

- Build with and without branding: the no-branding output stays byte-identical
  to the shared host copy, and a branded output reports the final host size and
  SHA-256.
- Deterministic two-build check: building the same branded project twice yields
  byte-identical branded host bytes and archive hash.
- Resource preservation: the branded host still exposes its required icon
  groups, application manifest, and executable code.
- Signed-template refusal: a signed template fails branding with a stable
  diagnostic and is not modified.
- Security review of the new resource-edit trust boundary, including staged
  path handling, icon path containment, and failure recovery.
- Installer validation named in `VALIDATION.md` covers ownership refusal,
  isolated-registry removal, deterministic Setup bytes, and payload tamper
  refusal. The engine and payload unit tests passed in the full Go suite, and
  all four installer intents (test, bundle, smoke, and File Notes Setup)
  passed locally for beta.3.

Record exact commands, versions, artifacts, and any skipped check.

## Rollback or Fallback

- Removing the `branding` object restores the exact current output; the
  staged-copy edit leaves the source host untouched, so rollback is
  configuration-only.
- If staging or resource editing fails, discard the staged copy and fail the
  build without promoting a partially edited host.
- If determinism or resource preservation cannot be guaranteed, keep branding
  unshipped and retain the unchanged default.

## Revisit Triggers

- A signing channel is approved, which changes the signed-template refusal.
- A per-user install would require elevation or machine-wide scope, or an
  updater, repair, or MSI/MSIX packaging is proposed.
- Branding needs fields, identity sources, or host resources beyond the named
  categories.
- Determinism or resource-preservation evidence fails.
- A new native capability, backend, or IPC method is proposed alongside
  branding.

## Synchronized Surfaces

- `docs/adr/README.md` (ADR table status for 0017 and 0020).
- `docs/adr/0017-continue-as-a-narrow-static-packager.md` (narrow amendment).
- `docs/cli/configuration.md` and `schema/velox-v1.schema.json` (branding
  fields).
- `docs/product/02-spec.md` and `docs/product/03-risk-register.md` (R-005,
  R-021).
- `docs/engineering/00-project-invariants.md` (staged-copy branding boundary).
- `README.md` (supported and deferred feature boundary).
- `VALIDATION.md` for the branded-path and installer validations.
- `docs/ops/windows-installer.md` for installer behavior and layout.
- `assets/branding/README.md` if the default icon generation path changes.
