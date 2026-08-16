import { createHash, randomBytes } from "node:crypto";
import { lstat, mkdir, readFile, realpath, rm, writeFile } from "node:fs/promises";
import { basename, isAbsolute, resolve, sep } from "node:path";
import { createHermesAttestation, inspectHermesSessionCompletion } from "./hermes-evaluation-attestation.ts";
import { loadAndVerifyTrial, summarizeSeries, type SeriesSummary, type TrialRecord } from "./llm-agent-evaluation.ts";

const sha256Pattern = /^[0-9a-f]{64}$/;
const trialIDPattern = /^trial-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$/;
const seriesIDPattern = /^series-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$/;
const releaseTagPattern = /^v[0-9]+\.[0-9]+\.[0-9]+-(?:alpha|beta)\.[1-9][0-9]*$/;
const promptVersion = "velox.llm-agent-task/v1" as const;

export interface EvaluationSeriesManifest {
  schemaVersion: "velox.llm-agent-orchestrator/v1";
  seriesId: string;
  createdAtUtc: string;
  task: {
    version: typeof promptVersion;
    url: string;
    sha256: string;
  };
  release: {
    tag: string;
    url: string;
    sha256: string;
  };
  trials: Array<{
    sequence: number;
    trialId: string;
    directory: string;
  }>;
}

export interface PrepareSeriesInput {
  evaluationRoot: string;
  taskPath: string;
  taskURL: string;
  releaseTag: string;
  releaseURL: string;
  releaseSha256: string;
  now?: Date;
  suffixes?: [string, string, string, string];
}

export interface BindSessionInput {
  seriesRoot: string;
  sequence: number;
  sessionId: string;
}

export interface AttestTrialInput extends BindSessionInput {
  stateDatabasePath: string;
  sandboxReceiptPath?: string;
}

export async function prepareEvaluationSeries(input: PrepareSeriesInput) {
  validatePrepareInput(input);
  const evaluationRoot = await realDirectory(input.evaluationRoot, "EVALUATION_ROOT_INVALID");
  const repositoryRoot = await realpath(resolve(import.meta.dir, ".."));
  if (isContained(repositoryRoot, evaluationRoot)) fail("EVALUATION_ROOT_INSIDE_VELOX_REPOSITORY");

  const taskPath = await realFile(input.taskPath, "PUBLIC_TASK_FILE_INVALID", 1024 * 1024);
  const taskSha256 = sha256(await readFile(taskPath));
  const now = input.now ?? new Date();
  if (!Number.isFinite(now.getTime())) fail("SERIES_TIME_INVALID");
  const timestamp = idTimestamp(now);
  const suffixes = input.suffixes ?? [randomSuffix(), randomSuffix(), randomSuffix(), randomSuffix()];
  if (new Set(suffixes).size !== 4 || suffixes.some((suffix) => !/^[a-z0-9]{8}$/.test(suffix))) {
    fail("SERIES_SUFFIXES_INVALID");
  }

  const seriesId = `series-${timestamp}-${suffixes[0]}`;
  const seriesRoot = resolve(evaluationRoot, seriesId);
  const manifest: EvaluationSeriesManifest = {
    schemaVersion: "velox.llm-agent-orchestrator/v1",
    seriesId,
    createdAtUtc: now.toISOString(),
    task: { version: promptVersion, url: input.taskURL, sha256: taskSha256 },
    release: { tag: input.releaseTag, url: input.releaseURL, sha256: input.releaseSha256 },
    trials: [1, 2, 3].map((sequence) => {
      const trialId = `trial-${timestamp}-${suffixes[sequence]}`;
      return { sequence, trialId, directory: trialId };
    }),
  };

  await mkdir(seriesRoot);
  try {
    await mkdir(resolve(seriesRoot, "orchestrator", "prompts"), { recursive: true });
    await mkdir(resolve(seriesRoot, "orchestrator", "bindings"), { recursive: true });
    await mkdir(resolve(seriesRoot, "orchestrator", "attestations"), { recursive: true });
    for (const trial of manifest.trials) await mkdir(resolve(seriesRoot, trial.directory));
    await writeExclusiveJSON(resolve(seriesRoot, "orchestrator", "series.json"), manifest);
  } catch (error) {
    await rm(seriesRoot, { recursive: true, force: true });
    throw error;
  }
  return { seriesRoot, manifest };
}

export async function bindEvaluationSession(input: BindSessionInput) {
  if (!input.sessionId.trim()) fail("HERMES_SESSION_ID_REQUIRED");
  const { seriesRoot, manifest } = await loadSeries(input.seriesRoot);
  const trial = selectTrial(manifest, input.sequence);
  const sessionIdSha256 = sha256(input.sessionId);
  const promptPath = resolve(seriesRoot, "orchestrator", "prompts", `${trial.trialId}.txt`);
  const bindingPath = resolve(seriesRoot, "orchestrator", "bindings", `${trial.trialId}.json`);
  await requireAbsent(promptPath, "TRIAL_PROMPT_ALREADY_EXISTS");
  await requireAbsent(bindingPath, "TRIAL_BINDING_ALREADY_EXISTS");

  const prompt = renderTrialPrompt(manifest, trial, sessionIdSha256);
  await writeExclusive(promptPath, prompt);
  try {
    await writeExclusiveJSON(bindingPath, {
      schemaVersion: "velox.llm-agent-session-binding/v1",
      seriesId: manifest.seriesId,
      trialId: trial.trialId,
      sequence: trial.sequence,
      sessionIdSha256,
    });
  } catch (error) {
    await rm(promptPath, { force: true });
    throw error;
  }
  return { trial, promptPath, bindingPath, sessionIdSha256 };
}

export async function attestEvaluationTrial(input: AttestTrialInput) {
  if (!isAbsolute(input.stateDatabasePath)) fail("HERMES_STATE_DB_MUST_BE_ABSOLUTE");
  if (!input.sessionId.trim()) fail("HERMES_SESSION_ID_REQUIRED");
  const { seriesRoot, manifest } = await loadSeries(input.seriesRoot);
  const trial = selectTrial(manifest, input.sequence);
  const binding = await readBinding(seriesRoot, trial.trialId);
  verifyBinding(binding, manifest, trial);
  if (binding.sessionIdSha256 !== sha256(input.sessionId)) fail("SESSION_BINDING_DIGEST_MISMATCH");

  const trialRoot = await realDirectory(resolve(seriesRoot, trial.directory), "TRIAL_ROOT_INVALID");
  const outputPath = resolve(seriesRoot, "orchestrator", "attestations", `${trial.trialId}.json`);
  return createHermesAttestation({
    stateDatabasePath: input.stateDatabasePath,
    sessionId: input.sessionId,
    trialRoot,
    outputPath,
    trialId: trial.trialId,
    seriesId: manifest.seriesId,
    sequence: trial.sequence,
    sandboxReceiptPath: input.sandboxReceiptPath,
  });
}

export async function verifyEvaluationSeries(seriesPath: string, taskPath: string): Promise<SeriesSummary> {
  const { seriesRoot, manifest } = await loadSeries(seriesPath);
  const observedTask = await realFile(taskPath, "PUBLIC_TASK_FILE_INVALID", 1024 * 1024);
  if (sha256(await readFile(observedTask)) !== manifest.task.sha256) fail("SERIES_TASK_DIGEST_MISMATCH");

  const trials: TrialRecord[] = [];
  for (const trial of manifest.trials) {
    const binding = await readBinding(seriesRoot, trial.trialId);
    verifyBinding(binding, manifest, trial);
    const trialRoot = await realDirectory(resolve(seriesRoot, trial.directory), "TRIAL_ROOT_INVALID");
    const attestationPath = resolve(seriesRoot, "orchestrator", "attestations", `${trial.trialId}.json`);
    const result = await loadAndVerifyTrial(resolve(trialRoot, "result.json"), trialRoot, observedTask, attestationPath);
    if (result.trialId !== trial.trialId || result.sequence !== trial.sequence) fail("SERIES_TRIAL_MANIFEST_MISMATCH");
    trials.push(result);
  }

  const summary = summarizeSeries(trials);
  if (summary.seriesId !== manifest.seriesId) fail("SERIES_SUMMARY_ID_MISMATCH");
  const summaryPath = resolve(seriesRoot, "summary.json");
  await requireAbsent(summaryPath, "SERIES_SUMMARY_ALREADY_EXISTS");
  await writeExclusiveJSON(summaryPath, summary);
  return summary;
}

export async function runLiveHermesSmoke(environment: NodeJS.ProcessEnv = process.env) {
  const localAppData = requiredEnvironment(environment, "LOCALAPPDATA");
  const sessionId = requiredEnvironment(environment, "VELOX_HERMES_SESSION_ID");
  const trialRoot = requiredEnvironment(environment, "VELOX_HERMES_TRIAL_ROOT");
  const trialId = requiredEnvironment(environment, "VELOX_HERMES_TRIAL_ID");
  const seriesId = requiredEnvironment(environment, "VELOX_HERMES_SERIES_ID");
  const sequence = Number(requiredEnvironment(environment, "VELOX_HERMES_SEQUENCE"));
  const repositoryRoot = await realpath(resolve(import.meta.dir, ".."));
  const outputRoot = resolve(repositoryRoot, ".cache", "hermes-attestation-smoke", `${idTimestamp(new Date())}-${randomSuffix()}`);
  await mkdir(outputRoot, { recursive: true });
  try {
    const result = await createHermesAttestation({
      stateDatabasePath: resolve(localAppData, "hermes", "state.db"),
      sessionId,
      trialRoot,
      outputPath: resolve(outputRoot, `${basename(resolve(trialRoot))}.json`),
      trialId,
      seriesId,
      sequence,
    });
    return { ...result, outputRoot };
  } catch (error) {
    await rm(outputRoot, { recursive: true, force: true });
    throw error;
  }
}

function renderTrialPrompt(
  manifest: EvaluationSeriesManifest,
  trial: EvaluationSeriesManifest["trials"][number],
  sessionIdSha256: string,
) {
  return `Run the public Velox clean-room evaluation task from the exact URL below.\n\n` +
    `TASK_URL=${manifest.task.url}\n` +
    `PROMPT_VERSION=${manifest.task.version}\n` +
    `PROMPT_SHA256=${manifest.task.sha256}\n` +
    `RELEASE_TAG=${manifest.release.tag}\n` +
    `RELEASE_URL=${manifest.release.url}\n` +
    `RELEASE_SHA256=${manifest.release.sha256}\n` +
    `RESULT_DIRECTORY=.\n` +
    `TRIAL_ID=${trial.trialId}\n` +
    `SERIES_ID=${manifest.seriesId}\n` +
    `SEQUENCE=${trial.sequence}\n` +
    `SESSION_ID_SHA256=${sessionIdSha256}\n` +
    `TOOL_CALL_BUDGET=70\n\n` +
    `Verify the task SHA-256 before following it. Work only in the current fresh trial directory. ` +
    `Do not read a local maintainer checkout or use an editor action that implicitly launches a forbidden runtime, package manager, or compiler.\n`;
}

async function loadSeries(seriesPath: string) {
  const seriesRoot = await realDirectory(seriesPath, "SERIES_ROOT_INVALID");
  const manifestPath = resolve(seriesRoot, "orchestrator", "series.json");
  const manifest = validateManifest(parseJSON(await readFile(manifestPath, "utf8"), "SERIES_MANIFEST_JSON_INVALID"));
  if (basename(seriesRoot) !== manifest.seriesId) fail("SERIES_DIRECTORY_ID_MISMATCH");
  return { seriesRoot, manifest };
}

function validateManifest(value: unknown): EvaluationSeriesManifest {
  const manifest = object(value, "SERIES_MANIFEST_INVALID");
  exactKeys(manifest, ["schemaVersion", "seriesId", "createdAtUtc", "task", "release", "trials"], "SERIES_MANIFEST_INVALID");
  equal(manifest.schemaVersion, "velox.llm-agent-orchestrator/v1", "SERIES_MANIFEST_VERSION_INVALID");
  stringMatch(manifest.seriesId, seriesIDPattern, "SERIES_ID_INVALID");
  dateTime(manifest.createdAtUtc, "SERIES_CREATED_AT_INVALID");

  const task = object(manifest.task, "SERIES_TASK_INVALID");
  exactKeys(task, ["version", "url", "sha256"], "SERIES_TASK_INVALID");
  equal(task.version, promptVersion, "SERIES_TASK_VERSION_INVALID");
  httpsURL(task.url, "SERIES_TASK_URL_INVALID");
  stringMatch(task.sha256, sha256Pattern, "SERIES_TASK_DIGEST_INVALID");

  const release = object(manifest.release, "SERIES_RELEASE_INVALID");
  exactKeys(release, ["tag", "url", "sha256"], "SERIES_RELEASE_INVALID");
  stringMatch(release.tag, releaseTagPattern, "SERIES_RELEASE_TAG_INVALID");
  httpsURL(release.url, "SERIES_RELEASE_URL_INVALID");
  stringMatch(release.sha256, sha256Pattern, "SERIES_RELEASE_DIGEST_INVALID");

  if (!Array.isArray(manifest.trials) || manifest.trials.length !== 3) fail("SERIES_TRIALS_INVALID");
  const trials = manifest.trials.map((raw, index) => {
    const trial = object(raw, "SERIES_TRIAL_INVALID");
    exactKeys(trial, ["sequence", "trialId", "directory"], "SERIES_TRIAL_INVALID");
    equal(trial.sequence, index + 1, "SERIES_TRIAL_SEQUENCE_INVALID");
    stringMatch(trial.trialId, trialIDPattern, "SERIES_TRIAL_ID_INVALID");
    equal(trial.directory, trial.trialId, "SERIES_TRIAL_DIRECTORY_INVALID");
    return trial as unknown as EvaluationSeriesManifest["trials"][number];
  });
  if (new Set(trials.map((trial) => trial.trialId)).size !== 3) fail("SERIES_TRIAL_ID_DUPLICATE");
  return { ...manifest, task, release, trials } as unknown as EvaluationSeriesManifest;
}

async function readBinding(seriesRoot: string, trialId: string) {
  const value = object(parseJSON(
    await readFile(resolve(seriesRoot, "orchestrator", "bindings", `${trialId}.json`), "utf8"),
    "SESSION_BINDING_JSON_INVALID",
  ), "SESSION_BINDING_INVALID");
  exactKeys(value, ["schemaVersion", "seriesId", "trialId", "sequence", "sessionIdSha256"], "SESSION_BINDING_INVALID");
  equal(value.schemaVersion, "velox.llm-agent-session-binding/v1", "SESSION_BINDING_VERSION_INVALID");
  stringMatch(value.sessionIdSha256, sha256Pattern, "SESSION_BINDING_DIGEST_INVALID");
  return value as unknown as {
    schemaVersion: "velox.llm-agent-session-binding/v1";
    seriesId: string;
    trialId: string;
    sequence: number;
    sessionIdSha256: string;
  };
}

function verifyBinding(
  binding: Awaited<ReturnType<typeof readBinding>>,
  manifest: EvaluationSeriesManifest,
  trial: EvaluationSeriesManifest["trials"][number],
) {
  if (
    binding.seriesId !== manifest.seriesId ||
    binding.trialId !== trial.trialId ||
    binding.sequence !== trial.sequence
  ) fail("SESSION_BINDING_MANIFEST_MISMATCH");
}

function selectTrial(manifest: EvaluationSeriesManifest, sequence: number) {
  if (!Number.isInteger(sequence) || sequence < 1 || sequence > 3) fail("TRIAL_SEQUENCE_INVALID");
  const trial = manifest.trials.find((candidate) => candidate.sequence === sequence);
  if (!trial) fail("TRIAL_SEQUENCE_NOT_FOUND");
  return trial;
}

function validatePrepareInput(input: PrepareSeriesInput) {
  if (!isAbsolute(input.evaluationRoot)) fail("EVALUATION_ROOT_MUST_BE_ABSOLUTE");
  if (!isAbsolute(input.taskPath)) fail("PUBLIC_TASK_PATH_MUST_BE_ABSOLUTE");
  immutableTaskURL(input.taskURL);
  stringMatch(input.releaseTag, releaseTagPattern, "RELEASE_TAG_INVALID");
  immutableReleaseURL(input.releaseURL, input.releaseTag);
  stringMatch(input.releaseSha256, sha256Pattern, "RELEASE_DIGEST_INVALID");
}

function immutableTaskURL(value: string) {
  httpsURL(value, "PUBLIC_TASK_URL_INVALID");
  const url = new URL(value);
  if (
    url.hostname !== "raw.githubusercontent.com" ||
    !/^\/0disoft\/velox\/[0-9a-f]{40}\/evals\/llm-agent\/v1\/task\.md$/.test(url.pathname) ||
    url.search ||
    url.hash
  ) fail("PUBLIC_TASK_URL_NOT_IMMUTABLE");
}

function immutableReleaseURL(value: string, releaseTag: string) {
  httpsURL(value, "RELEASE_URL_INVALID");
  const url = new URL(value);
  if (
    url.hostname !== "github.com" ||
    url.pathname !== `/0disoft/velox/releases/download/${releaseTag}/velox-windows-x64.zip` ||
    url.search ||
    url.hash
  ) fail("RELEASE_URL_TAG_MISMATCH");
}

async function realDirectory(path: string, code: string) {
  if (!isAbsolute(path)) fail(code);
  const supplied = await lstat(resolve(path));
  if (supplied.isSymbolicLink()) fail(code);
  const real = await realpath(path);
  const stat = await lstat(real);
  if (!stat.isDirectory()) fail(code);
  return real;
}

async function realFile(path: string, code: string, maxBytes: number) {
  if (!isAbsolute(path)) fail(code);
  const supplied = await lstat(resolve(path));
  if (supplied.isSymbolicLink()) fail(code);
  const real = await realpath(path);
  const stat = await lstat(real);
  if (!stat.isFile() || stat.size > maxBytes) fail(code);
  return real;
}

async function writeExclusiveJSON(path: string, value: unknown) {
  await writeExclusive(path, `${JSON.stringify(value, null, 2)}\n`);
}

async function writeExclusive(path: string, value: string) {
  await writeFile(path, value, { encoding: "utf8", flag: "wx" });
}

async function requireAbsent(path: string, code: string) {
  const existing = await lstat(path).catch((error: NodeJS.ErrnoException) => {
    if (error.code === "ENOENT") return null;
    throw error;
  });
  if (existing) fail(code);
}

function parseJSON(value: string, code: string): unknown {
  try {
    return JSON.parse(value);
  } catch {
    fail(code);
  }
}

function object(value: unknown, code: string): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) fail(code);
  return value as Record<string, unknown>;
}

function exactKeys(value: Record<string, unknown>, expected: string[], code: string) {
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  if (actual.length !== wanted.length || actual.some((key, index) => key !== wanted[index])) fail(code);
}

function equal(actual: unknown, expected: unknown, code: string) {
  if (actual !== expected) fail(code);
}

function stringMatch(value: unknown, pattern: RegExp, code: string): asserts value is string {
  if (typeof value !== "string" || !pattern.test(value)) fail(code);
}

function dateTime(value: unknown, code: string) {
  if (typeof value !== "string" || !Number.isFinite(Date.parse(value))) fail(code);
}

function httpsURL(value: unknown, code: string): asserts value is string {
  if (typeof value !== "string") fail(code);
  try {
    const parsed = new URL(value);
    if (parsed.protocol !== "https:") fail(code);
  } catch {
    fail(code);
  }
}

function idTimestamp(value: Date) {
  return value.toISOString().replace(/[-:]/g, "").replace(/\.\d{3}Z$/, "Z");
}

function randomSuffix() {
  return randomBytes(4).toString("hex");
}

function sha256(value: string | Uint8Array) {
  return createHash("sha256").update(value).digest("hex");
}

function isContained(root: string, path: string) {
  const normalizedRoot = process.platform === "win32" ? root.toLowerCase() : root;
  const normalizedPath = process.platform === "win32" ? path.toLowerCase() : path;
  return normalizedPath === normalizedRoot || normalizedPath.startsWith(`${normalizedRoot}${sep}`);
}

function requiredEnvironment(environment: NodeJS.ProcessEnv, key: string) {
  const value = environment[key]?.trim();
  if (!value) fail(`ENVIRONMENT_${key}_REQUIRED`);
  return value;
}

function parseFlags(argv: string[]) {
  const flags = new Map<string, string>();
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index];
    const value = argv[index + 1];
    if (!key?.startsWith("--") || !value || flags.has(key)) fail("ORCHESTRATOR_USAGE_INVALID");
    flags.set(key, value);
  }
  return flags;
}

function flag(flags: Map<string, string>, name: string) {
  const value = flags.get(name);
  if (!value) fail("ORCHESTRATOR_USAGE_INVALID");
  return value;
}

function requireFlagCount(flags: Map<string, string>, count: number) {
  if (flags.size !== count) fail("ORCHESTRATOR_USAGE_INVALID");
}

function fail(code: string): never {
  throw new Error(code);
}

async function main(argv: string[]) {
  const [command, ...rest] = argv;
  if (command === "live-diagnose" && rest.length === 0) {
    const localAppData = requiredEnvironment(process.env, "LOCALAPPDATA");
    const sessionId = requiredEnvironment(process.env, "VELOX_HERMES_SESSION_ID");
    console.log(JSON.stringify(inspectHermesSessionCompletion(
      resolve(localAppData, "hermes", "state.db"),
      sessionId,
    )));
    return;
  }
  if (command === "live-smoke" && rest.length === 0) {
    const result = await runLiveHermesSmoke();
    console.log(JSON.stringify({
      ok: true,
      trialId: result.attestation.trialId,
      toolCalls: result.attestation.trajectory.toolCalls,
      retries: result.attestation.trajectory.retries,
      forbiddenActions: result.attestation.trajectory.forbiddenActions,
      sessionCount: result.sessionCount,
      messageCount: result.messageCount,
    }));
    return;
  }

  const flags = parseFlags(rest);
  if (command === "prepare") {
    requireFlagCount(flags, 6);
    const result = await prepareEvaluationSeries({
      evaluationRoot: flag(flags, "--evaluation-root"),
      taskPath: flag(flags, "--task-path"),
      taskURL: flag(flags, "--task-url"),
      releaseTag: flag(flags, "--release-tag"),
      releaseURL: flag(flags, "--release-url"),
      releaseSha256: flag(flags, "--release-sha256"),
    });
    console.log(JSON.stringify({ ok: true, seriesId: result.manifest.seriesId, seriesRoot: result.seriesRoot }));
    return;
  }
  if (command === "bind") {
    requireFlagCount(flags, 3);
    const result = await bindEvaluationSession({
      seriesRoot: flag(flags, "--series-root"),
      sequence: Number(flag(flags, "--sequence")),
      sessionId: flag(flags, "--session-id"),
    });
    console.log(JSON.stringify({ ok: true, trialId: result.trial.trialId, sequence: result.trial.sequence, promptPath: result.promptPath }));
    return;
  }
  if (command === "attest") {
    if (flags.size !== 4 && flags.size !== 5) fail("ORCHESTRATOR_USAGE_INVALID");
    const result = await attestEvaluationTrial({
      seriesRoot: flag(flags, "--series-root"),
      sequence: Number(flag(flags, "--sequence")),
      sessionId: flag(flags, "--session-id"),
      stateDatabasePath: flag(flags, "--state-db"),
      sandboxReceiptPath: flags.get("--sandbox-receipt"),
    });
    console.log(JSON.stringify({ ok: true, trialId: result.attestation.trialId, forbiddenActions: result.attestation.trajectory.forbiddenActions }));
    return;
  }
  if (command === "verify") {
    requireFlagCount(flags, 2);
    console.log(JSON.stringify(await verifyEvaluationSeries(flag(flags, "--series-root"), flag(flags, "--task-path"))));
    return;
  }
  fail("usage: bun scripts/llm-agent-orchestrator.ts <prepare|bind|attest|verify|live-diagnose|live-smoke> [flags]");
}

if (import.meta.main) await main(process.argv.slice(2));
