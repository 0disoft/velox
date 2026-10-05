import { test, expect } from "bun:test";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";

const source = readFileSync(new URL("./listener.js", import.meta.url), "utf8");
class ErrorEvent {}
function fixture(topLevel = true, binding = true, sink?: Function) {
  const listeners = new Map<string, Function>();
  const reports: unknown[] = [];
  const window: any = { addEventListener: (name: string, callback: Function) => listeners.set(name, callback) };
  window.top = topLevel ? window : {};
  if (binding) window.__veloxDevDiagnostic = sink ?? ((report: unknown) => { reports.push(report); return Promise.resolve(); });
  runInNewContext(source, { window, location: { origin: "https://app.app.invalid" }, URL, ErrorEvent });
  return { listeners, reports };
}

test("keeps only same-origin path and bounded coordinates, not error contents", () => {
  const f = fixture();
  const event = Object.assign(new ErrorEvent(), { filename: "https://app.app.invalid/app.js?token=secret#secret", lineno: 4, colno: 8 });
  for (const field of ["error", "message", "stack"]) Object.defineProperty(event, field, { get() { throw new Error("private field read"); } });
  f.listeners.get("error")!(event);
  expect(f.reports).toEqual([{ kind: "uncaught-error", source: "app.js", line: 4, column: 8 }]);
});

test("rejects remote credentials and file URLs and ignores resource events", () => {
  const f = fixture();
  for (const filename of ["https://remote.invalid/private?token=secret", "file:///C:/private/secret.js", "https://user:secret@app.app.invalid/app.js", "not a URL"]) {
    f.listeners.get("error")!(Object.assign(new ErrorEvent(), { filename, lineno: -1, colno: Infinity }));
  }
  f.listeners.get("error")!({ target: { src: "private" } });
  expect(f.reports).toHaveLength(4);
  for (const report of f.reports) expect(report).toEqual({ kind: "uncaught-error", source: "", line: 0, column: 0 });
});

test("never reads rejection reason or prevents normal browser handling", () => {
  const f = fixture();
  const event = { get reason() { throw new Error("private reason read"); }, preventDefault() { throw new Error("default handling changed"); } };
  f.listeners.get("unhandledrejection")!(event);
  expect(f.reports).toEqual([{ kind: "unhandled-rejection", source: "", line: 0, column: 0 }]);
});

test("bounds storm reports", () => {
  const f = fixture();
  for (let i = 0; i < 100; i++) f.listeners.get("unhandledrejection")!({});
  expect(f.reports).toHaveLength(20);
});

test("suppresses throwing and rejecting debug bindings", async () => {
  const throwing = fixture(true, true, () => { throw new Error("binding failed"); });
  expect(() => throwing.listeners.get("unhandledrejection")!({})).not.toThrow();
  const rejecting = fixture(true, true, () => Promise.reject(new Error("binding failed")));
  expect(() => rejecting.listeners.get("unhandledrejection")!({})).not.toThrow();
  await Promise.resolve();
});

test("does not register in frames or without a debug binding", () => {
  expect(fixture(false).listeners.size).toBe(0);
  expect(fixture(true, false).listeners.size).toBe(0);
});
