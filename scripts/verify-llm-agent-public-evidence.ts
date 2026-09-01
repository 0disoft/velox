import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { parseAndVerifyPublicEvidence } from "./llm-agent-public-evidence.ts";

const [target] = process.argv.slice(2);
if (!target || process.argv.length !== 3) {
  throw new Error("usage: bun scripts/verify-llm-agent-public-evidence.ts <public-packet.json>");
}

const identity = parseAndVerifyPublicEvidence(await readFile(resolve(target), "utf8"));
console.log(JSON.stringify({ ok: true, ...identity }));
