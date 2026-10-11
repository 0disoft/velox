import { spawn } from "node:child_process";
import { createHash, randomUUID } from "node:crypto";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import { resolve, join, basename } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";

const mode = process.argv[3] ?? "short";
if (process.platform !== "win32" || process.argv.length < 3 || process.argv.length > 4 ||
    !["short", "--long-profile", "--extended-profile"].includes(mode)) {
  throw Error("Usage on Windows: node scripts/packaged-app-session.mjs PORTABLE_DIRECTORY [--long-profile|--extended-profile]");
}
const directory = resolve(process.argv[2]);
const report = JSON.parse(await readFile(join(directory, "build-result.json"), "utf8"));
if (typeof report.host?.file !== "string" || basename(report.host.file) !== report.host.file) {
  throw Error("Packaged host path is invalid.");
}
const executable = join(directory, report.host.file);
const bytes = await readFile(executable);
const digest = createHash("sha256").update(bytes).digest("hex");
if (digest !== report.host.sha256 || bytes.length !== report.host.bytes) {
  throw Error("Packaged executable does not match its build report.");
}
const root = fileURLToPath(new URL("..", import.meta.url));
const work = join(root, ".cache", "packaged-session-" + randomUUID());
await mkdir(work, { recursive: true });
const profile = mode === "short" ? await mkdtemp(join(tmpdir(), "vx-")) : join(work, "profile");
await mkdir(profile, { recursive: true });
const dataPath = mode === "--extended-profile" ? "\\\\?\\" + profile : profile;
const env = { ...process.env, VELOX_DATA_DIR: dataPath };
for (const key of Object.keys(env)) {
  if (key.startsWith("VELOX_BENCH_") || key === "WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS") delete env[key];
}
const child = spawn(executable, ["--config", join(directory, "velox.runtime.json")], {
  cwd: directory, env, windowsHide: false, stdio: ["ignore", "pipe", "pipe"],
});
let outputBytes = 0;
let forcedStopRequested = false;
let spawnFailure = null;
const stop = () => { forcedStopRequested = true; child.kill(); };
const deadline = setTimeout(stop, 8 * 60 * 1000);
process.once("SIGINT", stop);
process.once("SIGTERM", stop);
for (const stream of [child.stdout, child.stderr]) stream.on("data", bytes => { outputBytes += bytes.length; });
child.once("error", error => { spawnFailure = error.code ?? "SPAWN_FAILED"; });
const startedAtUtc = new Date().toISOString();
console.log(JSON.stringify({ state: "started", pid: child.pid, executable, work, profile, dataPath, mode, hostSha256: digest }));
const exit = await new Promise(done => child.once("close", (code, signal) => done({ code, signal })));
clearTimeout(deadline);
process.removeListener("SIGINT", stop);
process.removeListener("SIGTERM", stop);
const receipt = { startedAtUtc, finishedAtUtc: new Date().toISOString(), pid: child.pid,
  executable, profile, dataPath, mode, hostSha256: digest, exit, spawnFailure, forcedStopRequested,
  outputBytes, profilePreserved: true };
await writeFile(join(work, "session.json"), JSON.stringify(receipt, null, 2) + "\n");
console.log(JSON.stringify({ state: "finished", receipt: join(work, "session.json"), ...receipt }));
if (spawnFailure || exit.code !== 0 || forcedStopRequested) process.exitCode = 1;
