import { createHash } from "node:crypto";

const sha256Pattern = /^[0-9a-f]{64}$/;
const commitPattern = /^[0-9a-f]{40}$/;
const seriesIDPattern = /^series-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$/;
const trialIDPattern = /^trial-[0-9]{8}T[0-9]{6}Z-[a-z0-9]{8}$/;
const releaseTagPattern = /^v[0-9]+\.[0-9]+\.[0-9]+-(alpha|beta)\.[1-9][0-9]*$/;
const identifierPattern = /^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/;
const isoUtcPattern = /^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$/;
const absolutePathPattern = /(?:^|[\s"'(])(?:[A-Za-z]:\\|\\\\[^\\\s]+\\|\/(?:home|Users|tmp|var\/tmp)\/)/m;
const credentialPattern = /(?:github_pat_[A-Za-z0-9_]+|gh[pousr]_[A-Za-z0-9]+|Bearer\s+[A-Za-z0-9._~+/=-]+|sk-[A-Za-z0-9_-]{16,})/i;
const maximumPacketBytes = 256 * 1024;

const forbiddenPrivateKeys = new Set([
  "prompt",
  "prompttext",
  "transcript",
  "fulltranscript",
  "sessionid",
  "localsessionid",
  "localpath",
  "workspacepath",
  "trialroot",
  "statedatabase",
  "statedatabasepath",
  "credential",
  "credentials",
  "token",
  "apikey",
  "privatereasoning",
  "chainofthought",
  "projection",
  "environmentvariables",
  "commandline",
]);

export interface PublicEvidencePacket {
  schemaVersion: "velox.llm-agent-public-evidence/v1";
  assetName: string;
  payloadSha256: string;
  payload: {
    publishedAtUtc: string;
    retention: "permanent";
    series: {
      seriesId: string;
      outcome: "passed";
      betaTechnicalGate: true;
      humanAdoptionClaim: false;
    };
    release: {
      repository: "0disoft/velox";
      tag: string;
      url: string;
      sha256: string;
    };
    task: {
      schemaVersion: "velox.llm-agent-task/v1";
      sourceCommit: string;
      url: string;
      sha256: string;
    };
    sandbox: {
      attestationSchemaVersion: "velox.llm-agent-evaluation-attestation/v2";
      receiptSchemaVersion: "velox.eval-sandbox-receipt/v1";
      policySchemaVersion: "velox.eval-sandbox-policy/v1";
      filesystemBoundary: "appcontainer-explicit-acl";
      processBoundary: "job-object-no-breakaway";
      networkCapability: "internet-client";
    };
    trials: Array<{
      trialId: string;
      sequence: number;
      provider: string;
      model: string;
      outcome: "passed";
      resultSha256: string;
      attestationSha256: string;
      sandboxReceiptSha256: string;
    }>;
  };
}

export interface PublicEvidenceIdentity {
  seriesId: string;
  releaseTag: string;
  assetName: string;
  payloadSha256: string;
  models: string[];
}

export function parseAndVerifyPublicEvidence(raw: string): PublicEvidenceIdentity {
  if (Buffer.byteLength(raw, "utf8") > maximumPacketBytes) fail("PUBLIC_EVIDENCE_PACKET_TOO_LARGE");
  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    fail("PUBLIC_EVIDENCE_JSON_INVALID");
  }
  return verifyPublicEvidence(value);
}

export function verifyPublicEvidence(value: unknown): PublicEvidenceIdentity {
  rejectPrivateMaterial(value, "$");

  const packet = object(value, "$packet");
  exactKeys(packet, ["schemaVersion", "assetName", "payloadSha256", "payload"], "$packet");
  literal(packet.schemaVersion, "velox.llm-agent-public-evidence/v1", "PUBLIC_EVIDENCE_SCHEMA_INCOMPATIBLE");
  const assetName = text(packet.assetName, "PUBLIC_EVIDENCE_ASSET_NAME_INVALID");
  const payloadSha256 = match(packet.payloadSha256, sha256Pattern, "PUBLIC_EVIDENCE_PAYLOAD_DIGEST_INVALID");

  const payload = object(packet.payload, "payload");
  exactKeys(payload, ["publishedAtUtc", "retention", "series", "release", "task", "sandbox", "trials"], "payload");
  const publishedAtUtc = match(payload.publishedAtUtc, isoUtcPattern, "PUBLIC_EVIDENCE_PUBLISHED_AT_INVALID");
  if (Number.isNaN(Date.parse(publishedAtUtc))) fail("PUBLIC_EVIDENCE_PUBLISHED_AT_INVALID");
  literal(payload.retention, "permanent", "PUBLIC_EVIDENCE_RETENTION_INCOMPATIBLE");

  const series = object(payload.series, "payload.series");
  exactKeys(series, ["seriesId", "outcome", "betaTechnicalGate", "humanAdoptionClaim"], "payload.series");
  const seriesId = match(series.seriesId, seriesIDPattern, "PUBLIC_EVIDENCE_SERIES_ID_INVALID");
  literal(series.outcome, "passed", "PUBLIC_EVIDENCE_SERIES_OUTCOME_INVALID");
  literal(series.betaTechnicalGate, true, "PUBLIC_EVIDENCE_BETA_GATE_INVALID");
  literal(series.humanAdoptionClaim, false, "PUBLIC_EVIDENCE_HUMAN_CLAIM_INVALID");

  const release = object(payload.release, "payload.release");
  exactKeys(release, ["repository", "tag", "url", "sha256"], "payload.release");
  literal(release.repository, "0disoft/velox", "PUBLIC_EVIDENCE_RELEASE_REPOSITORY_INVALID");
  const releaseTag = match(release.tag, releaseTagPattern, "PUBLIC_EVIDENCE_RELEASE_TAG_INVALID");
  literal(release.url, `https://github.com/0disoft/velox/releases/tag/${releaseTag}`, "PUBLIC_EVIDENCE_RELEASE_URL_INVALID");
  match(release.sha256, sha256Pattern, "PUBLIC_EVIDENCE_RELEASE_DIGEST_INVALID");

  const task = object(payload.task, "payload.task");
  exactKeys(task, ["schemaVersion", "sourceCommit", "url", "sha256"], "payload.task");
  literal(task.schemaVersion, "velox.llm-agent-task/v1", "PUBLIC_EVIDENCE_TASK_SCHEMA_INCOMPATIBLE");
  const sourceCommit = match(task.sourceCommit, commitPattern, "PUBLIC_EVIDENCE_TASK_COMMIT_INVALID");
  literal(
    task.url,
    `https://raw.githubusercontent.com/0disoft/velox/${sourceCommit}/evals/llm-agent/v1/task.md`,
    "PUBLIC_EVIDENCE_TASK_URL_INVALID",
  );
  match(task.sha256, sha256Pattern, "PUBLIC_EVIDENCE_TASK_DIGEST_INVALID");

  const sandbox = object(payload.sandbox, "payload.sandbox");
  exactKeys(
    sandbox,
    [
      "attestationSchemaVersion",
      "receiptSchemaVersion",
      "policySchemaVersion",
      "filesystemBoundary",
      "processBoundary",
      "networkCapability",
    ],
    "payload.sandbox",
  );
  literal(
    sandbox.attestationSchemaVersion,
    "velox.llm-agent-evaluation-attestation/v2",
    "PUBLIC_EVIDENCE_ATTESTATION_SCHEMA_INCOMPATIBLE",
  );
  literal(sandbox.receiptSchemaVersion, "velox.eval-sandbox-receipt/v1", "PUBLIC_EVIDENCE_RECEIPT_SCHEMA_INCOMPATIBLE");
  literal(sandbox.policySchemaVersion, "velox.eval-sandbox-policy/v1", "PUBLIC_EVIDENCE_POLICY_SCHEMA_INCOMPATIBLE");
  literal(sandbox.filesystemBoundary, "appcontainer-explicit-acl", "PUBLIC_EVIDENCE_FILESYSTEM_BOUNDARY_INVALID");
  literal(sandbox.processBoundary, "job-object-no-breakaway", "PUBLIC_EVIDENCE_PROCESS_BOUNDARY_INVALID");
  literal(sandbox.networkCapability, "internet-client", "PUBLIC_EVIDENCE_NETWORK_CAPABILITY_INVALID");

  if (!Array.isArray(payload.trials) || payload.trials.length !== 3) fail("PUBLIC_EVIDENCE_TRIAL_COUNT_INVALID");
  const trialIds = new Set<string>();
  const sequences = new Set<number>();
  const models = new Set<string>();
  for (const [index, rawTrial] of payload.trials.entries()) {
    const path = `payload.trials[${index}]`;
    const trial = object(rawTrial, path);
    exactKeys(
      trial,
      ["trialId", "sequence", "provider", "model", "outcome", "resultSha256", "attestationSha256", "sandboxReceiptSha256"],
      path,
    );
    const trialId = match(trial.trialId, trialIDPattern, "PUBLIC_EVIDENCE_TRIAL_ID_INVALID");
    const sequence = integer(trial.sequence, "PUBLIC_EVIDENCE_SEQUENCE_INVALID");
    if (sequence < 1 || sequence > 3) fail("PUBLIC_EVIDENCE_SEQUENCE_INVALID");
    if (trialIds.has(trialId)) fail("PUBLIC_EVIDENCE_DUPLICATE_TRIAL_ID");
    if (sequences.has(sequence)) fail("PUBLIC_EVIDENCE_DUPLICATE_SEQUENCE");
    trialIds.add(trialId);
    sequences.add(sequence);
    const provider = match(trial.provider, identifierPattern, "PUBLIC_EVIDENCE_PROVIDER_INVALID");
    const model = match(trial.model, identifierPattern, "PUBLIC_EVIDENCE_MODEL_INVALID");
    models.add(`${provider}/${model}`);
    literal(trial.outcome, "passed", "PUBLIC_EVIDENCE_TRIAL_OUTCOME_INVALID");
    match(trial.resultSha256, sha256Pattern, "PUBLIC_EVIDENCE_RESULT_DIGEST_INVALID");
    match(trial.attestationSha256, sha256Pattern, "PUBLIC_EVIDENCE_ATTESTATION_DIGEST_INVALID");
    match(trial.sandboxReceiptSha256, sha256Pattern, "PUBLIC_EVIDENCE_SANDBOX_DIGEST_INVALID");
  }
  if ([...sequences].sort().join(",") !== "1,2,3") fail("PUBLIC_EVIDENCE_SEQUENCE_SET_INVALID");
  if (models.size < 2) fail("PUBLIC_EVIDENCE_MODEL_DIVERSITY_INVALID");

  const computedSha256 = sha256(canonicalJSON(payload));
  if (computedSha256 !== payloadSha256) fail("PUBLIC_EVIDENCE_PAYLOAD_DIGEST_MISMATCH");
  const expectedAssetName = `velox-llm-agent-evidence-${seriesId}-${payloadSha256}.json`;
  if (assetName !== expectedAssetName) fail("PUBLIC_EVIDENCE_ASSET_NAME_MISMATCH");

  return {
    seriesId,
    releaseTag,
    assetName,
    payloadSha256,
    models: [...models].sort(),
  };
}

export function canonicalJSON(value: unknown): string {
  if (value === null || typeof value === "boolean" || typeof value === "string") return JSON.stringify(value);
  if (typeof value === "number") {
    if (!Number.isFinite(value)) fail("PUBLIC_EVIDENCE_NON_JSON_NUMBER");
    return JSON.stringify(value);
  }
  if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
  if (typeof value === "object") {
    const record = value as Record<string, unknown>;
    return `{${Object.keys(record)
      .sort()
      .map((key) => `${JSON.stringify(key)}:${canonicalJSON(record[key])}`)
      .join(",")}}`;
  }
  fail("PUBLIC_EVIDENCE_NON_JSON_VALUE");
}

function rejectPrivateMaterial(value: unknown, path: string): void {
  if (typeof value === "string") {
    if (absolutePathPattern.test(value)) fail(`PUBLIC_EVIDENCE_FORBIDDEN_PRIVATE_VALUE:${path}`);
    if (credentialPattern.test(value)) fail(`PUBLIC_EVIDENCE_FORBIDDEN_PRIVATE_VALUE:${path}`);
    return;
  }
  if (Array.isArray(value)) {
    for (const [index, item] of value.entries()) rejectPrivateMaterial(item, `${path}[${index}]`);
    return;
  }
  if (value && typeof value === "object") {
    for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
      const normalized = key.toLowerCase().replace(/[^a-z0-9]/g, "");
      if (forbiddenPrivateKeys.has(normalized)) fail(`PUBLIC_EVIDENCE_FORBIDDEN_PRIVATE_FIELD:${path}.${key}`);
      rejectPrivateMaterial(item, `${path}.${key}`);
    }
  }
}

function exactKeys(record: Record<string, unknown>, allowed: string[], path: string): void {
  const allowedSet = new Set(allowed);
  for (const key of Object.keys(record)) {
    if (!allowedSet.has(key)) fail(`PUBLIC_EVIDENCE_UNKNOWN_FIELD:${path}.${key}`);
  }
  for (const key of allowed) {
    if (!Object.hasOwn(record, key)) fail(`PUBLIC_EVIDENCE_REQUIRED_FIELD_MISSING:${path}.${key}`);
  }
}

function object(value: unknown, path: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) fail(`PUBLIC_EVIDENCE_OBJECT_REQUIRED:${path}`);
  return value as Record<string, unknown>;
}

function text(value: unknown, code: string): string {
  if (typeof value !== "string" || value.length === 0 || value.length > 255) fail(code);
  return value;
}

function match(value: unknown, pattern: RegExp, code: string): string {
  const result = text(value, code);
  if (!pattern.test(result)) fail(code);
  return result;
}

function integer(value: unknown, code: string): number {
  if (typeof value !== "number" || !Number.isInteger(value)) fail(code);
  return value;
}

function literal<T extends string | boolean>(value: unknown, expected: T, code: string): T {
  if (value !== expected) fail(code);
  return expected;
}

function sha256(value: string): string {
  return createHash("sha256").update(value, "utf8").digest("hex");
}

function fail(code: string): never {
  throw new Error(code);
}
