(() => {
  if (window.top !== window || typeof window.__veloxDevDiagnostic !== "function") return;
  const send = window.__veloxDevDiagnostic;
  let reports = 0;
  const coordinate = value => Number.isInteger(value) && value >= 0 && value <= 10000000 ? value : 0;
  const report = (kind, source, line, column) => {
    if (reports >= 20) return;
    reports++;
    try {
      send({ kind, source, line, column }).catch(() => {});
    } catch {}
  };
  window.addEventListener("error", event => {
    if (!(event instanceof ErrorEvent)) return;
    let source = "";
    try {
      if (typeof event.filename === "string" && event.filename.length <= 4096) {
        const url = new URL(event.filename);
        if (url.origin === location.origin && !url.username && !url.password) {
          const path = decodeURIComponent(url.pathname).slice(1);
          if (path.length <= 512) source = path;
        }
      }
    } catch {}
    report("uncaught-error", source, coordinate(event.lineno), coordinate(event.colno));
  });
  window.addEventListener("unhandledrejection", () => {
    report("unhandled-rejection", "", 0, 0);
  });
})();
