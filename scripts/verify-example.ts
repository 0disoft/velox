import { mkdir, rm, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

const [configRelative, appID, example] = process.argv.slice(2);
if (!configRelative || !appID || !example || process.argv.length !== 5) {
  throw new Error("usage: bun scripts/verify-example.ts <config> <app-id> <example>");
}

const root = resolve(import.meta.dir, "..");
const cli = resolve(root, "dist/release/velox-windows-x64/velox.exe");
const config = resolve(root, configRelative);
const work = resolve(root, ".cache", `${example}-smoke`);

await rm(work, { recursive: true, force: true });
await mkdir(work, { recursive: true });

async function execute(command: string, args: string[], extraEnv: Record<string, string>, timeoutMilliseconds: number) {
  const child = Bun.spawn([command, ...args], {
    cwd: root,
    env: { ...process.env, ...extraEnv },
    stdout: "pipe",
    stderr: "pipe",
  });
  const stdout = new Response(child.stdout).text();
  const stderr = new Response(child.stderr).text();
  let timeout: ReturnType<typeof setTimeout> | undefined;
  const outcome = await Promise.race([
    child.exited.then((exitCode) => ({ exitCode, timedOut: false })),
    new Promise<{ exitCode: number; timedOut: boolean }>((resolveTimeout) => {
      timeout = setTimeout(async () => {
        const cleanup = Bun.spawn(["taskkill.exe", "/PID", String(child.pid), "/T", "/F"], {
          stdin: "ignore",
          stdout: "ignore",
          stderr: "ignore",
        });
        await cleanup.exited;
        if (!child.killed) child.kill();
        resolveTimeout({ exitCode: await child.exited, timedOut: true });
      }, timeoutMilliseconds);
    }),
  ]).finally(() => clearTimeout(timeout));
  const [stdoutText, stderrText] = await Promise.all([stdout, stderr]);
  if (outcome.timedOut) {
    throw new Error(`${command} exceeded ${timeoutMilliseconds}ms; stdout=${stdoutText.trim()} stderr=${stderrText.trim()}`);
  }
  if (outcome.exitCode !== 0) {
    throw new Error(`${command} exited ${outcome.exitCode}; stdout=${stdoutText.trim()} stderr=${stderrText.trim()}`);
  }
  return { exitCode: outcome.exitCode, stdoutText };
}

async function invoke(args: string[], extraEnv: Record<string, string> = {}, timeoutMilliseconds = 120_000) {
  const result = await execute(cli, args, extraEnv, timeoutMilliseconds);
  const document = JSON.parse(result.stdoutText);
  if (document.ok !== true) throw new Error(`velox ${args[0]} returned a failed JSON envelope`);
  return document;
}

await invoke(["validate", "--config", config, "--out", resolve(work, "validate"), "--json"]);
const doctor = await invoke(["doctor", "--config", config, "--out", resolve(work, "doctor"), "--json"]);
if (doctor.result.ready !== true) throw new Error(`${example} doctor result is not ready`);

const first = await invoke(["build", "--config", config, "--out", resolve(work, "first"), "--json"]);
const second = await invoke(["build", "--config", config, "--out", resolve(work, "second"), "--json"]);
if (first.result.archiveSha256 !== second.result.archiveSha256) throw new Error(`${example} builds are not deterministic`);

const archive = resolve(work, "first", `${appID}.zip`);
const inspection = await invoke(["inspect", archive, "--json"]);
if (inspection.result.app.id !== appID) throw new Error(`${example} inspection identity is invalid`);

const benchmarkEnvironment = {
  VELOX_BENCH_MODE: "1",
  VELOX_BENCH_EXIT_AFTER_READY: "1",
};
const packagedExecutable = resolve(work, "first", appID, `${appID}.exe`);
const directLaunch = await execute(packagedExecutable, [], {
  ...benchmarkEnvironment,
  VELOX_DATA_DIR: resolve(work, "packaged-profile"),
}, 30_000);

const run = await invoke([
  "run",
  "--config", config,
  "--out", resolve(work, "run"),
  "--json",
], {
  ...benchmarkEnvironment,
  VELOX_DATA_DIR: resolve(work, "source-profile"),
}, 30_000);
if (run.result.exitCode !== 0) throw new Error(`${example} source startup did not exit cleanly`);

const result = {
  schemaVersion: "velox.example-smoke/v1",
  example,
  appID,
  releaseVersion: first.result.releaseVersion,
  archiveSha256: first.result.archiveSha256,
  archiveBytes: first.result.archiveBytes,
  deterministic: true,
  doctorReady: true,
  packagedStartupExitCode: directLaunch.exitCode,
  sourceStartupExitCode: run.result.exitCode,
};
await writeFile(resolve(work, "result.json"), `${JSON.stringify(result, null, 2)}\n`, "utf8");
console.log(JSON.stringify(result));
