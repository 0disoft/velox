import { describe, expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { canonicalJSON, parseAndVerifyPublicEvidence } from "./llm-agent-public-evidence.ts";

const fixtureRoot = resolve(import.meta.dir, "../tests/fixtures/llm-agent-public-evidence");

async function fixture(name: string): Promise<string> {
  return readFile(resolve(fixtureRoot, name), "utf8");
}

describe("public clean-room evaluation evidence", () => {
  test("verifies a public-only packet and returns its immutable identity", async () => {
    expect(parseAndVerifyPublicEvidence(await fixture("valid.json"))).toEqual({
      seriesId: "series-20260901T000000Z-a1b2c3d4",
      releaseTag: "v0.5.10-beta.1",
      assetName:
        "velox-llm-agent-evidence-series-20260901T000000Z-a1b2c3d4-90862996910453309710da30fed06080fc806931392cdede1895ee5cfb1ca55e.json",
      payloadSha256: "90862996910453309710da30fed06080fc806931392cdede1895ee5cfb1ca55e",
      models: ["example-provider-a/example-model-a", "example-provider-b/example-model-b"],
    });
  });

  test("canonicalizes object keys while preserving array order", () => {
    expect(canonicalJSON({ z: 1, a: { d: 4, b: 2 }, list: [3, 1] })).toBe(
      '{"a":{"b":2,"d":4},"list":[3,1],"z":1}',
    );
  });

  test("rejects slash-form Windows paths before payload verification", async () => {
    const packet = JSON.parse(await fixture("valid.json"));
    packet.payload.trials[0].provider = "C:/Users/private-agent";
    expect(() => parseAndVerifyPublicEvidence(JSON.stringify(packet))).toThrow(
      "PUBLIC_EVIDENCE_FORBIDDEN_PRIVATE_VALUE:$raw",
    );
  });

  test("rejects private values hidden behind duplicate JSON keys", async () => {
    const raw = (await fixture("valid.json")).replace(
      '"provider": "example-provider-a"',
      '"provider": "C:\\\\Users\\\\private-agent", "provider": "example-provider-a"',
    );
    expect(() => parseAndVerifyPublicEvidence(raw)).toThrow("PUBLIC_EVIDENCE_FORBIDDEN_PRIVATE_VALUE:$raw");
  });

  test("rejects forbidden fields hidden behind duplicate JSON keys", async () => {
    const raw = (await fixture("valid.json")).replace(
      '"provider": "example-provider-a"',
      '"transcript": "private", "transcript": "redacted", "provider": "example-provider-a"',
    );
    expect(() => parseAndVerifyPublicEvidence(raw)).toThrow(
      "PUBLIC_EVIDENCE_FORBIDDEN_PRIVATE_FIELD:$.payload.trials[0].transcript",
    );
  });

  const rejections = [
    ["missing.json", "PUBLIC_EVIDENCE_REQUIRED_FIELD_MISSING:payload.task.sha256"],
    ["altered.json", "PUBLIC_EVIDENCE_PAYLOAD_DIGEST_MISMATCH"],
    ["duplicate.json", "PUBLIC_EVIDENCE_DUPLICATE_TRIAL_ID"],
    ["incompatible.json", "PUBLIC_EVIDENCE_ATTESTATION_SCHEMA_INCOMPATIBLE"],
    ["forbidden-private-field.json", "PUBLIC_EVIDENCE_FORBIDDEN_PRIVATE_FIELD:$.payload.trials[0].transcript"],
  ] as const;

  for (const [name, error] of rejections) {
    test(`rejects ${name}`, async () => {
      const raw = await fixture(name);
      expect(() => parseAndVerifyPublicEvidence(raw)).toThrow(error);
    });
  }
});
