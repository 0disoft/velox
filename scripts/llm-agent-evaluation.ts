import { createHash } from "node:crypto";
import { lstat, readFile, realpath } from "node:fs/promises";
import { isAbsolute, resolve, sep } from "node:path";

const sha256Pattern = /^[0-9a-f]{64}$/;
const trialIDPattern = /^trial-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$/;
const seriesIDPattern = /^series-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$/;
const releaseTagPattern = /^v[0-9]+\.[0-9]+\.[0-9]+-(alpha|beta)\.[1-9][0-9]*$/;
const diagnosticPattern = /^[A-Z][A-Z0-9_]{2,63}$/;
const absoluteWindowsPathPattern = /(?:^|[\s"'(])(?:[A-Za-z]:\\|\\\\[^\\\s]+\\)/m;
const credentialPattern = /(?:github_pat_[A-Za-z0-9_]+|gh[pousr]_[A-Za-z0-9]+|Bearer\s+[A-Za-z0-9._~+/=-]+)/i;

const commandClasses = new Set([
  "release-download",
  "checksum-verification",
  "version",
  "init",
  "validate",
  "doctor",
  "asset-edit",
  "build",
  "archive-hash",
  "inspect",
  "run",
  "behavior-check",
]);

const gateNames = [
  "releaseChecksumVerified",
  "publicDocsOnly",
  "noSourceCheckout",
  "noConsumerCompiler",
  "noNodeRuntime",
  "noPackageManager",
  "projectInitialized",
  "doctorReady",
  "deterministicBuild",
  "inspectionPassed",
  "startupReady",
  "appBehaviorVerified",
  "noForbiddenNativeCapability",
] as const;

const artifactPairs = [
  ["firstBuildArchive", "firstBuildSha256", 128 * 1024 * 1024],
  ["secondBuildArchive", "secondBuildSha256", 128 * 1024 * 1024],
  ["buildResult", "buildResultSha256", 1024 * 1024],
  ["safeReport", "safeReportSha256", 64 * 1024],
] as const;

export type TrialOutcome = "passed" | "failed" | "held";

export interface TrialRecord {
  schemaVersion: "velox.llm-agent-evaluation/v1";
  trialId: string;
  seriesId: string;
  sequence: number;
  promptVersion: "velox.llm-agent-task/v1";
  promptSha256: string;
  evaluator: {
    provider: string;
    model: string;
    sessionIdSha256: string;
    freshSession: true;
    memoryCarryover: false;
  };
  control: {
    maintainerOrchestrated: true;
    externalHuman: false;
    veloxSourceCheckout: false;
    unpublishedContext: false;
    interactiveMaintainerHints: 0;
  };
  release: {
    repository: "0disoft/velox";
    tag: string;
    url: string;
    expectedSha256: string;
    observedSha256: string;
  };
  application: {
    id: "dev.velox.agent.focusledger";
    name: "Focus Ledger";
    version: "0.1.0";
  };
  environment: {
    windowsVersion: string;
    webView2Version: string;
    architecture: "AMD64";
    workspaceIsolation: "fresh-local-directory" | "fresh-vm" | "fresh-hosted-runner";
  };
  startedAtUtc: string;
  finishedAtUtc: string;
  outcome: TrialOutcome;
  gates: Record<(typeof gateNames)[number], boolean>;
  trajectory: {
    toolCalls: number;
    retries: number;
    commandClasses: string[];
    forbiddenActions: [];
  };
  artifacts: Partial<Record<(typeof artifactPairs)[number][0] | (typeof artifactPairs)[number][1], string>>;
  diagnostics: string[];
  failure: null | { phase: string; code: string };
  evidenceLevel: "maintainer-orchestrated-clean-room-llm-agent";
  humanAdoptionClaim: false;
}

export interface SeriesSummary {
  schemaVersion: "velox.llm-agent-evaluation-series/v1";
  seriesId: string;
  releaseTag: string;
  releaseSha256: string;
  promptVersion: "velox.llm-agent-task/v1";
  promptSha256: string;
  trialIds: string[];
  modelIdentifiers: string[];
  passedTrials: number;
  failedTrials: number;
  heldTrials: number;
  outcome: TrialOutcome;
  betaTechnicalGate: boolean;
  diagnostics: string[];
  humanAdoptionClaim: false;
}

export async function loadAndVerifyTrial(resultPath: string, trialRoot: string, promptPath: string): Promise<TrialRecord> {
  const root = await realpath(trialRoot);
  const result = await readOwnedFile(root, resultPath, 1024 * 1024);
  const raw = parseJSON(result.bytes, "trial result");
  const trial = validateTrialShape(raw);

  const promptDigest = await digestFile(promptPath, 1024 * 1024);
  if (trial.promptSha256 !== promptDigest) fail("PROMPT_DIGEST_MISMATCH");
  if (trial.release.expectedSha256 !== trial.release.observedSha256) fail("RELEASE_DIGEST_MISMATCH");
  if (Date.parse(trial.startedAtUtc) > Date.parse(trial.finishedAtUtc)) fail("TRIAL_TIME_RANGE_INVALID");

  const artifactRealPaths = new Set<string>();
  for (const [pathName, digestName, limit] of artifactPairs) {
    const relativePath = trial.artifacts[pathName];
    const expectedDigest = trial.artifacts[digestName];
    if (trial.outcome === "passed" && (!relativePath || !expectedDigest)) fail(`ARTIFACT_REQUIRED_${pathName.toUpperCase()}`);
    if (!relativePath && !expectedDigest) continue;
    if (!relativePath || !expectedDigest || !sha256Pattern.test(expectedDigest)) fail("ARTIFACT_CONTRACT_INVALID");
    const artifact = await readOwnedFile(root, relativePath, limit);
    if (artifactRealPaths.has(artifact.path)) fail("ARTIFACT_PATH_DUPLICATE");
    artifactRealPaths.add(artifact.path);
    const observedDigest = digest(artifact.bytes);
    if (observedDigest !== expectedDigest) fail(`ARTIFACT_DIGEST_MISMATCH_${pathName.toUpperCase()}`);
    if (pathName === "buildResult") validateBuildResult(parseJSON(artifact.bytes, "build result"), trial);
    if (pathName === "buildResult" || pathName === "safeReport") rejectSensitiveText(artifact.bytes, pathName);
  }

  if (trial.outcome === "passed") {
    if (!gateNames.every((name) => trial.gates[name] === true)) fail("PASSED_TRIAL_HAS_FAILED_GATE");
    if (trial.failure !== null) fail("PASSED_TRIAL_HAS_FAILURE");
    if (trial.artifacts.firstBuildSha256 !== trial.artifacts.secondBuildSha256) fail("BUILD_DIGESTS_DIFFER");
  } else if (trial.failure === null) {
    fail("NON_PASSING_TRIAL_LACKS_FAILURE");
  }
  return trial;
}

export function summarizeSeries(trials: TrialRecord[]): SeriesSummary {
  if (trials.length !== 3) fail("SERIES_REQUIRES_THREE_TRIALS");
  const ordered = [...trials].sort((left, right) => left.sequence - right.sequence);
  if (ordered.some((trial, index) => trial.sequence !== index + 1)) fail("SERIES_SEQUENCE_INVALID");
  if (new Set(ordered.map((trial) => trial.trialId)).size !== 3) fail("SERIES_TRIAL_ID_DUPLICATE");
  if (new Set(ordered.map((trial) => trial.evaluator.sessionIdSha256)).size !== 3) fail("SERIES_SESSION_DUPLICATE");

  const first = ordered[0];
  for (const trial of ordered.slice(1)) {
    if (trial.seriesId !== first.seriesId) fail("SERIES_ID_MISMATCH");
    if (trial.promptVersion !== first.promptVersion || trial.promptSha256 !== first.promptSha256) fail("SERIES_PROMPT_MISMATCH");
    if (trial.release.tag !== first.release.tag || trial.release.observedSha256 !== first.release.observedSha256) fail("SERIES_RELEASE_MISMATCH");
  }

  const passedTrials = ordered.filter((trial) => trial.outcome === "passed").length;
  const failedTrials = ordered.filter((trial) => trial.outcome === "failed").length;
  const heldTrials = ordered.filter((trial) => trial.outcome === "held").length;
  const modelIdentifiers = [...new Set(ordered.map((trial) => `${trial.evaluator.provider}/${trial.evaluator.model}`))].sort();
  const diagnostics: string[] = [];
  if (modelIdentifiers.length < 2) diagnostics.push("MODEL_DIVERSITY_INSUFFICIENT");
  if (failedTrials > 0) diagnostics.push("TRIAL_FAILURE_PRESENT");
  if (heldTrials > 0) diagnostics.push("TRIAL_HOLD_PRESENT");
  const betaTechnicalGate = passedTrials === 3 && modelIdentifiers.length >= 2;
  const outcome: TrialOutcome = failedTrials > 0 ? "failed" : betaTechnicalGate ? "passed" : "held";

  return {
    schemaVersion: "velox.llm-agent-evaluation-series/v1",
    seriesId: first.seriesId,
    releaseTag: first.release.tag,
    releaseSha256: first.release.observedSha256,
    promptVersion: first.promptVersion,
    promptSha256: first.promptSha256,
    trialIds: ordered.map((trial) => trial.trialId),
    modelIdentifiers,
    passedTrials,
    failedTrials,
    heldTrials,
    outcome,
    betaTechnicalGate,
    diagnostics,
    humanAdoptionClaim: false,
  };
}

function validateTrialShape(raw: unknown): TrialRecord {
  const record = object(raw, "trial");
  exactKeys(record, [
    "schemaVersion", "trialId", "seriesId", "sequence", "promptVersion", "promptSha256", "evaluator", "control",
    "release", "application", "environment", "startedAtUtc", "finishedAtUtc", "outcome", "gates", "trajectory", "artifacts",
    "diagnostics", "failure", "evidenceLevel", "humanAdoptionClaim",
  ], "trial");
  equal(record.schemaVersion, "velox.llm-agent-evaluation/v1", "SCHEMA_VERSION_INVALID");
  stringMatch(record.trialId, trialIDPattern, "TRIAL_ID_INVALID");
  stringMatch(record.seriesId, seriesIDPattern, "SERIES_ID_INVALID");
  integerRange(record.sequence, 1, 3, "TRIAL_SEQUENCE_INVALID");
  equal(record.promptVersion, "velox.llm-agent-task/v1", "PROMPT_VERSION_INVALID");
  stringMatch(record.promptSha256, sha256Pattern, "PROMPT_DIGEST_INVALID");

  validateEvaluator(object(record.evaluator, "evaluator"));
  validateControl(object(record.control, "control"));
  validateRelease(object(record.release, "release"));
  validateApplication(object(record.application, "application"));
  validateEnvironment(object(record.environment, "environment"));
  dateTime(record.startedAtUtc, "START_TIME_INVALID");
  dateTime(record.finishedAtUtc, "FINISH_TIME_INVALID");
  oneOf(record.outcome, ["passed", "failed", "held"], "OUTCOME_INVALID");
  validateGates(object(record.gates, "gates"));
  validateTrajectory(object(record.trajectory, "trajectory"));
  validateArtifacts(object(record.artifacts, "artifacts"));
  validateDiagnostics(record.diagnostics);
  validateFailure(record.failure);
  equal(record.evidenceLevel, "maintainer-orchestrated-clean-room-llm-agent", "EVIDENCE_LEVEL_INVALID");
  equal(record.humanAdoptionClaim, false, "HUMAN_ADOPTION_CLAIM_FORBIDDEN");
  return record as unknown as TrialRecord;
}

function validateEvaluator(value: Record<string, unknown>) {
  exactKeys(value, ["provider", "model", "sessionIdSha256", "freshSession", "memoryCarryover"], "evaluator");
  boundedString(value.provider, 1, 100, "EVALUATOR_PROVIDER_INVALID");
  boundedString(value.model, 1, 200, "EVALUATOR_MODEL_INVALID");
  stringMatch(value.sessionIdSha256, sha256Pattern, "SESSION_DIGEST_INVALID");
  equal(value.freshSession, true, "FRESH_SESSION_REQUIRED");
  equal(value.memoryCarryover, false, "MEMORY_CARRYOVER_FORBIDDEN");
}

function validateControl(value: Record<string, unknown>) {
  exactKeys(value, ["maintainerOrchestrated", "externalHuman", "veloxSourceCheckout", "unpublishedContext", "interactiveMaintainerHints"], "control");
  equal(value.maintainerOrchestrated, true, "MAINTAINER_ORCHESTRATION_REQUIRED");
  equal(value.externalHuman, false, "EXTERNAL_HUMAN_CLAIM_FORBIDDEN");
  equal(value.veloxSourceCheckout, false, "SOURCE_CHECKOUT_FORBIDDEN");
  equal(value.unpublishedContext, false, "UNPUBLISHED_CONTEXT_FORBIDDEN");
  equal(value.interactiveMaintainerHints, 0, "MAINTAINER_HINT_FORBIDDEN");
}

function validateRelease(value: Record<string, unknown>) {
  exactKeys(value, ["repository", "tag", "url", "expectedSha256", "observedSha256"], "release");
  equal(value.repository, "0disoft/velox", "REPOSITORY_INVALID");
  stringMatch(value.tag, releaseTagPattern, "RELEASE_TAG_INVALID");
  url(value.url, "RELEASE_URL_INVALID");
  stringMatch(value.expectedSha256, sha256Pattern, "EXPECTED_RELEASE_DIGEST_INVALID");
  stringMatch(value.observedSha256, sha256Pattern, "OBSERVED_RELEASE_DIGEST_INVALID");
}

function validateEnvironment(value: Record<string, unknown>) {
  exactKeys(value, ["windowsVersion", "webView2Version", "architecture", "workspaceIsolation"], "environment");
  boundedString(value.windowsVersion, 1, 200, "WINDOWS_VERSION_INVALID");
  boundedString(value.webView2Version, 1, 100, "WEBVIEW2_VERSION_INVALID");
  equal(value.architecture, "AMD64", "ARCHITECTURE_INVALID");
  oneOf(value.workspaceIsolation, ["fresh-local-directory", "fresh-vm", "fresh-hosted-runner"], "WORKSPACE_ISOLATION_INVALID");
}

function validateApplication(value: Record<string, unknown>) {
  exactKeys(value, ["id", "name", "version"], "application");
  equal(value.id, "dev.velox.agent.focusledger", "APPLICATION_ID_INVALID");
  equal(value.name, "Focus Ledger", "APPLICATION_NAME_INVALID");
  equal(value.version, "0.1.0", "APPLICATION_VERSION_INVALID");
}

function validateBuildResult(raw: unknown, trial: TrialRecord) {
  const result = object(raw, "build_result");
  equal(result.schemaVersion, "velox.build-result/v1", "BUILD_RESULT_SCHEMA_INVALID");
  equal(result.releaseVersion, trial.release.tag.slice(1), "BUILD_RESULT_RELEASE_INVALID");
  equal(result.target, "windows-x64", "BUILD_RESULT_TARGET_INVALID");
  const app = object(result.app, "build_result_app");
  equal(app.id, trial.application.id, "BUILD_RESULT_APP_ID_INVALID");
  equal(app.name, trial.application.name, "BUILD_RESULT_APP_NAME_INVALID");
  equal(app.version, trial.application.version, "BUILD_RESULT_APP_VERSION_INVALID");
  const permissions = stringArray(result.permissions, "BUILD_RESULT_PERMISSIONS_INVALID");
  if (permissions.some((permission) => permission !== "app.info" && permission !== "window.basic")) fail("BUILD_RESULT_PERMISSION_FORBIDDEN");
  const assets = object(result.assets, "build_result_assets");
  integerRange(assets.files, 1, Number.MAX_SAFE_INTEGER, "BUILD_RESULT_ASSETS_INVALID");
  const outputs = object(result.outputs, "build_result_outputs");
  integerRange(outputs.portableFiles, 3, Number.MAX_SAFE_INTEGER, "BUILD_RESULT_OUTPUTS_INVALID");
}

function validateGates(value: Record<string, unknown>) {
  exactKeys(value, [...gateNames], "gates");
  for (const name of gateNames) if (typeof value[name] !== "boolean") fail(`GATE_INVALID_${name.toUpperCase()}`);
}

function validateTrajectory(value: Record<string, unknown>) {
  exactKeys(value, ["toolCalls", "retries", "commandClasses", "forbiddenActions"], "trajectory");
  integerRange(value.toolCalls, 0, 500, "TOOL_CALL_COUNT_INVALID");
  integerRange(value.retries, 0, 20, "RETRY_COUNT_INVALID");
  const classes = stringArray(value.commandClasses, "COMMAND_CLASSES_INVALID");
  if (new Set(classes).size !== classes.length || classes.some((name) => !commandClasses.has(name))) fail("COMMAND_CLASSES_INVALID");
  if (!Array.isArray(value.forbiddenActions) || value.forbiddenActions.length !== 0) fail("FORBIDDEN_ACTION_RECORDED");
}

function validateArtifacts(value: Record<string, unknown>) {
  exactKeys(value, artifactPairs.flatMap(([pathName, digestName]) => [pathName, digestName]), "artifacts", true);
  for (const [pathName, digestName] of artifactPairs) {
    if (value[pathName] !== undefined) relativePath(value[pathName], `ARTIFACT_PATH_INVALID_${pathName.toUpperCase()}`);
    if (value[digestName] !== undefined) stringMatch(value[digestName], sha256Pattern, `ARTIFACT_DIGEST_INVALID_${pathName.toUpperCase()}`);
  }
}

function validateDiagnostics(value: unknown) {
  const diagnostics = stringArray(value, "DIAGNOSTICS_INVALID");
  if (new Set(diagnostics).size !== diagnostics.length || diagnostics.some((code) => !diagnosticPattern.test(code))) fail("DIAGNOSTICS_INVALID");
}

function validateFailure(value: unknown) {
  if (value === null) return;
  const failure = object(value, "failure");
  exactKeys(failure, ["phase", "code"], "failure");
  oneOf(failure.phase, ["acquisition", "verification", "discovery", "initialization", "doctor", "authoring", "build", "inspection", "startup", "behavior", "evaluation"], "FAILURE_PHASE_INVALID");
  stringMatch(failure.code, diagnosticPattern, "FAILURE_CODE_INVALID");
}

async function readOwnedFile(root: string, relativeOrAbsolute: string, maximumBytes: number) {
  const candidate = isAbsolute(relativeOrAbsolute) ? resolve(relativeOrAbsolute) : resolve(root, relativeOrAbsolute);
  if (!inside(root, candidate)) fail("PATH_OUTSIDE_TRIAL_ROOT");
  const info = await lstat(candidate).catch(() => fail("ARTIFACT_MISSING"));
  if (info.isSymbolicLink() || !info.isFile()) fail("ARTIFACT_NOT_REGULAR_FILE");
  if (info.size > maximumBytes) fail("ARTIFACT_SIZE_LIMIT_EXCEEDED");
  const canonical = await realpath(candidate);
  if (!inside(root, canonical)) fail("ARTIFACT_REALPATH_OUTSIDE_TRIAL_ROOT");
  return { path: canonical, bytes: await readFile(canonical) };
}

async function digestFile(path: string, maximumBytes: number) {
  const info = await lstat(path).catch(() => fail("PROMPT_MISSING"));
  if (info.isSymbolicLink() || !info.isFile() || info.size > maximumBytes) fail("PROMPT_FILE_INVALID");
  return digest(await readFile(path));
}

function rejectSensitiveText(bytes: Buffer, name: string) {
  const text = bytes.toString("utf8");
  if (absoluteWindowsPathPattern.test(text)) fail(`SENSITIVE_ABSOLUTE_PATH_${name.toUpperCase()}`);
  if (credentialPattern.test(text)) fail(`SENSITIVE_CREDENTIAL_${name.toUpperCase()}`);
}

function inside(root: string, path: string) {
  return path === root || path.startsWith(`${root}${sep}`);
}

function digest(bytes: Uint8Array) {
  return createHash("sha256").update(bytes).digest("hex");
}

function parseJSON(bytes: Uint8Array, name: string) {
  try {
    return JSON.parse(Buffer.from(bytes).toString("utf8"));
  } catch {
    fail(`${name.toUpperCase().replaceAll(" ", "_")}_JSON_INVALID`);
  }
}

function object(value: unknown, name: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) fail(`${name.toUpperCase()}_OBJECT_INVALID`);
  return value as Record<string, unknown>;
}

function exactKeys(value: Record<string, unknown>, expected: readonly string[], name: string, allowMissing = false) {
  const expectedSet = new Set(expected);
  if (Object.keys(value).some((key) => !expectedSet.has(key))) fail(`${name.toUpperCase()}_UNKNOWN_FIELD`);
  if (!allowMissing && expected.some((key) => !(key in value))) fail(`${name.toUpperCase()}_MISSING_FIELD`);
}

function equal(actual: unknown, expected: unknown, code: string) {
  if (actual !== expected) fail(code);
}

function boundedString(value: unknown, minimum: number, maximum: number, code: string) {
  if (typeof value !== "string" || value.length < minimum || value.length > maximum) fail(code);
}

function stringMatch(value: unknown, pattern: RegExp, code: string) {
  if (typeof value !== "string" || !pattern.test(value)) fail(code);
}

function relativePath(value: unknown, code: string) {
  if (typeof value !== "string" || !value || value.includes("\\") || value.includes(":") || isAbsolute(value)) fail(code);
  if (value.split("/").some((segment) => segment === "" || segment === "." || segment === "..")) fail(code);
}

function integerRange(value: unknown, minimum: number, maximum: number, code: string) {
  if (!Number.isInteger(value) || (value as number) < minimum || (value as number) > maximum) fail(code);
}

function oneOf(value: unknown, allowed: readonly unknown[], code: string) {
  if (!allowed.includes(value)) fail(code);
}

function stringArray(value: unknown, code: string): string[] {
  if (!Array.isArray(value) || value.some((item) => typeof item !== "string")) fail(code);
  return value as string[];
}

function dateTime(value: unknown, code: string) {
  if (typeof value !== "string" || Number.isNaN(Date.parse(value))) fail(code);
}

function url(value: unknown, code: string) {
  if (typeof value !== "string") fail(code);
  try {
    const parsed = new URL(value);
    if (parsed.protocol !== "https:") fail(code);
  } catch {
    fail(code);
  }
}

function fail(code: string): never {
  throw new Error(code);
}
