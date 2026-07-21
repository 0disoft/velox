import { describe, expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./web/model.js", import.meta.url), "utf8");
const context: Record<string, unknown> = {};
runInNewContext(source, context);
const model = context.CapabilityProbeModel as {
  replaceResult(results: any[], next: any): any[];
  preserveOperationResults(existing: any[], detected: any[], operationIDs: string[]): any[];
  summarize(results: any[]): { total: number; passed: number; available: number; failed: number; canceled: number };
  buildReport(results: any[], runtime: unknown, environment: unknown, capturedAt: string): any;
};

describe("CapabilityProbeModel", () => {
  test("replaces an operation result without mutating the prior collection", () => {
    const original = [{ id: "open-picker", name: "Open", state: "available", detail: "Detected" }];
    const next = model.replaceResult(original, { id: "open-picker", name: "Open", state: "passed", detail: "Read note.md" });
    expect(original[0].state).toBe("available");
    expect(next[0].state).toBe("passed");
  });

  test("preserves exercised outcomes across non-interactive reruns", () => {
    const existing = [{ id: "clipboard", name: "Clipboard", state: "blocked", detail: "Denied" }];
    const detected = [{ id: "clipboard", name: "Clipboard", state: "available", detail: "Detected" }];
    expect(model.preserveOperationResults(existing, detected, ["clipboard"])[0].state).toBe("blocked");
  });

  test("summarizes evidence states without treating availability as success", () => {
    expect(model.summarize([
      { state: "passed" },
      { state: "available" },
      { state: "blocked" },
      { state: "failed" },
      { state: "canceled" },
    ])).toEqual({ total: 5, passed: 1, available: 1, failed: 2, canceled: 1 });
  });

  test("builds a versioned report without sharing mutable result objects", () => {
    const results = [{ id: "indexed-db", state: "passed", detail: "ok" }];
    const report = model.buildReport(results, { version: "0.1.0" }, { origin: "https://app.invalid" }, "2026-07-21T00:00:00.000Z");
    results[0].state = "failed";
    expect(report).toMatchObject({
      schemaVersion: "velox.capability-probe-result/v1",
      capturedAt: "2026-07-21T00:00:00.000Z",
      runtime: { version: "0.1.0" },
      environment: { origin: "https://app.invalid" },
      results: [{ id: "indexed-db", state: "passed", detail: "ok" }],
    });
  });
});
