(function defineCapabilityProbeModel(root) {
  "use strict";

  const states = new Set(["available", "unavailable", "passed", "canceled", "blocked", "failed"]);

  function replaceResult(results, next) {
    if (!next || !next.id || !states.has(next.state)) throw new Error("Capability result is invalid.");
    const replaced = results.map((item) => item.id === next.id ? { ...next } : { ...item });
    return replaced.some((item) => item.id === next.id) ? replaced : [...replaced, { ...next }];
  }

  function preserveOperationResults(existing, detected, operationIDs) {
    const operations = new Set(operationIDs);
    return detected.map((item) => {
      const prior = existing.find((candidate) => candidate.id === item.id);
      if (!operations.has(item.id) || !prior || prior.state === "available" || prior.state === "unavailable") return { ...item };
      return { ...prior };
    });
  }

  function summarize(results) {
    return results.reduce((summary, item) => {
      summary.total += 1;
      if (item.state === "passed") summary.passed += 1;
      if (item.state === "available") summary.available += 1;
      if (item.state === "blocked" || item.state === "failed") summary.failed += 1;
      if (item.state === "canceled") summary.canceled += 1;
      return summary;
    }, { total: 0, passed: 0, available: 0, failed: 0, canceled: 0 });
  }

  function buildReport(results, runtime, environment, capturedAt) {
    return {
      schemaVersion: "velox.capability-probe-result/v1",
      capturedAt,
      runtime: runtime ? { ...runtime } : null,
      environment: { ...environment },
      results: results.map((item) => ({ ...item })),
    };
  }

  root.CapabilityProbeModel = Object.freeze({
    replaceResult,
    preserveOperationResults,
    summarize,
    buildReport,
  });
})(globalThis);
