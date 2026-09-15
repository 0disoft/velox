# Velox Default Icon

`velox.png` is the user-approved V/mint icon generated with the built-in image
generation tool. It is the source for the Windows application icon, not a
File Notes-specific document symbol. The asset is distributed under the
repository's license terms.

`velox.ico` contains 16, 20, 24, 32, 40, 48, 64, 128, and 256 pixel PNG frames.
Separate large (1) and small (11) resource IDs are used because Windows
can reuse a shared icon by resource ID without considering its requested size.

The supported Windows x64 host includes `cmd/velox-host/icon_windows_amd64.syso`.
Keeping this generated resource object in source control gives local and hosted
Go builds the same icon without requiring a resource compiler during normal
builds. The CLI remains unchanged. No application icon configuration is added.

To regenerate after an intentional artwork change, use
`scripts/generate-windows-icon.mjs` with paths to Sharp and an installed
`github.com/tc-hib/winres` module (validated with v0.3.1), and Go available.
The generator copies the tool's module metadata into `.cache/windows-icon`, uses
the installed x/image v0.41.0 in offline mode without modifying either the
application module or the installed tool, and normalizes
the COFF timestamp to zero. The
workspace command contract exposes `velox_icon_generate` for the configured host.

Generation prompt: a high-contrast geometric V, white left stroke and mint right
stroke on a charcoal rounded square, transparent outer margin, without wording
or fine details, intended for small Windows title-bar and taskbar sizes.
