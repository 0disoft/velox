import { expect, test } from "bun:test";
import { summarizeAgentSessionEvidence, validateAgentSessionEvidence, type AgentSessionEvidence, type ToolInvocation } from "./agent-session-evidence.ts";

function invocation(index: number, status: ToolInvocation["status"] = "succeeded"): ToolInvocation {
  return { idSha256: index.toString(16).padStart(64, "0"), tool: "exec_command", kind: "command", inputSha256: "b".repeat(64), startedAtUtc: "2026-09-10T00:00:00.000Z", finishedAtUtc: status === "pending" ? null : "2026-09-10T00:00:01.000Z", status, retryOfSha256: null };
}

function fixture(): AgentSessionEvidence {
  return { schemaVersion: "velox.agent-session-evidence/v1", admission: "diagnostic-only", sourceFormat: "codex-rollout", sourceVersion: "0.153.4", sourceSha256: "a".repeat(64), sessionIdSha256: "c".repeat(64), promptSha256: "d".repeat(64), startedAtUtc: "2026-09-10T00:00:00.000Z", finishedAtUtc: "2026-09-10T00:00:02.000Z", state: "completed", observationComplete: true, invocations: [invocation(1)] };
}

test("accepts runner-neutral source formats without any Hermes database", () => {
  for (const sourceFormat of ["codex-rollout", "hermes-session-projection", "another-agent"]) {
    expect(validateAgentSessionEvidence({ ...fixture(), sourceFormat }).sourceFormat).toBe(sourceFormat);
  }
});

test("counts successful, failed, cancelled and denied requests equally", () => {
  const data = fixture();
  data.invocations = ["succeeded", "failed", "cancelled", "denied"].map((status, i) => invocation(i, status as ToolInvocation["status"]));
  expect(summarizeAgentSessionEvidence(data)).toMatchObject({ toolCalls: 4, retries: 0, withinBudget: true, countingComplete: true, betaTechnicalGate: false });
});

test("counts a new retry request once and checks its prior terminal target", () => {
  const data = fixture();
  data.invocations = [invocation(1, "failed"), { ...invocation(2), startedAtUtc: "2026-09-10T00:00:01.000Z", retryOfSha256: invocation(1).idSha256 }];
  expect(summarizeAgentSessionEvidence(data)).toMatchObject({ toolCalls: 2, retries: 1 });
  data.invocations[0].status = "succeeded";
  expect(() => validateAgentSessionEvidence(data)).toThrow("INVOCATION_RETRY_TARGET_INVALID");
});

test("reports an exceeded budget without discarding the diagnostic evidence", () => {
  const data = fixture();
  data.invocations = Array.from({ length: 71 }, (_, i) => invocation(i));
  expect(summarizeAgentSessionEvidence(data)).toMatchObject({ toolCalls: 71, toolCallBudget: 70, withinBudget: false, betaTechnicalGate: false });
  data.invocations.pop();
  expect(summarizeAgentSessionEvidence(data).withinBudget).toBe(true);
});

test("preserves partial observation and pending work without claiming complete counts", () => {
  const data = { ...fixture(), state: "in-progress" as const, finishedAtUtc: null, invocations: [invocation(1, "pending")] };
  expect(summarizeAgentSessionEvidence(data)).toMatchObject({ toolCalls: 1, pendingInvocations: 1, countingComplete: false, betaTechnicalGate: false });
  expect(summarizeAgentSessionEvidence({ ...data, invocations: [] }).countingComplete).toBe(false);
  expect(summarizeAgentSessionEvidence({ ...fixture(), observationComplete: false }).countingComplete).toBe(false);
});

for (const [name, change, error] of [
  ["duplicate lifecycle/replay IDs", (v: AgentSessionEvidence) => v.invocations.push({ ...v.invocations[0] }), "INVOCATION_DUPLICATE_ID"],
  ["unfinished completed session", (v: AgentSessionEvidence) => v.invocations = [invocation(1, "pending")], "COMPLETED_SESSION_HAS_PENDING_INVOCATION"],
  ["out of session invocation", (v: AgentSessionEvidence) => v.invocations[0].startedAtUtc = "2026-09-09T00:00:00.000Z", "INVOCATION_TIME_RANGE_INVALID"],
  ["impossible date", (v: AgentSessionEvidence) => v.startedAtUtc = "2026-02-30T00:00:00.000Z", "SESSION_TIMESTAMP_INVALID"],
  ["unknown retry target", (v: AgentSessionEvidence) => v.invocations[0].retryOfSha256 = "e".repeat(64), "INVOCATION_RETRY_TARGET_INVALID"],
  ["raw identifier", (v: AgentSessionEvidence) => v.sessionIdSha256 = "private-session", "SESSION_DIGEST_INVALID"],
] as const) {
  test(`rejects ${name}`, () => { const data = fixture(); change(data); expect(() => validateAgentSessionEvidence(data)).toThrow(error); });
}

test("rejects qualification claims and arbitrary raw content fields", () => {
  expect(() => validateAgentSessionEvidence({ ...fixture(), admission: "qualified" })).toThrow("SESSION_ADMISSION_INVALID");
  expect(() => validateAgentSessionEvidence({ ...fixture(), sandboxEnforced: true })).toThrow("SESSION_FIELDS_INVALID");
  const data = fixture();
  expect(() => validateAgentSessionEvidence({ ...data, invocations: [{ ...data.invocations[0], arguments: "raw private input" }] })).toThrow("SESSION_FIELDS_INVALID");
});

test("published schema keeps diagnostics separate from legacy attestation admission", async () => {
  const schema = await Bun.file(new URL("../schema/agent-session-evidence-v1.schema.json", import.meta.url)).json();
  expect(schema.additionalProperties).toBe(false);
  expect(schema.properties.admission.const).toBe("diagnostic-only");
  expect(schema.$defs.invocation.properties.kind.enum).not.toContain("orchestration");
  const legacy = await Bun.file(new URL("./llm-agent-evaluation.ts", import.meta.url)).text();
  expect(legacy).not.toContain("velox.agent-session-evidence/v1");
});
