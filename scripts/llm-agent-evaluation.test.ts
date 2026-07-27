import { createHash } from "node:crypto";
import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, resolve } from "node:path";
import { describe, expect, test } from "bun:test";
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
});

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
