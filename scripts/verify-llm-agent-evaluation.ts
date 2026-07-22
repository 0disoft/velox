import { lstat, mkdir, readdir, rename, rm, writeFile } from "node:fs/promises";
import { basename, dirname, resolve } from "node:path";
import { loadAndVerifyTrial, summarizeSeries } from "./llm-agent-evaluation.ts";

const [command, target, prompt, output] = process.argv.slice(2);

if (command === "trial" && target && prompt && !output && process.argv.length === 5) {
  const root = resolve(target);
  const result = await loadAndVerifyTrial(resolve(root, "result.json"), root, resolve(prompt));
  console.log(JSON.stringify({ ok: true, trialId: result.trialId, outcome: result.outcome }));
} else if (command === "series" && target && prompt && output && process.argv.length === 6) {
  const seriesRoot = resolve(target);
  const entries = await readdir(seriesRoot, { withFileTypes: true });
  const trialDirectories = entries.filter((entry) => entry.isDirectory() && entry.name.startsWith("trial-")).map((entry) => resolve(seriesRoot, entry.name)).sort();
  const trials = [];
  for (const trialRoot of trialDirectories) {
    trials.push(await loadAndVerifyTrial(resolve(trialRoot, "result.json"), trialRoot, resolve(prompt)));
  }
  const summary = summarizeSeries(trials);
  const destination = resolve(output);
  if (dirname(destination) !== seriesRoot) throw new Error("SUMMARY_OUTPUT_OUTSIDE_SERIES_ROOT");
  if (basename(destination) !== "summary.json") throw new Error("SUMMARY_OUTPUT_NAME_INVALID");
  const existing = await lstat(destination).catch((error: NodeJS.ErrnoException) => {
    if (error.code === "ENOENT") return null;
    throw error;
  });
  if (existing) throw new Error("SUMMARY_OUTPUT_ALREADY_EXISTS");
  await mkdir(seriesRoot, { recursive: true });
  const temporary = resolve(seriesRoot, `.summary-${process.pid}.tmp`);
  await writeFile(temporary, `${JSON.stringify(summary, null, 2)}\n`, { encoding: "utf8", flag: "wx" });
  try {
    await rename(temporary, destination);
  } catch (error) {
    await rm(temporary, { force: true });
    throw error;
  }
  console.log(JSON.stringify(summary));
} else {
  throw new Error("usage: bun scripts/verify-llm-agent-evaluation.ts trial <trial-dir> <prompt> | series <series-dir> <prompt> <series-dir/summary.json>");
}
