import { createHash } from "node:crypto";
import { lstat, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, resolve } from "node:path";
import { Database } from "bun:sqlite";
import { expect, test } from "bun:test";
import { createHermesAttestation } from "./hermes-evaluation-attestation.ts";
import {
  attestSandboxEvaluationTrial,
  attestEvaluationTrial,
  bindEvaluationSession,
  buildSandboxTrialExecutionPlan,
  prepareEvaluationSeries,
  runSandboxEvaluationTrial,
  stageSandboxEvaluationTrial,
  verifyEvaluationSeries,
} from "./llm-agent-orchestrator.ts";
import {
  combineEnforcedSandboxAttestation,
  loadAndVerifyTrial,
  summarizeSeries,
  type TrialAttestation,
  type TrialAttestationV1,
  type TrialAttestationV2,
  type TrialRecord,
} from "./llm-agent-evaluation.ts";

const digest = "a".repeat(64);
const prompt = "public evaluation task\n";
const taskURL = `https://raw.githubusercontent.com/0disoft/velox/${"a".repeat(40)}/evals/llm-agent/v1/task.md`;
const alpha2URL = "https://github.com/0disoft/velox/releases/download/v0.5.10-alpha.2/velox-windows-x64.zip";

for (const scenario of [
  { name: "verifies artifact bytes and summarizes three diverse passing trials", enforced: false, outcome: "held", diagnostics: ["SANDBOX_ENFORCEMENT_UNVERIFIED"] },
  { name: "admits three diverse trials only with enforced sandbox v2 receipts", enforced: true, outcome: "passed", diagnostics: [] },
]) {
  test(scenario.name, async () => {
    expect(summarizeSeries(await verifiedSeries(["model-a", "model-a", "model-b"], { enforced: scenario.enforced }))).toMatchObject({
      passedTrials: 3, failedTrials: 0, heldTrials: 0, outcome: scenario.outcome,
      betaTechnicalGate: scenario.enforced, diagnostics: scenario.diagnostics,
      modelIdentifiers: ["provider/model-a", "provider/model-b"], humanAdoptionClaim: false,
    });
  });
}

for (const scenario of [
  { name: "rejects a self-consistent v2 receipt with an incomplete sandbox grant set", error: "ATTESTATION_SANDBOX_GRANTS_INVALID", mutate: (value: TrialAttestationV2) => { value.evidence.sandbox.receipt.grants = value.evidence.sandbox.receipt.grants.slice(0, 1); } },
  { name: "rejects a sandbox receipt that does not cover the evaluation session", error: "ATTESTATION_SANDBOX_TIME_RANGE_NOT_COVERED", mutate: (value: TrialAttestationV2) => { value.evidence.sandbox.receipt.startedAtUtc = value.finishedAtUtc; } },
  { name: "rejects a sandbox receipt whose prompt was not observed", error: "ATTESTATION_SANDBOX_PROMPT_NOT_OBSERVED", mutate: (value: TrialAttestationV2) => { value.evidence.sandbox.receipt.promptSha256 = "0".repeat(64); } },
]) {
  test(scenario.name, async () => expectCorruptSandboxReceipt(scenario.mutate, scenario.error));
}

test("rejects artifact tampering", async () => {
  const { root, trialRoot } = await basicTrial();
  await writeFile(resolve(trialRoot, "artifacts/first.zip"), "tampered", "utf8");
  await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ARTIFACT_DIGEST_MISMATCH_FIRSTBUILDARCHIVE");
});

for (const scenario of [
  { name: "rejects path traversal before reading an artifact", error: "ARTIFACT_PATH_INVALID_SAFEREPORT", mutate: (record: TrialRecord) => { record.artifacts.safeReport = "../report.md"; } },
  { name: "rejects reuse of one artifact path for both builds", error: "ARTIFACT_PATH_DUPLICATE", mutate: (record: TrialRecord) => { record.artifacts.secondBuildArchive = record.artifacts.firstBuildArchive; } },
  { name: "rejects a passed claim with a failed hard gate", error: "PASSED_TRIAL_HAS_FAILED_GATE", mutate: (record: TrialRecord) => { record.gates.startupReady = false; } },
  { name: "rejects an agent-generated session digest that differs from the orchestrator attestation", error: "ATTESTATION_SESSION_DIGEST_MISMATCH", mutate: (record: TrialRecord) => { record.evaluator.sessionIdSha256 = sha(Buffer.from("invented-session")); } },
  { name: "rejects a reported time range that does not cover the orchestrator session", error: "ATTESTATION_TIME_RANGE_NOT_COVERED", mutate: (record: TrialRecord) => { record.startedAtUtc = record.finishedAtUtc; } },
]) {
  test(scenario.name, async () => {
    const { root, trialRoot, record } = await basicTrial();
    scenario.mutate(record);
    await writeResult(trialRoot, record);
    await expect(verifyTrial(root, trialRoot)).rejects.toThrow(scenario.error);
  });
}

test("rejects a self-consistent digest for the wrong build-result identity", async () => {
  const { root, trialRoot, record } = await basicTrial();
  const wrong = Buffer.from('{"schemaVersion":"not-velox"}\n');
  await writeFile(resolve(trialRoot, "artifacts/build-result.json"), wrong);
  record.artifacts.buildResultSha256 = sha(wrong);
  await writeResult(trialRoot, record);
  await expect(verifyTrial(root, trialRoot)).rejects.toThrow("BUILD_RESULT_SCHEMA_INVALID");
});

test("rejects a passed claim that hides an attested Node.js invocation", async () => {
  const { root, trialRoot } = await basicTrial();
  const attestation = await readAttestation(trialRoot);
  attestation.trajectory.forbiddenActions = ["NODE_RUNTIME_INVOKED"];
  await writeAttestation(trialRoot, attestation);
  await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ATTESTATION_FORBIDDEN_ACTIONS_MISMATCH");
});

test("rejects an under-reported tool-call count", async () => {
  const { root, trialRoot } = await basicTrial();
  const attestation = await readAttestation(trialRoot);
  attestation.trajectory.toolCalls += 1;
  await writeAttestation(trialRoot, attestation);
  await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ATTESTATION_TOOL_CALL_COUNT_MISMATCH");
});

test("rejects an attested tool-call budget overrun", async () => {
  const { root, trialRoot, record } = await basicTrial();
  const attestation = await readAttestation(trialRoot);
  record.trajectory.toolCalls = 71;
  attestation.trajectory.toolCalls = 71;
  await writeResult(trialRoot, record);
  await writeAttestation(trialRoot, attestation);
  await expect(verifyTrial(root, trialRoot)).rejects.toThrow("ATTESTED_TOOL_CALL_BUDGET_EXCEEDED");
});

test("rejects an attestation stored inside the agent-controlled trial root", async () => {
  const { root, trialRoot } = await basicTrial();
  const localAttestation = resolve(trialRoot, "attestation.json");
  await writeFile(localAttestation, `${JSON.stringify(await readAttestation(trialRoot), null, 2)}\n`, "utf8");
  await expect(loadAndVerifyTrial(resolve(trialRoot, "result.json"), trialRoot, resolve(root, "task.md"), localAttestation)).rejects.toThrow("ATTESTATION_INSIDE_TRIAL_ROOT");
});

test("holds an otherwise passing single-model series", async () => {
  expect(summarizeSeries(await verifiedSeries(["model-a", "model-a", "model-a"]))).toMatchObject({
    outcome: "held",
    betaTechnicalGate: false,
    diagnostics: ["MODEL_DIVERSITY_INSUFFICIENT", "SANDBOX_ENFORCEMENT_UNVERIFIED"],
  });
});

test("preserves a failed sequence in the series verdict", async () => {
  const trials = await verifiedSeries(["model-a", "model-a", "model-b"], {
    mutate: async (record, trialRoot, sequence) => {
      if (sequence !== 2) return;
      record.outcome = "failed";
      record.gates.startupReady = false;
      record.failure = { phase: "startup", code: "STARTUP_NOT_READY" };
      await writeResult(trialRoot, record);
    },
  });
  expect(summarizeSeries(trials)).toMatchObject({
    passedTrials: 2,
    failedTrials: 1,
    outcome: "failed",
    betaTechnicalGate: false,
    diagnostics: ["TRIAL_FAILURE_PRESENT", "SANDBOX_ENFORCEMENT_UNVERIFIED"],
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
  expect(JSON.parse(stdout)).toMatchObject({ betaTechnicalGate: false, outcome: "held" });
  expect(JSON.parse(await Bun.file(summaryPath).text())).toMatchObject({
    schemaVersion: "velox.llm-agent-evaluation-series/v1",
    betaTechnicalGate: false,
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

test("combines a finished Hermes session with its supervisor receipt", async () => {
  const fixture = await createHermesFixture([
    toolCallMessage(2, "call-1", "terminal", { command: "velox validate" }),
    toolResultMessage(3, "call-1", { status: "ok" }),
  ]);
  const receiptPath = resolve(fixture.input.trialRoot, "..", "sandbox-receipt.json");
  const receipt = sandboxReceipt({
    trialId: fixture.input.trialId,
    seriesId: fixture.input.seriesId,
    sequence: fixture.input.sequence,
    promptSha256: sha(Buffer.from("Run the public Velox evaluation task.")),
    stateDatabaseSha256: sha(await readFile(fixture.input.stateDatabasePath)),
  });
  await writeFile(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, "utf8");
  const result = await createHermesAttestation({ ...fixture.input, sandboxReceiptPath: receiptPath });
  expect(result.attestation).toMatchObject({
    schemaVersion: "velox.llm-agent-evaluation-attestation/v2",
    evidence: {
      sandboxEnforced: true,
      sandbox: { receipt: { promptSha256: sha(Buffer.from("Run the public Velox evaluation task.")) } },
    },
  });
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

for (const scenario of [
  {
    name: "rejects an attestation output inside the agent trial workspace", error: "ATTESTATION_OUTPUT_INSIDE_TRIAL_ROOT",
    create: async () => { const fixture = await createHermesFixture([]); fixture.input.outputPath = resolve(fixture.input.trialRoot, `${basename(fixture.input.trialRoot)}.json`); return fixture; },
  },
  {
    name: "rejects a Hermes counter that disagrees with persisted tool calls", error: "HERMES_TOOL_CALL_COUNT_MISMATCH",
    create: () => createHermesFixture([toolCallMessage(2, "call-1", "terminal", { command: "velox version" }), toolResultMessage(3, "call-1", { status: "ok" })], { recordedToolCalls: 2 }),
  },
  { name: "rejects an unfinished Hermes session", error: "HERMES_SESSION_NOT_FINISHED", create: () => createHermesFixture([], { endedAt: null }) },
  {
    name: "does not treat an assistant tool-call boundary as session completion", error: "HERMES_SESSION_NOT_FINISHED",
    create: () => createHermesFixture([{ ...toolCallMessage(2, "call-1", "terminal", { command: "Get-Date" }), finishReason: "tool_calls" }], { endedAt: null }),
  },
  { name: "rejects a Hermes database controlled by the agent workspace", error: "HERMES_STATE_DB_INSIDE_TRIAL_ROOT", create: () => createHermesFixture([], { stateDatabaseInsideTrial: true }) },
  {
    name: "refuses to replace an existing orchestrator attestation", error: "ATTESTATION_OUTPUT_ALREADY_EXISTS",
    create: async () => { const fixture = await createHermesFixture([]); await writeFile(fixture.input.outputPath, "existing\n", "utf8"); return fixture; },
  },
]) {
  test(scenario.name, async () => {
    const fixture = await scenario.create();
    await expect(createHermesAttestation(fixture.input)).rejects.toThrow(scenario.error);
  });
}

test("accepts Hermes completion recorded by a terminal assistant stop message", async () => {
  const finalMessage = { ...message(2, "assistant", "Done."), finishReason: "stop" };
  const fixture = await createHermesFixture([finalMessage], { endedAt: null });
  const result = await createHermesAttestation(fixture.input);
  expect(result.attestation.finishedAtUtc).toBe(new Date(finalMessage.timestamp * 1000).toISOString());
});

test("prepares three isolated trials and binds a session without storing its raw ID", async () => {
  const series = await createPreparedSeries();
  expect(series.manifest).toMatchObject({
    schemaVersion: "velox.llm-agent-orchestrator/v1",
    seriesId: "series-20260730T010203Z-aaaaaaaa",
    task: { version: "velox.llm-agent-task/v1", sha256: sha(Buffer.from(prompt)) },
    trials: [
      { sequence: 1, trialId: "trial-20260730T010203Z-bbbbbbbb" },
      { sequence: 2, trialId: "trial-20260730T010203Z-cccccccc" },
      { sequence: 3, trialId: "trial-20260730T010203Z-dddddddd" },
    ],
  });
  for (const trial of series.manifest.trials) {
    expect((await Bun.file(resolve(series.seriesRoot, trial.directory)).stat()).isDirectory()).toBe(true);
  }

  const rawSessionId = "20260730_010203_private";
  const binding = await bindEvaluationSession({ seriesRoot: series.seriesRoot, sequence: 1, sessionId: rawSessionId });
  const promptBody = await readFile(binding.promptPath, "utf8");
  const bindingBody = await readFile(binding.bindingPath, "utf8");
  expect(promptBody).toContain(`TRIAL_ID=${binding.trial.trialId}`);
  expect(promptBody).toContain(`SESSION_ID_SHA256=${sha(Buffer.from(rawSessionId))}`);
  expect(promptBody).not.toContain(rawSessionId);
  expect(bindingBody).not.toContain(rawSessionId);
  await expect(bindEvaluationSession({ seriesRoot: series.seriesRoot, sequence: 1, sessionId: rawSessionId })).rejects.toThrow("TRIAL_PROMPT_ALREADY_EXISTS");
});

test("attests and verifies one immutable prepared three-trial series", async () => {
  const series = await createPreparedSeries();
  for (const trial of series.manifest.trials) {
    const trialRoot = resolve(series.seriesRoot, trial.directory);
    const fixture = await createHermesFixture([], {
      trialRoot,
      model: trial.sequence === 3 ? "fixture-model-b" : "fixture-model-a",
      sessionId: `20260730_01020${trial.sequence}_fixture`,
    });
    await bindEvaluationSession({ seriesRoot: series.seriesRoot, sequence: trial.sequence, sessionId: fixture.sessionId });
    const attested = await attestEvaluationTrial({
      seriesRoot: series.seriesRoot,
      sequence: trial.sequence,
      sessionId: fixture.sessionId,
      stateDatabasePath: fixture.input.stateDatabasePath,
    });
    const record = await createTrial(trialRoot, trial.sequence, attested.attestation.evaluator.model);
    record.trialId = trial.trialId;
    record.seriesId = series.manifest.seriesId;
    record.promptSha256 = series.manifest.task.sha256;
    record.evaluator = { ...attested.attestation.evaluator };
    record.startedAtUtc = attested.attestation.startedAtUtc;
    record.finishedAtUtc = attested.attestation.finishedAtUtc;
    record.trajectory.toolCalls = attested.attestation.trajectory.toolCalls;
    record.trajectory.retries = attested.attestation.trajectory.retries;
    await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
  }

  const summary = await verifyEvaluationSeries(series.seriesRoot, series.taskPath);
  expect(summary).toMatchObject({
    seriesId: series.manifest.seriesId,
    passedTrials: 3,
    failedTrials: 0,
    heldTrials: 0,
    betaTechnicalGate: false,
    outcome: "held",
    diagnostics: ["SANDBOX_ENFORCEMENT_UNVERIFIED"],
    modelIdentifiers: ["custom/fixture-model-a", "custom/fixture-model-b"],
  });
  await expect(verifyEvaluationSeries(series.seriesRoot, series.taskPath)).rejects.toThrow("SERIES_SUMMARY_ALREADY_EXISTS");
});

test("stages and attests a newly created sandbox session without a pre-known ID", async () => {
  const series = await createPreparedSeries();
  const staged = await stageSandboxEvaluationTrial({ seriesRoot: series.seriesRoot, sequence: 1 });
  const promptBody = await readFile(staged.promptPath, "utf8");
  const trialRoot = resolve(series.seriesRoot, staged.trial.directory);
  const fixture = await createHermesFixture([], {
    trialRoot,
    sessionId: "20260816_010203_sandbox",
    initialPrompt: promptBody,
  });
  const receiptPath = resolve(series.seriesRoot, "orchestrator", "sandbox-receipt.json");
  const receipt = sandboxReceipt({
    trialId: staged.trial.trialId,
    seriesId: series.manifest.seriesId,
    sequence: 1,
    promptSha256: sha(Buffer.from(promptBody)),
    stateDatabaseSha256: sha(await readFile(fixture.input.stateDatabasePath)),
  });
  await writeFile(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, "utf8");
  const result = await attestSandboxEvaluationTrial({
    seriesRoot: series.seriesRoot,
    sequence: 1,
    stateDatabasePath: fixture.input.stateDatabasePath,
    sandboxReceiptPath: receiptPath,
  });
  expect(result.attestation).toMatchObject({
    schemaVersion: "velox.llm-agent-evaluation-attestation/v2",
    evaluator: { sessionIdSha256: sha(Buffer.from(fixture.sessionId)) },
    evidence: { sandboxEnforced: true },
  });
});

test("builds a secret-free Hermes sandbox execution plan from explicit inputs", async () => {
  const series = await createPreparedSeries();
  const staged = await stageSandboxEvaluationTrial({ seriesRoot: series.seriesRoot, sequence: 2 });
  const { evaluatorRoot, supervisorPath, evaluatorPath } = await createSandboxTools(series.seriesRoot);

  const plan = await buildSandboxTrialExecutionPlan({
    seriesRoot: series.seriesRoot,
    sequence: 2,
    supervisorPath,
    evaluatorPath,
    evaluatorRoot,
    provider: "fixture-provider",
    model: "fixture/model-b",
    passEnvironment: ["FIXTURE_PROVIDER_KEY"],
    environment: { PATH: "fixture-path", FIXTURE_PROVIDER_KEY: "never-serialize-this-secret" },
  });
  const promptBody = await readFile(staged.promptPath, "utf8");
  expect(plan).toMatchObject({
    trialId: staged.trial.trialId,
    seriesId: series.manifest.seriesId,
    sequence: 2,
    promptPath: staged.promptPath,
  });
  expect(plan.supervisorCommand).toContain(promptBody);
  expect(plan.supervisorCommand).toContain("fixture-provider");
  expect(plan.supervisorCommand).toContain("fixture/model-b");
  expect(plan.supervisorCommand).toContain("FIXTURE_PROVIDER_KEY");
  expect(plan.supervisorCommand).not.toContain("never-serialize-this-secret");
  expect(plan.supervisorCommand.slice(-5)).toEqual([
    "--reasoning", "high", "--pass-session-id", "--ignore-user-config", "--ignore-rules",
  ]);
});

test("runs attestation and removes the temporary Hermes database after supervisor success", async () => {
  const series = await createPreparedSeries();
  const staged = await stageSandboxEvaluationTrial({ seriesRoot: series.seriesRoot, sequence: 1 });
  const { evaluatorRoot, supervisorPath, evaluatorPath } = await createSandboxTools(series.seriesRoot);

  const result = await runSandboxEvaluationTrial({
    seriesRoot: series.seriesRoot,
    sequence: 1,
    supervisorPath,
    evaluatorPath,
    evaluatorRoot,
    provider: "fixture-provider",
    model: "fixture/model-a",
    passEnvironment: ["FIXTURE_PROVIDER_KEY"],
    environment: { PATH: "fixture-path", FIXTURE_PROVIDER_KEY: "never-serialize-this-secret" },
  }, async (plan) => {
    const promptBody = await readFile(plan.promptPath, "utf8");
    await createHermesFixture([], {
      trialRoot: plan.trialRoot,
      sessionId: "20260831_010203_sandbox",
      initialPrompt: promptBody,
      stateDatabasePath: plan.stateDatabasePath,
    });
    await writeFile(plan.sandboxReceiptPath, `${JSON.stringify(sandboxReceipt({
      trialId: plan.trialId,
      seriesId: plan.seriesId,
      sequence: plan.sequence,
      promptSha256: sha(Buffer.from(promptBody)),
      stateDatabaseSha256: sha(await readFile(plan.stateDatabasePath)),
      supervisorVersion: "0.5.10-alpha.36",
    }), null, 2)}\n`, "utf8");
    return 0;
  });

  expect(result.attestation).toMatchObject({
    schemaVersion: "velox.llm-agent-evaluation-attestation/v2",
    trialId: staged.trial.trialId,
    evidence: { sandboxEnforced: true },
  });
  await expect(lstat(result.plan.stateDatabasePath)).rejects.toThrow();
  expect(await Bun.file(result.plan.sandboxReceiptPath).exists()).toBe(true);
  expect(await Bun.file(resolve(series.seriesRoot, "orchestrator", "bindings", `${staged.trial.trialId}.json`)).exists()).toBe(true);
  expect(await Bun.file(resolve(series.seriesRoot, "orchestrator", "attestations", `${staged.trial.trialId}.json`)).exists()).toBe(true);
  await writeFile(result.plan.stateDatabasePath, "retained state", "utf8");
  await expect(verifyEvaluationSeries(series.seriesRoot, series.taskPath)).rejects.toThrow("SERIES_TEMPORARY_STATE_RETAINED");
  await rm(result.plan.stateDatabasePath);
});

test("rejects a session binding whose immutable trial identity was altered", async () => {
  const series = await createPreparedSeries();
  const sessionId = "20260730_010203_private";
  const binding = await bindEvaluationSession({ seriesRoot: series.seriesRoot, sequence: 1, sessionId });
  const value = JSON.parse(await readFile(binding.bindingPath, "utf8"));
  value.sequence = 2;
  await writeFile(binding.bindingPath, `${JSON.stringify(value, null, 2)}\n`, "utf8");

  await expect(attestEvaluationTrial({
    seriesRoot: series.seriesRoot,
    sequence: 1,
    sessionId,
    stateDatabasePath: resolve(series.seriesRoot, "outside-state.db"),
  })).rejects.toThrow("SESSION_BINDING_MANIFEST_MISMATCH");
});

test("refuses to prepare agent workspaces inside the Velox repository", async () => {
  const repositoryRoot = resolve(import.meta.dir, "..");
  await expect(prepareEvaluationSeries({
    evaluationRoot: repositoryRoot,
    taskPath: resolve(repositoryRoot, "evals", "llm-agent", "v1", "task.md"),
    taskURL,
    releaseTag: "v0.5.10-alpha.2",
    releaseURL: alpha2URL,
    releaseSha256: digest,
  })).rejects.toThrow("EVALUATION_ROOT_INSIDE_VELOX_REPOSITORY");
});

test("refuses mutable task URLs and release URLs bound to another tag", async () => {
  const root = await createSeries();
  const input = {
    evaluationRoot: root,
    taskPath: resolve(root, "task.md"),
    taskURL: "https://raw.githubusercontent.com/0disoft/velox/main/evals/llm-agent/v1/task.md",
    releaseTag: "v0.5.10-alpha.2",
    releaseURL: alpha2URL,
    releaseSha256: digest,
  };
  await expect(prepareEvaluationSeries(input)).rejects.toThrow("PUBLIC_TASK_URL_NOT_IMMUTABLE");
  input.taskURL = taskURL;
  input.releaseURL = "https://github.com/0disoft/velox/releases/download/v0.5.10-alpha.3/velox-windows-x64.zip";
  await expect(prepareEvaluationSeries(input)).rejects.toThrow("RELEASE_URL_TAG_MISMATCH");
});

type Row = ReturnType<typeof message>;

async function createHermesFixture(
  extraMessages: Row[],
  options: {
    recordedToolCalls?: number; cwd?: string; endedAt?: number | null; stateDatabaseInsideTrial?: boolean;
    trialRoot?: string; model?: string; sessionId?: string; initialPrompt?: string; stateDatabasePath?: string;
  } = {},
) {
  const root = await mkdtemp(resolve(tmpdir(), "velox-hermes-attestation-"));
  const trialRoot = options.trialRoot ?? resolve(root, "trial-20260722T010101Z-11111111");
  const attestationRoot = resolve(root, "attestations");
  await mkdir(trialRoot, { recursive: true });
  await mkdir(attestationRoot, { recursive: true });
  const stateDatabasePath = options.stateDatabasePath ?? resolve(options.stateDatabaseInsideTrial ? trialRoot : root, "state.db");
  const sessionId = options.sessionId ?? "20260722_010101_fixture";
  const messages = [userMessage(1, options.initialPrompt ?? "Run the public Velox evaluation task."), ...extraMessages];
  const parsedToolCalls = messages.reduce((count, message) => {
    if (!message.toolCalls) return count;
    return count + (JSON.parse(message.toolCalls) as unknown[]).length;
  }, 0);

  const database = new Database(stateDatabasePath, { create: true, strict: true });
  try {
    database.exec(`
      CREATE TABLE sessions (id TEXT PRIMARY KEY, source TEXT NOT NULL, model TEXT, parent_session_id TEXT,
        started_at REAL NOT NULL, ended_at REAL, tool_call_count INTEGER DEFAULT 0, cwd TEXT, billing_provider TEXT);
      CREATE TABLE messages (id INTEGER PRIMARY KEY, session_id TEXT NOT NULL, role TEXT NOT NULL, content TEXT,
        tool_call_id TEXT, tool_calls TEXT, tool_name TEXT, effect_disposition TEXT, timestamp REAL NOT NULL,
        finish_reason TEXT, active INTEGER NOT NULL DEFAULT 1, compacted INTEGER NOT NULL DEFAULT 0, display_kind TEXT);
    `);
    database.query(`INSERT INTO sessions (id, source, model, parent_session_id, started_at, ended_at, tool_call_count, cwd, billing_provider)
      VALUES (?1, 'cli', ?2, NULL, 100, ?3, ?4, ?5, 'custom')`).run(sessionId, options.model ?? "fixture-model", options.endedAt === undefined ? 120 : options.endedAt, options.recordedToolCalls ?? parsedToolCalls, options.cwd ?? trialRoot);
    const insert = database.query(`
      INSERT INTO messages (id, session_id, role, content, tool_call_id, tool_calls, tool_name,
        effect_disposition, timestamp, finish_reason, active, compacted, display_kind)
      VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13)
    `);
    for (const message of messages) {
      insert.run(message.id, sessionId, message.role, message.content, message.toolCallId, message.toolCalls,
        message.toolName, message.effectDisposition, message.timestamp, message.finishReason, message.active, message.compacted, message.displayKind);
    }
  } finally {
    database.close(false);
  }

  return { sessionId, input: {
    stateDatabasePath, sessionId, trialRoot, outputPath: resolve(attestationRoot, `${basename(trialRoot)}.json`),
    trialId: "trial-20260722T010101Z-11111111", seriesId: "series-20260722T010100Z-abcdefgh", sequence: 1,
  } };
}

async function createPreparedSeries() {
  const evaluationRoot = await mkdtemp(resolve(tmpdir(), "velox-agent-series-"));
  const taskPath = resolve(evaluationRoot, "task.md");
  await writeFile(taskPath, prompt, "utf8");
  const series = await prepareEvaluationSeries({
    evaluationRoot, taskPath, taskURL, releaseTag: "v0.5.10-alpha.2", releaseURL: alpha2URL,
    releaseSha256: digest, now: new Date("2026-07-30T01:02:03Z"), suffixes: ["aaaaaaaa", "bbbbbbbb", "cccccccc", "dddddddd"],
  });
  return { ...series, taskPath };
}

async function createSandboxTools(seriesRoot: string) {
  const toolRoot = resolve(seriesRoot, "maintainer-tools");
  const evaluatorRoot = resolve(toolRoot, "hermes-agent");
  const supervisorPath = resolve(toolRoot, "velox-eval-sandbox.exe");
  const evaluatorPath = resolve(evaluatorRoot, "venv", "Scripts", "hermes.exe");
  await mkdir(resolve(evaluatorPath, ".."), { recursive: true });
  await writeFile(supervisorPath, "supervisor fixture", "utf8");
  await writeFile(evaluatorPath, "evaluator fixture", "utf8");
  return { evaluatorRoot, supervisorPath, evaluatorPath };
}

function userMessage(id: number, content: string): Row {
  return message(id, "user", content);
}

function toolCallMessage(id: number, callId: string, name: string, args: Record<string, unknown>): Row {
  return {
    ...message(id, "assistant", null),
    toolCalls: JSON.stringify([{ id: callId, type: "function", function: { name, arguments: JSON.stringify(args) } }]),
  };
}

function toolResultMessage(id: number, callId: string, content: Record<string, unknown>): Row {
  return {
    ...message(id, "tool", JSON.stringify(content)),
    toolCallId: callId,
  };
}

function message(id: number, role: string, content: string | null): Row {
  return {
    id, role, content, toolCallId: null, toolCalls: null, toolName: null, effectDisposition: null,
    timestamp: 100 + id, finishReason: null, active: 1, compacted: 0, displayKind: null,
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
    schemaVersion: "velox.build-result/v1", releaseVersion: "0.5.10-alpha.2", target: "windows-x64",
    app: { id: "dev.velox.agent.focusledger", name: "Focus Ledger", version: "0.1.0" },
    contracts: { manifest: 1, runtime: 1, host: 1, ipc: 1 }, host: { file: "velox-host.exe", bytes: 1, sha256: digest },
    assets: { files: 3, bytes: 10, sha256: digest }, permissions: ["app.info", "window.basic"], outputs: { portableFiles: 6 },
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
    evaluator: { provider: "provider", model, sessionIdSha256: sha(Buffer.from(`session-${sequence}`)), freshSession: true, memoryCarryover: false },
    control: { maintainerOrchestrated: true, externalHuman: false, veloxSourceCheckout: false, unpublishedContext: false, interactiveMaintainerHints: 0 },
    release: { repository: "0disoft/velox", tag: "v0.5.10-alpha.2", url: "https://github.com/0disoft/velox/releases/tag/v0.5.10-alpha.2", expectedSha256: digest, observedSha256: digest },
    application: { id: "dev.velox.agent.focusledger", name: "Focus Ledger", version: "0.1.0" },
    environment: { windowsVersion: "Windows fixture", webView2Version: "fixture", architecture: "AMD64", workspaceIsolation: "fresh-local-directory" },
    startedAtUtc: `2026-07-22T01:01:0${sequence}Z`,
    finishedAtUtc: `2026-07-22T01:02:0${sequence}Z`,
    outcome: "passed",
    gates: passingGates(),
    trajectory: {
      toolCalls: 12,
      retries: 0,
      commandClasses: ["release-download", "checksum-verification", "init", "doctor", "build", "inspect", "run", "behavior-check"],
      forbiddenActions: [],
    },
    artifacts: { firstBuildArchive: "artifacts/first.zip", firstBuildSha256: archiveDigest, secondBuildArchive: "artifacts/second.zip", secondBuildSha256: archiveDigest, buildResult: "artifacts/build-result.json", buildResultSha256: sha(buildResult), safeReport: "artifacts/report.md", safeReportSha256: sha(report) },
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
    trajectory: { toolCalls: record.trajectory.toolCalls, retries: record.trajectory.retries, toolCallBudget: 70, forbiddenActions: [] },
    evidence: { kind: "orchestrator-session-log", observationLevel: "session-log-heuristic", sandboxEnforced: false, sha256: "", projection: { schemaVersion: "velox.hermes-session-log-digest/v1", sessions: [], messages: [{ contentSha256: "9".repeat(64) }] } },
  };
  attestation.evidence.sha256 = sha(Buffer.from(JSON.stringify(attestation.evidence.projection)));
  await writeAttestation(root, attestation);
  return record;
}

async function verifyTrial(seriesRoot: string, trialRoot: string) {
  return loadAndVerifyTrial(resolve(trialRoot, "result.json"), trialRoot, resolve(seriesRoot, "task.md"), attestationPath(trialRoot));
}

async function basicTrial() {
  const root = await createSeries();
  const trialRoot = resolve(root, "trial-1");
  const record = await createTrial(trialRoot, 1, "model-a");
  return { root, trialRoot, record };
}

async function verifiedSeries(
  models: [string, string, string],
  options: { enforced?: boolean; mutate?: (record: TrialRecord, trialRoot: string, sequence: number) => Promise<void> } = {},
) {
  const root = await createSeries();
  const trials = [];
  for (const [index, model] of models.entries()) {
    const sequence = index + 1;
    const trialRoot = resolve(root, `trial-${sequence}`);
    const record = await createTrial(trialRoot, sequence, model);
    if (options.enforced) await upgradeAttestationToV2(trialRoot);
    await options.mutate?.(record, trialRoot, sequence);
    const observed = await verifyTrial(root, trialRoot);
    expect(observed.trialId).toBe(record.trialId);
    trials.push(observed);
  }
  return trials;
}

async function writeResult(trialRoot: string, record: TrialRecord) {
  await writeFile(resolve(trialRoot, "result.json"), `${JSON.stringify(record, null, 2)}\n`, "utf8");
}

async function expectCorruptSandboxReceipt(mutate: (value: TrialAttestationV2) => void, error: string) {
  const { root, trialRoot } = await basicTrial();
  const attestation = await upgradeAttestationToV2(trialRoot);
  mutate(attestation);
  attestation.evidence.sandbox.receiptSha256 = sha(Buffer.from(JSON.stringify(attestation.evidence.sandbox.receipt)));
  await writeAttestation(trialRoot, attestation);
  await expect(verifyTrial(root, trialRoot)).rejects.toThrow(error);
}

function passingGates(): TrialRecord["gates"] {
  return {
    releaseChecksumVerified: true, publicDocsOnly: true, noSourceCheckout: true,
    noConsumerCompiler: true, noNodeRuntime: true, noPackageManager: true,
    projectInitialized: true, doctorReady: true, deterministicBuild: true,
    inspectionPassed: true, startupReady: true, appBehaviorVerified: true,
    noForbiddenNativeCapability: true,
  };
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

async function upgradeAttestationToV2(trialRoot: string): Promise<TrialAttestationV2> {
  const current = await readAttestation(trialRoot) as TrialAttestationV1;
  const receipt = sandboxReceipt({
    trialId: current.trialId,
    seriesId: current.seriesId,
    sequence: current.sequence,
    promptSha256: "9".repeat(64),
    stateDatabaseSha256: "8".repeat(64),
    startedAtUtc: new Date(Date.parse(current.startedAtUtc) - 1000).toISOString(),
    finishedAtUtc: new Date(Date.parse(current.finishedAtUtc) + 1000).toISOString(),
  });
  const attestation = combineEnforcedSandboxAttestation(current, receipt, "8".repeat(64));
  await writeAttestation(trialRoot, attestation);
  return attestation;
}

function sha(value: Uint8Array) {
  return createHash("sha256").update(value).digest("hex");
}

function sandboxReceipt(input: {
  trialId: string; seriesId: string; sequence: number; promptSha256: string; stateDatabaseSha256: string;
  supervisorVersion?: string; startedAtUtc?: string; finishedAtUtc?: string;
}) {
  return {
    schemaVersion: "velox.eval-sandbox-receipt/v1" as const,
    trialId: input.trialId, seriesId: input.seriesId, sequence: input.sequence,
    policy: {
      schemaVersion: "velox.eval-sandbox-policy/v1" as const, platform: "windows" as const,
      filesystemBoundary: "appcontainer-explicit-acl" as const, processBoundary: "job-object-no-breakaway" as const,
      networkCapability: "internet-client" as const,
    },
    supervisor: { version: input.supervisorVersion ?? "0.5.10-alpha.34", sha256: "b".repeat(64) },
    commandSha256: "c".repeat(64), environmentSha256: "f".repeat(64),
    promptSha256: input.promptSha256, stateDatabaseSha256: input.stateDatabaseSha256,
    startedAtUtc: input.startedAtUtc ?? "1970-01-01T00:01:30.000Z",
    finishedAtUtc: input.finishedAtUtc ?? "1970-01-01T00:03:00.000Z",
    exitCode: 0 as const, timedOut: false as const,
    containment: { filesystemEnforced: true as const, processTreeEnforced: true as const, cleanupCompleted: true as const },
    grants: [
      { role: "trial-read-write-execute" as const, pathSha256: "d".repeat(64), rights: "read-write-execute" as const },
      { role: "tool-read-execute" as const, pathSha256: "e".repeat(64), rights: "read-execute" as const },
    ],
  };
}
