import { TOOL_CALL_BUDGET } from "./llm-agent-evaluation.ts";

export interface AgentSessionEvidence {
  schemaVersion: "velox.agent-session-evidence/v1";
  admission: "diagnostic-only";
  sourceFormat: string;
  sourceVersion: string;
  sourceSha256: string;
  sessionIdSha256: string;
  promptSha256: string;
  startedAtUtc: string;
  finishedAtUtc: string | null;
  state: "in-progress" | "completed" | "failed" | "aborted";
  observationComplete: boolean;
  invocations: ToolInvocation[];
}

export interface ToolInvocation {
  idSha256: string;
  tool: string;
  kind: "command" | "file-change" | "mcp" | "web" | "browser" | "other";
  inputSha256: string;
  startedAtUtc: string;
  finishedAtUtc: string | null;
  status: "pending" | "succeeded" | "failed" | "cancelled" | "denied";
  retryOfSha256: string | null;
}

const hash = /^[0-9a-f]{64}$/;
const identifier = /^[a-zA-Z0-9][a-zA-Z0-9._:/-]{0,79}$/;

function check(condition: unknown, code: string): asserts condition {
  if (!condition) throw new Error(code);
}

function record(value: unknown, names: string[]): Record<string, unknown> {
  check(value !== null && typeof value === "object" && !Array.isArray(value), "SESSION_OBJECT_INVALID");
  check(Object.keys(value).sort().join("|") === names.sort().join("|"), "SESSION_FIELDS_INVALID");
  return value as Record<string, unknown>;
}

function matches(value: unknown, pattern: RegExp, code: string) {
  check(typeof value === "string" && pattern.test(value), code);
}

function timestamp(value: unknown): number {
  check(typeof value === "string" && /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}Z$/.test(value), "SESSION_TIMESTAMP_INVALID");
  const parsed = Date.parse(value);
  check(Number.isFinite(parsed) && new Date(parsed).toISOString() === value, "SESSION_TIMESTAMP_INVALID");
  return parsed;
}

export function validateAgentSessionEvidence(raw: unknown): AgentSessionEvidence {
  const session = record(raw, ["schemaVersion", "admission", "sourceFormat", "sourceVersion", "sourceSha256", "sessionIdSha256", "promptSha256", "startedAtUtc", "finishedAtUtc", "state", "observationComplete", "invocations"]);
  check(session.schemaVersion === "velox.agent-session-evidence/v1", "SESSION_SCHEMA_INVALID");
  check(session.admission === "diagnostic-only", "SESSION_ADMISSION_INVALID");
  matches(session.sourceFormat, /^[a-z][a-z0-9-]{0,63}$/, "SESSION_SOURCE_FORMAT_INVALID");
  matches(session.sourceVersion, identifier, "SESSION_SOURCE_VERSION_INVALID");
  for (const name of ["sourceSha256", "sessionIdSha256", "promptSha256"]) matches(session[name], hash, "SESSION_DIGEST_INVALID");
  check(["in-progress", "completed", "failed", "aborted"].includes(session.state as string), "SESSION_STATE_INVALID");
  check(typeof session.observationComplete === "boolean", "SESSION_OBSERVATION_INVALID");
  const start = timestamp(session.startedAtUtc);
  const finish = session.finishedAtUtc === null ? null : timestamp(session.finishedAtUtc);
  check((session.state === "in-progress") === (finish === null), "SESSION_COMPLETION_INVALID");
  check(finish === null || finish >= start, "SESSION_TIME_RANGE_INVALID");
  check(Array.isArray(session.invocations) && session.invocations.length <= 500, "SESSION_INVOCATIONS_INVALID");
  const seen = new Map<string, ToolInvocation>();
  for (const rawCall of session.invocations) {
    const call = record(rawCall, ["idSha256", "tool", "kind", "inputSha256", "startedAtUtc", "finishedAtUtc", "status", "retryOfSha256"]);
    for (const name of ["idSha256", "inputSha256"]) matches(call[name], hash, "INVOCATION_DIGEST_INVALID");
    matches(call.tool, identifier, "INVOCATION_TOOL_INVALID");
    check(["command", "file-change", "mcp", "web", "browser", "other"].includes(call.kind as string), "INVOCATION_KIND_INVALID");
    check(["pending", "succeeded", "failed", "cancelled", "denied"].includes(call.status as string), "INVOCATION_STATUS_INVALID");
    check(!seen.has(call.idSha256 as string), "INVOCATION_DUPLICATE_ID");
    const callStart = timestamp(call.startedAtUtc);
    const callFinish = call.finishedAtUtc === null ? null : timestamp(call.finishedAtUtc);
    check((call.status === "pending") === (callFinish === null), "INVOCATION_COMPLETION_INVALID");
    check(callStart >= start && (finish === null || callStart <= finish) && (callFinish === null || (callFinish >= callStart && (finish === null || callFinish <= finish))), "INVOCATION_TIME_RANGE_INVALID");
    check(session.state !== "completed" || call.status !== "pending", "COMPLETED_SESSION_HAS_PENDING_INVOCATION");
    if (call.retryOfSha256 !== null) {
      matches(call.retryOfSha256, hash, "INVOCATION_RETRY_ID_INVALID");
      const original = seen.get(call.retryOfSha256 as string);
      check(original && ["failed", "cancelled", "denied"].includes(original.status), "INVOCATION_RETRY_TARGET_INVALID");
      check(original.tool === call.tool && original.kind === call.kind && timestamp(original.finishedAtUtc) <= callStart, "INVOCATION_RETRY_MISMATCH");
    }
    seen.set(call.idSha256 as string, call as unknown as ToolInvocation);
  }
  return session as unknown as AgentSessionEvidence;
}

export function summarizeAgentSessionEvidence(raw: unknown) {
  const session = validateAgentSessionEvidence(raw);
  const pendingInvocations = session.invocations.filter((call) => call.status === "pending").length;
  // The adapter emits one record per leaf request, not one per lifecycle event
  // or orchestration wrapper. Completeness is an observation, not attestation.
  return {
    admission: "diagnostic-only" as const,
    betaTechnicalGate: false as const,
    state: session.state,
    toolCalls: session.invocations.length,
    retries: session.invocations.filter((call) => call.retryOfSha256 !== null).length,
    toolCallBudget: TOOL_CALL_BUDGET,
    withinBudget: session.invocations.length <= TOOL_CALL_BUDGET,
    pendingInvocations,
    countingComplete: session.state !== "in-progress" && session.observationComplete && pendingInvocations === 0,
  };
}
