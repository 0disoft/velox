import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join, resolve, dirname, basename } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const work = join(root, ".cache", `installer-smoke-${process.pid}`);
const cli = join(root, "dist/release/velox-windows-x64/velox.exe");
const execute = (command, args, env = {}, windowsHide = true) => execFileSync(command, args, {
  cwd: root, env: { ...process.env, ...env }, encoding: "utf8", windowsHide,
  stdio: ["ignore", "pipe", "pipe"], timeout: 30_000,
}).trim();
const invoke = args => {
  const result = JSON.parse(execute(cli, [...args, "--json"]));
  if (!result.ok) throw Error(`CLI failed: ${JSON.stringify(result)}`);
  return result.result;
};

mkdirSync(work, { recursive: true });
const project = join(work, `installer-smoke-${process.pid}`);
const initialized = invoke(["init", project]);
writeFileSync(join(project, "web/app.js"), 'requestAnimationFrame(() => requestAnimationFrame(() => window.__veloxReady("dom-2raf")));\n');
const first = invoke(["build", "--config", join(project, "velox.json"), "--out", join(work, "first"), "--installer"]);
const second = invoke(["build", "--config", join(project, "velox.json"), "--out", join(work, "second"), "--installer"]);
if (!first.installer || first.installer.sha256 !== second.installer?.sha256) throw Error("Setup determinism failed");
const setup = join(work, "first", first.installer.file);
const id = initialized.appId;
const installed = join(process.env.LOCALAPPDATA, "Programs/Velox", id);
const shortcut = join(process.env.APPDATA, "Microsoft/Windows/Start Menu/Programs/Velox", `${id}.lnk`);
const document = join(work, "keep.md");
writeFileSync(document, "# Keep this document\n", { flag: "wx" });
let helperDirectory;
try {
  execute(setup, ["--silent"]);
  const installedExe = join(installed, "app", `${id}.exe`);
  if (!existsSync(installedExe)) throw Error("Installed executable missing");
  if (!existsSync(shortcut)) throw Error("Start Menu shortcut missing");
  invoke(["inspect", join(installed, "app")]);
  const registryPath = `HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\Velox.${id}`;
  const registry = execute("reg.exe", ["query", registryPath]);
  if (!registry.includes("DisplayName") || !registry.includes("UninstallString")) throw Error("Missing uninstall registration");
  execute(installedExe, [], {
    VELOX_BENCH_MODE: "1", VELOX_BENCH_EXIT_AFTER_READY: "1", VELOX_DATA_DIR: join(work, "profile"),
  }, false);
  helperDirectory = execute(join(installed, "uninstall.exe"), ["--uninstall", id, "--silent"]);
  const deadline = Date.now() + 15_000;
  while (existsSync(installed) && Date.now() < deadline) await new Promise(resolve => setTimeout(resolve, 150));
  if (existsSync(installed)) throw Error("Uninstall did not complete");
  if (existsSync(shortcut)) throw Error("Start Menu shortcut remains");
  let registrationRemains = false;
  try { execute("reg.exe", ["query", registryPath]); registrationRemains = true; } catch {}
  if (registrationRemains) throw Error("Uninstall registry entry remains");
  if (readFileSync(document, "utf8") !== "# Keep this document\n") throw Error("User document changed");
  const evidence = {
    appId: id, installerSha256: first.installer.sha256, deterministic: true,
    installedStartup: true, uninstallComplete: true, documentsPreserved: true,
  };
  writeFileSync(join(work, "result.json"), JSON.stringify(evidence, null, 2) + "\n");
  console.log(JSON.stringify(evidence));
} finally {
  if (existsSync(join(installed, "uninstall.exe"))) {
    helperDirectory = execute(join(installed, "uninstall.exe"), ["--uninstall", id, "--silent"]);
    const deadline = Date.now() + 15_000;
    while (existsSync(installed) && Date.now() < deadline) await new Promise(resolve => setTimeout(resolve, 150));
    if (existsSync(installed)) throw Error(`Smoke cleanup failed for ${id}; installation was preserved`);
  }
  if (helperDirectory) {
    const path = resolve(helperDirectory);
    if (dirname(path).toLowerCase() !== resolve(tmpdir()).toLowerCase() || !basename(path).startsWith("velox-uninstall-")) throw Error("Unsafe helper cleanup path");
    for (let attempt = 0; attempt < 20; attempt++) {
      try { rmSync(path, { recursive: true }); break; } catch (error) {
        if (attempt === 19) throw error;
        await new Promise(resolve => setTimeout(resolve, 100));
      }
    }
  }
}
