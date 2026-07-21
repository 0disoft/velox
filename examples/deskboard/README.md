# Velox Deskboard

Deskboard is a functional local-first application built entirely from static
HTML, CSS, and JavaScript. It demonstrates the application boundary Velox
supports today rather than pretending to provide a Wails-style Go backend.

## What It Exercises

- persistent task CRUD through versioned `localStorage` data;
- search, status filters, priority filters, and derived progress metrics;
- native `app.getInfo`, window-state, minimize, maximize, and restore IPC;
- keyboard, focus, empty-state, validation, and narrow-window behavior;
- deterministic Velox packaging and startup-ready smoke verification.

The app performs no network request and has no frontend dependency, bundler,
compiler, filesystem API, process API, database, installer, or updater.

## Run

From an assembled Velox release directory, run:

```powershell
velox run --config examples/deskboard/velox.json --out .cache/deskboard-run
```

Build output can be produced with:

```powershell
velox build --config examples/deskboard/velox.json --out ../../dist/examples/deskboard
```

Application data belongs to the WebView2 profile for
`dev.velox.deskboard`. Removing build output does not remove that profile.
