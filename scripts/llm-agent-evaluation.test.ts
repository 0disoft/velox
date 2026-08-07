import { createHash } from "node:crypto";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, resolve } from "node:path";
import { Database } from "bun:sqlite";
import { describe, expect, test } from "bun:test";
import { createHermesAttestation } from "./hermes-evaluation-attestation.ts";
import {
  attestEvaluationTrial,
  bindEvaluationSession,
  prepareEvaluationSeries,
  verifyEvaluationSeries,
} from "./llm-agent-orchestrator.ts";
import { loadAndVerifyTrial, summarizeSeries, type TrialAttestation, type TrialRecord } from "./llm-agent-evaluation.ts";

const fixedDigest = "a".repeat(64);
const prompt = "public evaluation task\n";

describe("LLM agent evaluation", () => {
  test("verifies artifact bytes and summarizes three diverse passing trials", async () => {
    const root = await createSeries();
    const trials = [];
    for (const [index, model] of ["model-a", "model-a", "model-b"].entries()) {
      const trialRoot = resolve(root, `trial-${index + 1}`);
      const record = await createTrial(trialRoot, index + 1, model);
      trials.push(await verifyTrial(root, trialRoot));
      expect(trials[index].trialId).toBe(record.trialId);
    }
    expect(summarizeSeries(trials)).toMatchObject({
      passedTrials: 3,
      failedTrials: 0,
      heldTrials: 0,
      outcome: "passed",
      betaTechnicalGate: true,
      modelIdentifiers: ["provider/model-a", "provider/model-b"],
      humanAdoptionClaim: false,
    });
  });

  test("rejects artifact tampering", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    await createTrial(trialRoot, 1, "model-a");
    await writeFile(resolve(trialRoot, "artifacts/first.zip"), "tampered", "utf8");
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ARTIFACT_DIGEST_MISMATCH_FIRSTBUILDARCHIVE");
  });

  test("rejects path traversal before reading an artifact", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    const record = await createTrial(trialRoot, 1, "model-a");
    record.artifacts.safeReport = "../report.md";
    await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ARTIFACT_PATH_INVALID_SAFEREPORT");
  });

  test("rejects reuse of one artifact path for both builds", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    const record = await createTrial(trialRoot, 1, "model-a");
    record.artifacts.secondBuildArchive = record.artifacts.firstBuildArchive;
    await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ARTIFACT_PATH_DUPLICATE");
  });

  test("rejects a self-consistent digest for the wrong build-result identity", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    const record = await createTrial(trialRoot, 1, "model-a");
    const wrong = Buffer.from('{"schemaVersion":"not-velox"}\n');
    await writeFile(resolve(trialRoot, "artifacts/build-result.json"), wrong);
    record.artifacts.buildResultSha256 = sha(wrong);
    await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("BUILD_RESULT_SCHEMA_INVALID");
  });

  test("rejects a passed claim with a failed hard gate", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    const record = await createTrial(trialRoot, 1, "model-a");
    record.gates.startupReady = false;
    await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("PASSED_TRIAL_HAS_FAILED_GATE");
  });

  test("rejects an agent-generated session digest that differs from the orchestrator attestation", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    const record = await createTrial(trialRoot, 1, "model-a");
    record.evaluator.sessionIdSha256 = sha(Buffer.from("invented-session"));
    await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ATTESTATION_SESSION_DIGEST_MISMATCH");
  });

  test("rejects a passed claim that hides an attested Node.js invocation", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    await createTrial(trialRoot, 1, "model-a");
    const attestation = await readAttestation(trialRoot);
    attestation.trajectory.forbiddenActions = ["NODE_RUNTIME_INVOKED"];
    await writeAttestation(trialRoot, attestation);
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ATTESTATION_FORBIDDEN_ACTIONS_MISMATCH");
  });

  test("rejects an under-reported tool-call count", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    await createTrial(trialRoot, 1, "model-a");
    const attestation = await readAttestation(trialRoot);
    attestation.trajectory.toolCalls += 1;
    await writeAttestation(trialRoot, attestation);
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ATTESTATION_TOOL_CALL_COUNT_MISMATCH");
  });

  test("rejects a reported time range that does not cover the orchestrator session", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    const record = await createTrial(trialRoot, 1, "model-a");
    record.startedAtUtc = record.finishedAtUtc;
    await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ATTESTATION_TIME_RANGE_NOT_COVERED");
  });

  test("rejects an attested tool-call budget overrun", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    const record = await createTrial(trialRoot, 1, "model-a");
    const attestation = await readAttestation(trialRoot);
    record.trajectory.toolCalls = 71;
    attestation.trajectory.toolCalls = 71;
    await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
    await writeAttestation(trialRoot, attestation);
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ATTESTED_TOOL_CALL_BUDGET_EXCEEDED");
  });

  test("rejects an attestation stored inside the agent-controlled trial root", async () => {
    const root = await createSeries();
    const trialRoot = resolve(root, "trial-1");
    await createTrial(trialRoot, 1, "model-a");
    const localAttestation = resolve(trialRoot, "attestation.json");
    await writeFile(localAttestation, `${JSON.stringify(await readAttestation(trialRoot), null, 2)}\n`, "utf8");
    await expect(loadAndVerifyTrial(resolve(trialRoot, "result.json"), trialRoot, resolve(root, "task.md"), localAttestation)).rejects.toThrow("ATTESTATION_INSIDE_TRIAL_ROOT");
  });

  test("holds an otherwise passing single-model series", async () => {
    const root = await createSeries();
    const trials = [];
    for (let index = 1; index <= 3; index += 1) {
      const trialRoot = resolve(root, `trial-${index}`);
      await createTrial(trialRoot, index, "model-a");
      trials.push(await verifyTrial(root, trialRoot));
    }
    expect(summarizeSeries(trials)).toMatchObject({
      outcome: "held",
      betaTechnicalGate: false,
      diagnostics: ["MODEL_DIVERSITY_INSUFFICIENT"],
    });
  });

  test("preserves a failed sequence in the series verdict", async () => {
    const root = await createSeries();
    const trials = [];
    for (let index = 1; index <= 3; index += 1) {
      const trialRoot = resolve(root, `trial-${index}`);
      const record = await createTrial(trialRoot, index, index === 3 ? "model-b" : "model-a");
      if (index === 2) {
        record.outcome = "failed";
        record.gates.startupReady = false;
        record.failure = { phase: "startup", code: "STARTUP_NOT_READY" };
        await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
      }
      trials.push(await verifyTrial(root, trialRoot));
    }
    expect(summarizeSeries(trials)).toMatchObject({
      passedTrials: 2,
      failedTrials: 1,
      outcome: "failed",
      betaTechnicalGate: false,
      diagnostics: ["TRIAL_FAILURE_PRESENT"],
    });
  });

  test("writes a machine-readable series summary through the bounded CLI", async () => {
    const root = await createSeries();
    for (const [index, model] of ["model-a", "model-a", "model-b"].entries()) {
      await createTrial(resolve(root, `trial-${index + 1}`), index + 1, model);
    }
    const summaryPath = resolve(root, "summary.json");
    const child = Bun.spawn([
      process.execPath,
      resolve(import.meta.dir, "verify-llm-agent-evaluation.ts"),
      "series",
      root,
      resolve(root, "task.md"),
      resolve(root, "attestations"),
      summaryPath,
    ], { stdout: "pipe", stderr: "pipe" });
    const [exitCode, stdout, stderr] = await Promise.all([
      child.exited,
      new Response(child.stdout).text(),
      new Response(child.stderr).text(),
    ]);
    expect(exitCode, stderr).toBe(0);
    expect(JSON.parse(stdout)).toMatchObject({ betaTechnicalGate: true, outcome: "passed" });
    expect(JSON.parse(await Bun.file(summaryPath).text())).toMatchObject({
      schemaVersion: "velox.llm-agent-evaluation-series/v1",
      betaTechnicalGate: true,
      humanAdoptionClaim: false,
    });
  });

  test("generates an external attestation from a finished Hermes session", async () => {
    const fixture = await createHermesFixture([
      toolCallMessage(2, "call-1", "terminal", { command: "velox validate" }),
      toolResultMessage(3, "call-1", { error: "validation failed" }),
      toolCallMessage(4, "call-2", "terminal", { command: "velox validate" }),
      toolResultMessage(5, "call-2", { status: "ok" }),
    ]);
    const result = await createHermesAttestation(fixture.input);
    const output = await Bun.file(fixture.input.outputPath).text();
    expect(result).toMatchObject({
      sessionCount: 1,
      messageCount: 5,
      attestation: {
        evaluator: { provider: "custom", model: "fixture-model", freshSession: true, memoryCarryover: false },
        trajectory: { toolCalls: 2, retries: 1, toolCallBudget: 70, forbiddenActions: [] },
      },
    });
    expect(result.attestation.evaluator.sessionIdSha256).toBe(sha(Buffer.from(fixture.sessionId)));
    expect(result.attestation.evidence.sha256).toMatch(/^[0-9a-f]{64}$/);
    expect(output).not.toContain(fixture.sessionId);
  });

  test("writes the Hermes attestation through the bounded CLI", async () => {
    const fixture = await createHermesFixture([]);
    const child = Bun.spawn([
      process.execPath,
      resolve(import.meta.dir, "hermes-evaluation-attestation.ts"),
      "--state-db", fixture.input.stateDatabasePath,
      "--session-id", fixture.input.sessionId,
      "--trial-root", fixture.input.trialRoot,
      "--output", fixture.input.outputPath,
      "--trial-id", fixture.input.trialId,
      "--series-id", fixture.input.seriesId,
      "--sequence", String(fixture.input.sequence),
    ], { stdout: "pipe", stderr: "pipe" });
    const [exitCode, stdout, stderr] = await Promise.all([
      child.exited,
      new Response(child.stdout).text(),
      new Response(child.stderr).text(),
    ]);
    expect(exitCode, stderr).toBe(0);
    expect(JSON.parse(stdout)).toMatchObject({
      ok: true,
      trialId: fixture.input.trialId,
      toolCalls: 0,
      retries: 0,
      forbiddenActions: [],
      sessionCount: 1,
      messageCount: 1,
    });
    expect(await Bun.file(fixture.input.outputPath).exists()).toBe(true);
  });

  test("detects explicit commands and implicit editor toolchains in Hermes tool calls", async () => {
    const fixture = await createHermesFixture([
      toolCallMessage(2, "call-1", "write_file", { path: "web/app.js", content: "console.log('ready')" }),
      toolResultMessage(3, "call-1", { status: "ok" }),
      toolCallMessage(4, "call-2", "terminal", { command: "Add-Type -TypeDefinition 'public class Probe {}'" }),
      toolResultMessage(5, "call-2", { status: "ok" }),
      toolCallMessage(6, "call-3", "terminal", { command: "npm --version" }),
      toolResultMessage(7, "call-3", { status: "ok" }),
      toolCallMessage(8, "call-4", "terminal", { command: "git clone https://github.com/0disoft/velox.git" }),
      toolResultMessage(9, "call-4", { status: "ok" }),
      toolCallMessage(10, "call-5", "terminal", { command: "Get-Command node -ErrorAction SilentlyContinue" }),
      toolResultMessage(11, "call-5", { status: "ok" }),
    ]);
    const result = await createHermesAttestation(fixture.input);
    expect(result.attestation.trajectory.forbiddenActions).toEqual([
      "CONSUMER_COMPILER_INVOKED",
      "NODE_RUNTIME_INVOKED",
      "PACKAGE_MANAGER_INVOKED",
      "SOURCE_CHECKOUT_OBSERVED",
    ]);
  });

  test("detects a later maintainer message and workspace escape", async () => {
    const fixture = await createHermesFixture([
      userMessage(2, "Please finish the two missing files."),
      toolCallMessage(3, "call-1", "read_file", { path: resolve(tmpdir(), "maintainer-only.txt") }),
      toolResultMessage(4, "call-1", { status: "ok" }),
    ]);
    const result = await createHermesAttestation(fixture.input);
    expect(result.attestation.trajectory.forbiddenActions).toEqual([
      "MAINTAINER_HINT_OBSERVED",
      "UNPUBLISHED_CONTEXT_OBSERVED",
    ]);
  });

  test("rejects an attestation output inside the agent trial workspace", async () => {
    const fixture = await createHermesFixture([]);
    fixture.input.outputPath = resolve(fixture.input.trialRoot, `${basename(fixture.input.trialRoot)}.json`);
    await expect(createHermesAttestation(fixture.input)).rejects.toThrow("ATTESTATION_OUTPUT_INSIDE_TRIAL_ROOT");
  });

  test("rejects a Hermes counter that disagrees with persisted tool calls", async () => {
    const fixture = await createHermesFixture([
      toolCallMessage(2, "call-1", "terminal", { command: "velox version" }),
      toolResultMessage(3, "call-1", { status: "ok" }),
    ], { recordedToolCalls: 2 });
    await expect(createHermesAttestation(fixture.input)).rejects.toThrow("HERMES_TOOL_CALL_COUNT_MISMATCH");
  });

  test("rejects an unfinished Hermes session", async () => {
    const fixture = await createHermesFixture([], { endedAt: null });
    await expect(createHermesAttestation(fixture.input)).rejects.toThrow("HERMES_SESSION_NOT_FINISHED");
  });

  test("accepts Hermes completion recorded by a terminal assistant stop message", async () => {
    const finalMessage = { ...message(2, "assistant", "Done."), finishReason: "stop" };
    const fixture = await createHermesFixture([finalMessage], { endedAt: null });
    const result = await createHermesAttestation(fixture.input);
    expect(result.attestation.finishedAtUtc).toBe(new Date(finalMessage.timestamp * 1000).toISOString());
  });

  test("does not treat an assistant tool-call boundary as session completion", async () => {
    const pendingToolCall = { ...toolCallMessage(2, "call-1", "terminal", { command: "Get-Date" }), finishReason: "tool_calls" };
    const fixture = await createHermesFixture([pendingToolCall], { endedAt: null });
    await expect(createHermesAttestation(fixture.input)).rejects.toThrow("HERMES_SESSION_NOT_FINISHED");
  });

  test("rejects a Hermes database controlled by the agent workspace", async () => {
    const fixture = await createHermesFixture([], { stateDatabaseInsideTrial: true });
    await expect(createHermesAttestation(fixture.input)).rejects.toThrow("HERMES_STATE_DB_INSIDE_TRIAL_ROOT");
  });

  test("refuses to replace an existing orchestrator attestation", async () => {
    const fixture = await createHermesFixture([]);
    await writeFile(fixture.input.outputPath, "existing\n", "utf8");
    await expect(createHermesAttestation(fixture.input)).rejects.toThrow("ATTESTATION_OUTPUT_ALREADY_EXISTS");
  });

  test("prepares three isolated trials and binds a session without storing its raw ID", async () => {
    const prepared = await createPreparedSeries();
    expect(prepared.manifest).toMatchObject({
      schemaVersion: "velox.llm-agent-orchestrator/v1",
      seriesId: "series-20260730T010203Z-aaaaaaaa",
      task: { version: "velox.llm-agent-task/v1", sha256: sha(Buffer.from(prompt)) },
      trials: [
        { sequence: 1, trialId: "trial-20260730T010203Z-bbbbbbbb" },
        { sequence: 2, trialId: "trial-20260730T010203Z-cccccccc" },
        { sequence: 3, trialId: "trial-20260730T010203Z-dddddddd" },
      ],
    });
    for (const trial of prepared.manifest.trials) {
      expect((await Bun.file(resolve(prepared.seriesRoot, trial.directory)).stat()).isDirectory()).toBe(true);
    }

    const rawSessionId = "20260730_010203_private";
    const binding = await bindEvaluationSession({ seriesRoot: prepared.seriesRoot, sequence: 1, sessionId: rawSessionId });
    const promptBody = await readFile(binding.promptPath, "utf8");
    const bindingBody = await readFile(binding.bindingPath, "utf8");
    expect(promptBody).toContain(`TRIAL_ID=${binding.trial.trialId}`);
    expect(promptBody).toContain(`SESSION_ID_SHA256=${sha(Buffer.from(rawSessionId))}`);
    expect(promptBody).not.toContain(rawSessionId);
    expect(bindingBody).not.toContain(rawSessionId);
    await expect(bindEvaluationSession({ seriesRoot: prepared.seriesRoot, sequence: 1, sessionId: rawSessionId })).rejects.toThrow("TRIAL_PROMPT_ALREADY_EXISTS");
  });

  test("attests and verifies one immutable prepared three-trial series", async () => {
    const prepared = await createPreparedSeries();
    for (const trial of prepared.manifest.trials) {
      const trialRoot = resolve(prepared.seriesRoot, trial.directory);
      const fixture = await createHermesFixture([], {
        trialRoot,
        model: trial.sequence === 3 ? "fixture-model-b" : "fixture-model-a",
        sessionId: `20260730_01020${trial.sequence}_fixture`,
      });
      await bindEvaluationSession({ seriesRoot: prepared.seriesRoot, sequence: trial.sequence, sessionId: fixture.sessionId });
      const attested = await attestEvaluationTrial({
        seriesRoot: prepared.seriesRoot,
        sequence: trial.sequence,
        sessionId: fixture.sessionId,
        stateDatabasePath: fixture.input.stateDatabasePath,
      });
      const record = await createTrial(trialRoot, trial.sequence, attested.attestation.evaluator.model);
      record.trialId = trial.trialId;
      record.seriesId = prepared.manifest.seriesId;
      record.promptSha256 = prepared.manifest.task.sha256;
      record.evaluator = { ...attested.attestation.evaluator };
      record.startedAtUtc = attested.attestation.startedAtUtc;
      record.finishedAtUtc = attested.attestation.finishedAtUtc;
      record.trajectory.toolCalls = attested.attestation.trajectory.toolCalls;
      record.trajectory.retries = attested.attestation.trajectory.retries;
      await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
    }

    const summary = await verifyEvaluationSeries(prepared.seriesRoot, prepared.taskPath);
    expect(summary).toMatchObject({
      seriesId: prepared.manifest.seriesId,
      passedTrials: 3,
      failedTrials: 0,
      heldTrials: 0,
      betaTechnicalGate: true,
      outcome: "passed",
      modelIdentifiers: ["custom/fixture-model-a", "custom/fixture-model-b"],
    });
    await expect(verifyEvaluationSeries(prepared.seriesRoot, prepared.taskPath)).rejects.toThrow("SERIES_SUMMARY_ALREADY_EXISTS");
  });

  test("rejects a session binding whose immutable trial identity was altered", async () => {
    const prepared = await createPreparedSeries();
    const sessionId = "20260730_010203_private";
    const binding = await bindEvaluationSession({ seriesRoot: prepared.seriesRoot, sequence: 1, sessionId });
    const value = JSON.parse(await readFile(binding.bindingPath, "utf8"));
    value.sequence = 2;
    await writeFile(binding.bindingPath, `${JSON.stringify(value, null, 2)}\n`, "utf8");

    await expect(attestEvaluationTrial({
      seriesRoot: prepared.seriesRoot,
      sequence: 1,
      sessionId,
      stateDatabasePath: resolve(prepared.seriesRoot, "outside-state.db"),
    })).rejects.toThrow("SESSION_BINDING_MANIFEST_MISMATCH");
  });

  test("refuses to prepare agent workspaces inside the Velox repository", async () => {
    const repositoryRoot = resolve(import.meta.dir, "..");
    await expect(prepareEvaluationSeries({
      evaluationRoot: repositoryRoot,
      taskPath: resolve(repositoryRoot, "evals", "llm-agent", "v1", "task.md"),
      taskURL: `https://raw.githubusercontent.com/0disoft/velox/${"a".repeat(40)}/evals/llm-agent/v1/task.md`,
      releaseTag: "v0.5.10-alpha.2",
      releaseURL: "https://github.com/0disoft/velox/releases/download/v0.5.10-alpha.2/velox-windows-x64.zip",
      releaseSha256: fixedDigest,
    })).rejects.toThrow("EVALUATION_ROOT_INSIDE_VELOX_REPOSITORY");
  });

  test("refuses mutable task URLs and release URLs bound to another tag", async () => {
    const root = await createSeries();
    const input = {
      evaluationRoot: root,
      taskPath: resolve(root, "task.md"),
      taskURL: "https://raw.githubusercontent.com/0disoft/velox/main/evals/llm-agent/v1/task.md",
      releaseTag: "v0.5.10-alpha.2",
      releaseURL: "https://github.com/0disoft/velox/releases/download/v0.5.10-alpha.2/velox-windows-x64.zip",
      releaseSha256: fixedDigest,
    };
    await expect(prepareEvaluationSeries(input)).rejects.toThrow("PUBLIC_TASK_URL_NOT_IMMUTABLE");
    input.taskURL = `https://raw.githubusercontent.com/0disoft/velox/${"a".repeat(40)}/evals/llm-agent/v1/task.md`;
    input.releaseURL = "https://github.com/0disoft/velox/releases/download/v0.5.10-alpha.3/velox-windows-x64.zip";
    await expect(prepareEvaluationSeries(input)).rejects.toThrow("RELEASE_URL_TAG_MISMATCH");
  });
});

interface HermesFixtureMessage {
  id: number;
  role: string;
  content: string | null;
  toolCallId: string | null;
  toolCalls: string | null;
  toolName: string | null;
  effectDisposition: string | null;
  timestamp: number;
  finishReason: string | null;
  active: number;
  compacted: number;
  displayKind: string | null;
}

async function createHermesFixture(
  extraMessages: HermesFixtureMessage[],
  options: {
    recordedToolCalls?: number;
    cwd?: string;
    endedAt?: number | null;
    stateDatabaseInsideTrial?: boolean;
    trialRoot?: string;
    model?: string;
    sessionId?: string;
  } = {},
) {
  const root = await mkdtemp(resolve(tmpdir(), "velox-hermes-attestation-"));
  const trialRoot = options.trialRoot ?? resolve(root, "trial-20260722T010101Z-11111111");
  const attestationRoot = resolve(root, "attestations");
  await mkdir(trialRoot, { recursive: true });
  await mkdir(attestationRoot, { recursive: true });
  const stateDatabasePath = resolve(options.stateDatabaseInsideTrial ? trialRoot : root, "state.db");
  const sessionId = options.sessionId ?? "20260722_010101_fixture";
  const messages = [userMessage(1, "Run the public Velox evaluation task."), ...extraMessages];
  const parsedToolCalls = messages.reduce((count, message) => {
    if (!message.toolCalls) return count;
    return count + (JSON.parse(message.toolCalls) as unknown[]).length;
  }, 0);

  const database = new Database(stateDatabasePath, { create: true, strict: true });
  try {
    database.exec(`
      CREATE TABLE sessions (
        id TEXT PRIMARY KEY,
        source TEXT NOT NULL,
        model TEXT,
        parent_session_id TEXT,
        started_at REAL NOT NULL,
        ended_at REAL,
        tool_call_count INTEGER DEFAULT 0,
        cwd TEXT,
        billing_provider TEXT
      );
      CREATE TABLE messages (
        id INTEGER PRIMARY KEY,
        session_id TEXT NOT NULL,
        role TEXT NOT NULL,
        content TEXT,
        tool_call_id TEXT,
        tool_calls TEXT,
        tool_name TEXT,
        effect_disposition TEXT,
        timestamp REAL NOT NULL,
        finish_reason TEXT,
        active INTEGER NOT NULL DEFAULT 1,
        compacted INTEGER NOT NULL DEFAULT 0,
        display_kind TEXT
      );
    `);
    database.query(`
      INSERT INTO sessions (
        id, source, model, parent_session_id, started_at, ended_at, tool_call_count, cwd, billing_provider
      ) VALUES (?1, 'cli', ?2, NULL, 100, ?3, ?4, ?5, 'custom')
    `).run(sessionId, options.model ?? "fixture-model", options.endedAt === undefined ? 120 : options.endedAt, options.recordedToolCalls ?? parsedToolCalls, options.cwd ?? trialRoot);
    const insert = database.query(`
      INSERT INTO messages (
        id, session_id, role, content, tool_call_id, tool_calls, tool_name,
        effect_disposition, timestamp, finish_reason, active, compacted, display_kind
      ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13)
    `);
    for (const message of messages) {
      insert.run(
        message.id,
        sessionId,
        message.role,
        message.content,
        message.toolCallId,
        message.toolCalls,
        message.toolName,
        message.effectDisposition,
        message.timestamp,
        message.finishReason,
        message.active,
        message.compacted,
        message.displayKind,
      );
    }
  } finally {
    database.close(false);
  }

  return {
    sessionId,
    input: {
      stateDatabasePath,
      sessionId,
      trialRoot,
      outputPath: resolve(attestationRoot, `${basename(trialRoot)}.json`),
      trialId: "trial-20260722T010101Z-11111111",
      seriesId: "series-20260722T010100Z-abcdefgh",
      sequence: 1,
    },
  };
}

async function createPreparedSeries() {
  const evaluationRoot = await mkdtemp(resolve(tmpdir(), "velox-agent-series-"));
  const taskPath = resolve(evaluationRoot, "task.md");
  await writeFile(taskPath, prompt, "utf8");
  const prepared = await prepareEvaluationSeries({
    evaluationRoot,
    taskPath,
    taskURL: `https://raw.githubusercontent.com/0disoft/velox/${"a".repeat(40)}/evals/llm-agent/v1/task.md`,
    releaseTag: "v0.5.10-alpha.2",
    releaseURL: "https://github.com/0disoft/velox/releases/download/v0.5.10-alpha.2/velox-windows-x64.zip",
    releaseSha256: fixedDigest,
    now: new Date("2026-07-30T01:02:03Z"),
    suffixes: ["aaaaaaaa", "bbbbbbbb", "cccccccc", "dddddddd"],
  });
  return { ...prepared, taskPath };
}

function userMessage(id: number, content: string): HermesFixtureMessage {
  return message(id, "user", content);
}

function toolCallMessage(id: number, callId: string, name: string, args: Record<string, unknown>): HermesFixtureMessage {
  return {
    ...message(id, "assistant", null),
    toolCalls: JSON.stringify([{ id: callId, type: "function", function: { name, arguments: JSON.stringify(args) } }]),
  };
}

function toolResultMessage(id: number, callId: string, content: Record<string, unknown>): HermesFixtureMessage {
  return {
    ...message(id, "tool", JSON.stringify(content)),
    toolCallId: callId,
  };
}

function message(id: number, role: string, content: string | null): HermesFixtureMessage {
  return {
    id,
    role,
    content,
    toolCallId: null,
    toolCalls: null,
    toolName: null,
    effectDisposition: null,
    timestamp: 100 + id,
    finishReason: null,
    active: 1,
    compacted: 0,
    displayKind: null,
  };
}

async function createSeries() {
  const root = await mkdtemp(resolve(tmpdir(), "velox-llm-eval-"));
  await mkdir(root, { recursive: true });
  await writeFile(resolve(root, "task.md"), prompt, "utf8");
  return root;
}

async function createTrial(root: string, sequence: number, model: string): Promise<TrialRecord> {
  const artifacts = resolve(root, "artifacts");
  await mkdir(artifacts, { recursive: true });
  const archive = Buffer.from("deterministic archive");
  const buildResult = Buffer.from(`${JSON.stringify({
    schemaVersion: "velox.build-result/v1",
    releaseVersion: "0.5.10-alpha.2",
    app: { id: "dev.velox.agent.focusledger", name: "Focus Ledger", version: "0.1.0" },
    target: "windows-x64",
    contracts: { manifest: 1, runtime: 1, host: 1, ipc: 1 },
    host: { file: "velox-host.exe", bytes: 1, sha256: fixedDigest },
    assets: { files: 3, bytes: 10, sha256: fixedDigest },
    permissions: ["app.info", "window.basic"],
    outputs: { portableFiles: 6 },
  })}\n`);
  const report = Buffer.from("# Safe trial report\n\nAll observable checks passed.\n");
  await writeFile(resolve(artifacts, "first.zip"), archive);
  await writeFile(resolve(artifacts, "second.zip"), archive);
  await writeFile(resolve(artifacts, "build-result.json"), buildResult);
  await writeFile(resolve(artifacts, "report.md"), report);
  const archiveDigest = sha(archive);
  const record: TrialRecord = {
    schemaVersion: "velox.llm-agent-evaluation/v1",
    trialId: `trial-20260722T01010${sequence}Z-${String(sequence).repeat(8)}`,
    seriesId: "series-20260722T010100Z-abcdefgh",
    sequence,
    promptVersion: "velox.llm-agent-task/v1",
    promptSha256: sha(Buffer.from(prompt)),
    evaluator: {
      provider: "provider",
      model,
      sessionIdSha256: sha(Buffer.from(`session-${sequence}`)),
      freshSession: true,
      memoryCarryover: false,
    },
    control: {
      maintainerOrchestrated: true,
      externalHuman: false,
      veloxSourceCheckout: false,
      unpublishedContext: false,
      interactiveMaintainerHints: 0,
    },
    release: {
      repository: "0disoft/velox",
      tag: "v0.5.10-alpha.2",
      url: "https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.2",
      expectedSha256: fixedDigest,
      observedSha256: fixedDigest,
    },
    application: {
      id: "dev.velox.agent.focusledger",
      name: "Focus Ledger",
      version: "0.1.0",
    },
    environment: {
      windowsVersion: "Windows fixture",
      webView2Version: "fixture",
      architecture: "AMD64",
      workspaceIsolation: "fresh-local-directory",
    },
    startedAtUtc: `2026-07-22T01:01:0${sequence}Z`,
    finishedAtUtc: `2026-07-22T01:02:0${sequence}Z`,
    outcome: "passed",
    gates: {
      releaseChecksumVerified: true,
      publicDocsOnly: true,
      noSourceCheckout: true,
      noConsumerCompiler: true,
      noNodeRuntime: true,
      noPackageManager: true,
      projectInitialized: true,
      doctorReady: true,
      deterministicBuild: true,
      inspectionPassed: true,
      startupReady: true,
      appBehaviorVerified: true,
      noForbiddenNativeCapability: true,
    },
    trajectory: {
      toolCalls: 12,
      retries: 0,
      commandClasses: ["release-download", "checksum-verification", "init", "doctor", "build", "inspect", "run", "behavior-check"],
      forbiddenActions: [],
    },
    artifacts: {
      firstBuildArchive: "artifacts/first.zip",
      firstBuildSha256: archiveDigest,
      secondBuildArchive: "artifacts/second.zip",
      secondBuildSha256: archiveDigest,
      buildResult: "artifacts/build-result.json",
      buildResultSha256: sha(buildResult),
      safeReport: "artifacts/report.md",
      safeReportSha256: sha(report),
    },
    diagnostics: [],
    failure: null,
    evidenceLevel: "maintainer-orchestrated-clean-room-llm-agent",
    humanAdoptionClaim: false,
  };
  await writeFile(resolve(root, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
  const attestation: TrialAttestation = {
    schemaVersion: "velox.llm-agent-evaluation-attestation/v1",
    trialId: record.trialId,
    seriesId: record.seriesId,
    sequence: record.sequence,
    evaluator: { ...record.evaluator },
    startedAtUtc: record.startedAtUtc,
    finishedAtUtc: record.finishedAtUtc,
    trajectory: {
      toolCalls: record.trajectory.toolCalls,
      retries: record.trajectory.retries,
      toolCallBudget: 70,
      forbiddenActions: [],
    },
    evidence: {
      kind: "orchestrator-session-log",
      sha256: sha(Buffer.from(`session-log-${sequence}`)),
    },
  };
  await writeAttestation(root, attestation);
  return record;
}

async function verifyTrial(seriesRoot: string, trialRoot: string) {
  return loadAndVerifyTrial(resolve(trialRoot, "result.json"), trialRoot, resolve(seriesRoot, "task.md"), attestationPath(trialRoot));
}

function attestationPath(trialRoot: string) {
  return resolve(trialRoot, "..", "attestations", `${basename(trialRoot)}.json`);
}

async function readAttestation(trialRoot: string): Promise<TrialAttestation> {
  return JSON.parse(await Bun.file(attestationPath(trialRoot)).text()) as TrialAttestation;
}

async function writeAttestation(trialRoot: string, attestation: TrialAttestation) {
  const path = attestationPath(trialRoot);
  await mkdir(resolve(path, ".."), { recursive: true });
  await writeFile(path, `${JSON.stringify(attestation, null, 2)}\n`, "utf8");
}

function sha(value: Uint8Array) {
  return createHash("sha256").update(value).digest("hex");
}
