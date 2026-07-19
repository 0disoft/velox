# CLI Configuration

- Status: Active
- Repository type: cli-tool
- Owner: Project maintainer

## Configuration Sources

Velox has one project manifest, velox.json by default.

Precedence is:

1. Explicit command-line options.
2. Project manifest values.
3. Documented built-in defaults.

Velox does not merge parent-directory manifests, user-global configuration, or
environment-derived application permissions.

## Manifest Shape

The machine-readable contract is `schema/velox-v1.schema.json`. Its top-level
shape is:

    {
      "schemaVersion": 1,
      "app": {
        "id": "com.example.hello",
        "name": "Hello",
        "version": "0.1.0"
      },
      "assets": {
        "root": "web",
        "entry": "index.html"
      },
      "window": {
        "width": 960,
        "height": 640
      },
      "security": {
        "permissions": []
      }
    }

This example matches the implemented parser and JSON Schema contract.

## Field Ownership

### schemaVersion

Required integer identifying manifest syntax and semantics. Unsupported newer
required versions fail closed.

### app

Application identifier, display name, and version. In M0 and M1 these values
remain external configuration and do not patch host executable resources.

### assets

Project-relative asset directory and HTML entry point. Both must remain inside
the canonical project root after validation.

### window

Initial width and height. Zero or omitted values resolve to 960 by 640. Widths
below 320 and heights below 240 are rejected. Resizable state, position policy,
and background color are not manifest fields in v1.

### security

A closed permission list and production browser settings. Unknown permissions
are errors, not warnings.

## Path Rules

- Relative paths resolve from the manifest's project root.
- Absolute source paths are rejected.
- Parent traversal is rejected.
- Links, junctions, and reparse points that escape ownership are rejected.
- Windows reserved names, alternate data streams, invalid trailing characters,
  and case collisions are rejected.
- Output paths cannot overlap source assets.

## Command-Line Overrides

The MVP allows operational overrides such as manifest path, target, and output
root. Identity, permissions, and security policy are not silently overridden by
environment variables.

## Environment Variables

Production configuration does not depend on environment variables.

Development or benchmark-only variables may select a ready-marker channel or
diagnostic verbosity. They must be explicitly named, documented, bounded, and
ignored by production builds.

## Defaults

Defaults must be:

- Stable within a manifest major version.
- Visible through validate or inspect output.
- Representable in normalized machine-readable output.
- Defined by the schema or one shared implementation source.

No default may grant a native capability.

## Validation

Configuration validation separates:

1. JSON syntax.
2. Schema shape.
3. Semantic constraints.
4. Filesystem and target checks.
5. Host and runtime compatibility.

Each failure returns a stable diagnostic code and project-relative location
when available.

## Deferred Configuration

- Plugin declarations.
- Sidecars and native backends.
- Installer, updater, and signing settings.
- Frontend build commands.
- Multiple windows.
- Remote application URLs.
- macOS and Linux targets.
