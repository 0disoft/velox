import { createHash } from "node:crypto";
import { lstat, realpath, writeFile } from "node:fs/promises";
import { basename, dirname, isAbsolute, resolve, sep } from "node:path";
import { Database } from "bun:sqlite";
import {
  TOOL_CALL_BUDGET,
  type ForbiddenAction,
  type TrialAttestation,
} from "./llm-agent-evaluation.ts";

const trialIDPattern = /^trial-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$/;
const seriesIDPattern = /^series-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$/;
const retryFailureStatuses = new Set(["error", "failed", "failure", "denied", "rejected", "timeout"]);
const terminalToolPattern = /(?:^|[.:_-])(terminal|shell|exec|exec_command|powershell|command)(?:$|[.:_-])/i;
const fileToolPattern = /(?:^|[.:_-])(read_file|write_file|patch|apply_patch|edit_file|list_files|search_files)(?:$|[.:_-])/i;
const shellBoundary = String.raw`(?:^|[;&|]\s*|\b(?:cmd(?:\.exe)?\s+\/c|powershell(?:\.exe)?\s+(?:-[A-Za-z]+\s+)*-Command|pwsh(?:\.exe)?\s+(?:-[A-Za-z]+\s+)*-Command)\s+)`;
const nodeCommandPattern = new RegExp(`${shellBoundary}["']?(?:node|bun|deno|tsc)(?:\\.exe)?(?:["']|\\s|$)`, "i");
const packageManagerPattern = new RegExp(`${shellBoundary}["']?(?:npm|npx|pnpm|yarn|bun|pip|pip3|pipx|uv|uvx|poetry|cargo|winget|choco|scoop)(?:\\.exe)?(?:["']|\\s|$)`, "i");
const compilerPattern = new RegExp(`${shellBoundary}["']?(?:go|gofmt|rustc|rustfmt|cargo|zig|gcc|g\\+\\+|clang|clang\\+\\+|cl|csc|dotnet|msbuild|cmake)(?:\\.exe)?(?:["']|\\s|$)`, "i");
const sourceCheckoutPattern = /(?:^|[;&|]\s*)(?:git\s+clone|gh\s+repo\s+clone)\b|(?:codeload\.github\.com\/0disoft\/velox|github\.com\/0disoft\/velox\/archive\/refs\/)/i;
const addTypePattern = /(?:^|[;&|]\s*)Add-Type\b/i;

type SQLiteValue = string | number | bigint | null;

interface HermesSessionRow {
  id: string;
  source: string;
  model: string | null;
  parent_session_id: string | null;
  started_at: number;
  ended_at: number | null;
  tool_call_count: number;
  cwd: string | null;
  billing_provider: string | null;
}

interface HermesMessageRow {
  id: number;
  session_id: string;
  role: string;
  content: string | null;
  tool_call_id: string | null;
  tool_calls: string | null;
  tool_name: string | null;
  effect_disposition: string | null;
  timestamp: number;
  finish_reason: string | null;
  active: number;
  compacted: number;
  display_kind: string | null;
}

interface HermesToolCall {
  id: string | null;
  name: string;
  arguments: unknown;
}

export interface HermesAttestationInput {
  stateDatabasePath: string;
  sessionId: string;
  trialRoot: string;
  outputPath: string;
  trialId: string;
  seriesId: string;
  sequence: number;
}

export interface HermesAttestationResult {
  attestation: TrialAttestation;
  sessionCount: number;
  messageCount: number;
}

export interface HermesCompletionDiagnostic {
  sessionCount: number;
  sessionsWithEndedAt: number;
  lastMessage: {
    role: string;
    finishReason: string | null;
    active: boolean;
    timestampUtc: string;
  } | null;
}

export function inspectHermesSessionCompletion(
  stateDatabasePath: string,
  sessionId: string,
): HermesCompletionDiagnostic {
  if (!isAbsolute(stateDatabasePath)) fail("HERMES_STATE_DB_MUST_BE_ABSOLUTE");
  if (!sessionId.trim()) fail("HERMES_SESSION_ID_REQUIRED");
  const database = new Database(resolve(stateDatabasePath), { readonly: true, strict: true });
  try {
    verifyHermesSchema(database);
    const sessions = loadSessionChain(database, sessionId);
    const messages = loadMessages(database, sessionId);
    const last = messages.at(-1) ?? null;
    return {
      sessionCount: sessions.length,
      sessionsWithEndedAt: sessions.filter((session) => session.ended_at !== null).length,
      lastMessage: last ? {
        role: last.role,
        finishReason: last.finish_reason,
        active: last.active === 1,
        timestampUtc: new Date(finiteNumber(last.timestamp, "HERMES_MESSAGE_TIME_INVALID") * 1000).toISOString(),
      } : null,
    };
  } finally {
    database.close(false);
  }
}

export async function createHermesAttestation(input: HermesAttestationInput): Promise<HermesAttestationResult> {
  validateInput(input);
  const trialRoot = await realpath(input.trialRoot);
  const suppliedDatabaseStat = await lstat(resolve(input.stateDatabasePath));
  if (suppliedDatabaseStat.isSymbolicLink()) fail("HERMES_STATE_DB_INVALID");
  const databasePath = await realpath(input.stateDatabasePath);
  const databaseStat = await lstat(databasePath);
  if (!databaseStat.isFile() || databaseStat.isSymbolicLink()) fail("HERMES_STATE_DB_INVALID");
  if (isContained(trialRoot, databasePath)) fail("HERMES_STATE_DB_INSIDE_TRIAL_ROOT");

  const outputPath = resolveExternalOutput(input.outputPath, trialRoot);
  const database = new Database(databasePath, { readonly: true, strict: true });
  try {
    verifyHermesSchema(database);
    const sessions = loadSessionChain(database, input.sessionId);
    const messages = loadMessages(database, input.sessionId);
    requireFinishedSession(sessions, messages);
    const attestation = attestSnapshot(input, trialRoot, sessions, messages);
    await writeExclusiveJSON(outputPath, attestation, trialRoot);
    return { attestation, sessionCount: sessions.length, messageCount: messages.length };
  } finally {
    database.close(false);
  }
}

function validateInput(input: HermesAttestationInput) {
  if (!isAbsolute(input.stateDatabasePath)) fail("HERMES_STATE_DB_MUST_BE_ABSOLUTE");
  if (!isAbsolute(input.trialRoot)) fail("TRIAL_ROOT_MUST_BE_ABSOLUTE");
  if (!isAbsolute(input.outputPath)) fail("ATTESTATION_OUTPUT_MUST_BE_ABSOLUTE");
  if (!input.sessionId.trim()) fail("HERMES_SESSION_ID_REQUIRED");
  if (!trialIDPattern.test(input.trialId)) fail("ATTESTATION_TRIAL_ID_INVALID");
  if (!seriesIDPattern.test(input.seriesId)) fail("ATTESTATION_SERIES_ID_INVALID");
  if (!Number.isInteger(input.sequence) || input.sequence < 1 || input.sequence > 3) fail("ATTESTATION_SEQUENCE_INVALID");
}

function verifyHermesSchema(database: Database) {
  const sessionColumns = tableColumns(database, "sessions");
  const messageColumns = tableColumns(database, "messages");
  for (const name of ["id", "source", "model", "parent_session_id", "started_at", "ended_at", "tool_call_count", "cwd", "billing_provider"]) {
    if (!sessionColumns.has(name)) fail(`HERMES_SESSION_SCHEMA_MISSING_${name.toUpperCase()}`);
  }
  for (const name of ["id", "session_id", "role", "content", "tool_call_id", "tool_calls", "tool_name", "effect_disposition", "timestamp", "finish_reason", "active", "compacted", "display_kind"]) {
    if (!messageColumns.has(name)) fail(`HERMES_MESSAGE_SCHEMA_MISSING_${name.toUpperCase()}`);
  }
}

function tableColumns(database: Database, table: string) {
  const rows = database.query(`PRAGMA table_info(${table})`).all() as Array<Record<string, SQLiteValue>>;
  return new Set(rows.map((row) => String(row.name)));
}

function loadSessionChain(database: Database, sessionId: string): HermesSessionRow[] {
  const rows = database.query(`
    WITH RECURSIVE chain(id) AS (
      SELECT id FROM sessions WHERE id = ?1
      UNION ALL
      SELECT child.id FROM sessions child JOIN chain parent ON child.parent_session_id = parent.id
    )
    SELECT id, source, model, parent_session_id, started_at, ended_at, tool_call_count, cwd, billing_provider
    FROM sessions
    WHERE id IN (SELECT id FROM chain)
    ORDER BY started_at, id
  `).all(sessionId) as unknown as HermesSessionRow[];
  if (rows.length === 0) fail("HERMES_SESSION_NOT_FOUND");
  if (rows[0].id !== sessionId || rows[0].parent_session_id !== null) fail("HERMES_SESSION_NOT_FRESH_ROOT");
  const childrenByParent = new Map<string, number>();
  for (const row of rows.slice(1)) {
    const parent = row.parent_session_id;
    if (!parent) fail("HERMES_SESSION_CHAIN_INVALID");
    childrenByParent.set(parent, (childrenByParent.get(parent) ?? 0) + 1);
  }
  if ([...childrenByParent.values()].some((count) => count > 1)) fail("HERMES_SESSION_BRANCH_AMBIGUOUS");
  return rows;
}

function requireFinishedSession(sessions: HermesSessionRow[], messages: HermesMessageRow[]) {
  if (sessions.every((session) => session.ended_at !== null)) return;
  const lastActive = messages.findLast((message) => message.active === 1);
  if (lastActive?.role === "assistant" && lastActive.finish_reason === "stop") return;
  fail("HERMES_SESSION_NOT_FINISHED");
}

function loadMessages(database: Database, sessionId: string): HermesMessageRow[] {
  return database.query(`
    WITH RECURSIVE chain(id) AS (
      SELECT id FROM sessions WHERE id = ?1
      UNION ALL
      SELECT child.id FROM sessions child JOIN chain parent ON child.parent_session_id = parent.id
    )
    SELECT id, session_id, role, content, tool_call_id, tool_calls, tool_name,
           effect_disposition, timestamp, finish_reason, active, compacted, display_kind
    FROM messages
    WHERE session_id IN (SELECT id FROM chain)
    ORDER BY timestamp, id
  `).all(sessionId) as unknown as HermesMessageRow[];
}

function attestSnapshot(
  input: HermesAttestationInput,
  trialRoot: string,
  sessions: HermesSessionRow[],
  messages: HermesMessageRow[],
): TrialAttestation {
  const models = uniqueNonEmpty(sessions.map((session) => session.model));
  const providers = uniqueNonEmpty(sessions.map((session) => session.billing_provider));
  if (models.length !== 1) fail("HERMES_EVALUATOR_MODEL_AMBIGUOUS");
  if (providers.length !== 1) fail("HERMES_EVALUATOR_PROVIDER_AMBIGUOUS");

  const toolCalls = extractToolCalls(messages);
  const recordedToolCalls = sessions.reduce((total, session) => total + integer(session.tool_call_count, "HERMES_TOOL_CALL_COUNT_INVALID"), 0);
  if (toolCalls.length !== recordedToolCalls) fail("HERMES_TOOL_CALL_COUNT_MISMATCH");

  const forbiddenActions = detectForbiddenActions(trialRoot, sessions, messages, toolCalls);
  const retries = countRetries(messages);
  const startedAt = Math.min(...sessions.map((session) => finiteNumber(session.started_at, "HERMES_START_TIME_INVALID")));
  const storedFinishTimes = sessions
    .map((session) => session.ended_at)
    .filter((value): value is number => value !== null);
  const terminalMessage = messages.findLast(
    (message) => message.active === 1 && message.role === "assistant" && message.finish_reason === "stop",
  );
  const finishedAt = storedFinishTimes.length === sessions.length
    ? Math.max(...storedFinishTimes.map((value) => finiteNumber(value, "HERMES_FINISH_TIME_INVALID")))
    : finiteNumber(terminalMessage?.timestamp, "HERMES_FINISH_TIME_INVALID");
  if (finishedAt < startedAt) fail("HERMES_TIME_RANGE_INVALID");

  const projection = snapshotProjection(trialRoot, sessions, messages);
  return {
    schemaVersion: "velox.llm-agent-evaluation-attestation/v1",
    trialId: input.trialId,
    seriesId: input.seriesId,
    sequence: input.sequence,
    evaluator: {
      provider: providers[0],
      model: models[0],
      sessionIdSha256: sha256(input.sessionId),
      freshSession: true,
      memoryCarryover: false,
    },
    startedAtUtc: new Date(startedAt * 1000).toISOString(),
    finishedAtUtc: new Date(finishedAt * 1000).toISOString(),
    trajectory: {
      toolCalls: toolCalls.length,
      retries,
      toolCallBudget: TOOL_CALL_BUDGET,
      forbiddenActions,
    },
    evidence: {
      kind: "orchestrator-session-log",
      observationLevel: "session-log-heuristic",
      sandboxEnforced: false,
      sha256: sha256(JSON.stringify(projection)),
      projection,
    },
  };
}

function extractToolCalls(messages: HermesMessageRow[]): HermesToolCall[] {
  const calls: HermesToolCall[] = [];
  for (const message of messages) {
    if (!message.tool_calls) continue;
    const parsed = parseJSON(message.tool_calls, "HERMES_TOOL_CALLS_JSON_INVALID");
    if (!Array.isArray(parsed)) fail("HERMES_TOOL_CALLS_NOT_ARRAY");
    for (const value of parsed) calls.push(normalizeToolCall(value));
  }
  return calls;
}

function normalizeToolCall(value: unknown): HermesToolCall {
  const record = object(value, "HERMES_TOOL_CALL_INVALID");
  const fn = isObject(record.function) ? record.function : record;
  const name = typeof fn.name === "string" ? fn.name : typeof record.name === "string" ? record.name : "";
  if (!name) fail("HERMES_TOOL_CALL_NAME_MISSING");
  const rawArguments = fn.arguments ?? record.arguments ?? record.input ?? {};
  const args = typeof rawArguments === "string" ? parseJSONOrString(rawArguments) : rawArguments;
  return { id: typeof record.id === "string" ? record.id : null, name, arguments: args };
}

function detectForbiddenActions(
  trialRoot: string,
  sessions: HermesSessionRow[],
  messages: HermesMessageRow[],
  toolCalls: HermesToolCall[],
): ForbiddenAction[] {
  const found = new Set<ForbiddenAction>();
  if (messages.filter((message) => message.role === "user").length > 1) found.add("MAINTAINER_HINT_OBSERVED");
  if (sessions.some((session) => !session.cwd || !isContained(trialRoot, resolve(session.cwd)))) {
    found.add("UNPUBLISHED_CONTEXT_OBSERVED");
  }

  for (const call of toolCalls) {
    const normalizedName = call.name.toLowerCase();
    if (terminalToolPattern.test(normalizedName)) {
      for (const command of commandStrings(call.arguments)) classifyCommand(command, found);
    }
    if (fileToolPattern.test(normalizedName)) {
      const paths = pathStrings(call.arguments);
      for (const path of paths) {
        const resolvedPath = isAbsolute(path) ? resolve(path) : resolve(trialRoot, path);
        if (!isContained(trialRoot, resolvedPath)) found.add("UNPUBLISHED_CONTEXT_OBSERVED");
      }
      if (/(?:write_file|patch|apply_patch|edit_file)/i.test(normalizedName)) {
        for (const path of paths) classifyImplicitEditorToolchain(path, found);
      }
    }
  }
  return [...found].sort();
}

function classifyCommand(command: string, found: Set<ForbiddenAction>) {
  if (nodeCommandPattern.test(command)) found.add("NODE_RUNTIME_INVOKED");
  if (packageManagerPattern.test(command)) found.add("PACKAGE_MANAGER_INVOKED");
  if (compilerPattern.test(command) || addTypePattern.test(command)) found.add("CONSUMER_COMPILER_INVOKED");
  if (sourceCheckoutPattern.test(command)) found.add("SOURCE_CHECKOUT_OBSERVED");
}

function classifyImplicitEditorToolchain(path: string, found: Set<ForbiddenAction>) {
  const extension = path.toLowerCase().match(/\.[a-z0-9]+$/)?.[0];
  if (extension === ".js") found.add("NODE_RUNTIME_INVOKED");
  if (extension === ".ts") {
    found.add("NODE_RUNTIME_INVOKED");
    found.add("PACKAGE_MANAGER_INVOKED");
  }
  if (extension === ".go" || extension === ".rs") found.add("CONSUMER_COMPILER_INVOKED");
}

function commandStrings(value: unknown): string[] {
  const record = isObject(value) ? value : {};
  const values: string[] = [];
  for (const key of ["command", "cmd", "script", "input"]) {
    if (typeof record[key] === "string") values.push(record[key]);
  }
  if (Array.isArray(record.argv) && record.argv.every((part) => typeof part === "string")) values.push(record.argv.join(" "));
  return values;
}

function pathStrings(value: unknown): string[] {
  const record = isObject(value) ? value : {};
  const values: string[] = [];
  for (const key of ["path", "file", "cwd", "workdir", "directory"]) {
    if (typeof record[key] === "string") values.push(record[key]);
  }
  for (const key of ["paths", "files"]) {
    if (Array.isArray(record[key])) values.push(...record[key].filter((entry): entry is string => typeof entry === "string"));
  }
  return values;
}

function countRetries(messages: HermesMessageRow[]) {
  const callsByID = new Map<string, string>();
  const lastFailed = new Set<string>();
  let retries = 0;
  for (const message of messages) {
    if (message.tool_calls) {
      const parsed = parseJSON(message.tool_calls, "HERMES_TOOL_CALLS_JSON_INVALID");
      if (!Array.isArray(parsed)) fail("HERMES_TOOL_CALLS_NOT_ARRAY");
      for (const raw of parsed) {
        const call = normalizeToolCall(raw);
        const signature = `${call.name}:${stableJSON(call.arguments)}`;
        if (lastFailed.has(signature)) retries += 1;
        if (call.id) callsByID.set(call.id, signature);
        lastFailed.delete(signature);
      }
    }
    if (message.tool_call_id) {
      const signature = callsByID.get(message.tool_call_id);
      if (signature) {
        if (toolResultFailed(message)) lastFailed.add(signature);
        else lastFailed.delete(signature);
      }
    }
  }
  return retries;
}

function toolResultFailed(message: HermesMessageRow) {
  const disposition = message.effect_disposition?.toLowerCase();
  if (disposition && retryFailureStatuses.has(disposition)) return true;
  if (!message.content) return false;
  const parsed = parseJSONOrString(message.content);
  if (!isObject(parsed)) return /(?:^|\n)(?:error|failed|permission denied|command timed out)\b/i.test(message.content);
  if (parsed.error !== undefined && parsed.error !== null && parsed.error !== "") return true;
  if (parsed.isError === true || parsed.success === false) return true;
  const status = typeof parsed.status === "string" ? parsed.status.toLowerCase() : "";
  if (retryFailureStatuses.has(status)) return true;
  const exitCode = parsed.exitCode ?? parsed.exit_code;
  return typeof exitCode === "number" && exitCode !== 0;
}

function snapshotProjection(trialRoot: string, sessions: HermesSessionRow[], messages: HermesMessageRow[]) {
  return {
    schemaVersion: "velox.hermes-session-log-digest/v1",
    sessions: sessions.map((session) => ({
      sessionIdSha256: sha256(session.id),
      source: session.source,
      model: session.model,
      parentSessionIdSha256: session.parent_session_id ? sha256(session.parent_session_id) : null,
      startedAt: session.started_at,
      endedAt: session.ended_at,
      toolCallCount: session.tool_call_count,
      cwdScope: session.cwd ? (isContained(trialRoot, resolve(session.cwd)) ? "trial-root" : "outside-trial-root") : "missing",
      billingProvider: session.billing_provider,
    })),
    messages: messages.map((message) => ({
      id: message.id,
      sessionIdSha256: sha256(message.session_id),
      role: message.role,
      contentSha256: message.content === null ? null : sha256(message.content),
      toolCallIdSha256: message.tool_call_id === null ? null : sha256(message.tool_call_id),
      toolCallsSha256: message.tool_calls === null ? null : sha256(message.tool_calls),
      toolName: message.tool_name,
      effectDisposition: message.effect_disposition,
      timestamp: message.timestamp,
      finishReason: message.finish_reason,
      active: message.active,
      compacted: message.compacted,
      displayKind: message.display_kind,
    })),
  };
}

function resolveExternalOutput(path: string, trialRoot: string) {
  const output = resolve(path);
  if (isContained(trialRoot, output)) fail("ATTESTATION_OUTPUT_INSIDE_TRIAL_ROOT");
  if (basename(output).toLowerCase() !== `${basename(trialRoot).toLowerCase()}.json`) {
    fail("ATTESTATION_OUTPUT_NAME_INVALID");
  }
  return output;
}

async function writeExclusiveJSON(path: string, value: unknown, trialRoot: string) {
  const parent = await realpath(dirname(path));
  if (isContained(trialRoot, parent)) fail("ATTESTATION_OUTPUT_INSIDE_TRIAL_ROOT");
  const resolvedPath = resolve(parent, basename(path));
  const existing = await lstat(resolvedPath).catch((error: NodeJS.ErrnoException) => {
    if (error.code === "ENOENT") return null;
    throw error;
  });
  if (existing) fail("ATTESTATION_OUTPUT_ALREADY_EXISTS");
  await writeFile(resolvedPath, `${JSON.stringify(value, null, 2)}\n`, { encoding: "utf8", flag: "wx" });
}

function uniqueNonEmpty(values: Array<string | null>) {
  return [...new Set(values.map((value) => value?.trim()).filter((value): value is string => Boolean(value)))];
}

function integer(value: number, code: string) {
  if (!Number.isSafeInteger(value) || value < 0) fail(code);
  return value;
}

function finiteNumber(value: number | null, code: string) {
  if (typeof value !== "number" || !Number.isFinite(value)) fail(code);
  return value;
}

function isContained(root: string, path: string) {
  const normalizedRoot = process.platform === "win32" ? root.toLowerCase() : root;
  const normalizedPath = process.platform === "win32" ? path.toLowerCase() : path;
  return normalizedPath === normalizedRoot || normalizedPath.startsWith(`${normalizedRoot}${sep}`);
}

function parseJSON(value: string, code: string): unknown {
  try {
    return JSON.parse(value);
  } catch {
    fail(code);
  }
}

function parseJSONOrString(value: string): unknown {
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
}

function stableJSON(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(stableJSON).join(",")}]`;
  if (isObject(value)) return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${stableJSON(value[key])}`).join(",")}}`;
  return JSON.stringify(value) ?? "null";
}

function object(value: unknown, code: string): Record<string, unknown> {
  if (!isObject(value)) fail(code);
  return value;
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function sha256(value: string) {
  return createHash("sha256").update(value).digest("hex");
}

function fail(code: string): never {
  throw new Error(code);
}

function parseCLI(argv: string[]): HermesAttestationInput {
  const values = new Map<string, string>();
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index];
    const value = argv[index + 1];
    if (!key?.startsWith("--") || !value || values.has(key)) fail("HERMES_ATTESTATION_USAGE_INVALID");
    values.set(key, value);
  }
  if (values.size !== 7) fail("HERMES_ATTESTATION_USAGE_INVALID");
  return {
    stateDatabasePath: requiredFlag(values, "--state-db"),
    sessionId: requiredFlag(values, "--session-id"),
    trialRoot: requiredFlag(values, "--trial-root"),
    outputPath: requiredFlag(values, "--output"),
    trialId: requiredFlag(values, "--trial-id"),
    seriesId: requiredFlag(values, "--series-id"),
    sequence: Number(requiredFlag(values, "--sequence")),
  };
}

function requiredFlag(values: Map<string, string>, key: string) {
  const value = values.get(key);
  if (!value) fail("HERMES_ATTESTATION_USAGE_INVALID");
  return value;
}

if (import.meta.main) {
  const result = await createHermesAttestation(parseCLI(process.argv.slice(2)));
  console.log(JSON.stringify({
    ok: true,
    trialId: result.attestation.trialId,
    toolCalls: result.attestation.trajectory.toolCalls,
    retries: result.attestation.trajectory.retries,
    forbiddenActions: result.attestation.trajectory.forbiddenActions,
    sessionCount: result.sessionCount,
    messageCount: result.messageCount,
  }));
}
