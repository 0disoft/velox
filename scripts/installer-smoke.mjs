import { execFile, execFileSync } from "node:child_process";
import { cpSync, existsSync, mkdirSync, readFileSync, readdirSync, renameSync, rmSync, writeFileSync } from "node:fs";
import { createHash, randomUUID } from "node:crypto";
import { join, resolve, dirname, basename } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
if (process.platform !== "win32") throw Error("Installer smoke requires Windows");
const options = {};
for (let index = 2; index < process.argv.length; index += 2) {
  const key = process.argv[index];
  if (!["--cli", "--example"].includes(key) || !process.argv[index + 1] || options[key]) {
    throw Error("Usage: installer-smoke.mjs [--cli PATH] [--example PATH]");
  }
  options[key] = resolve(process.argv[index + 1]);
}
const work = join(root, ".cache", `installer-smoke-${randomUUID()}`);
const cli = options["--cli"] ?? join(root, "dist/release/velox-windows-x64/velox.exe");
const execute = (command, args, env = {}, windowsHide = true) => execFileSync(command, args, {
  cwd: root, env: { ...process.env, ...env }, encoding: "utf8", windowsHide,
  stdio: ["ignore", "pipe", "pipe"], timeout: 30_000,
}).trim();
const invoke = args => {
  const result = JSON.parse(execute(cli, [...args, "--json"]));
  if (!result.ok) throw Error(`CLI failed: ${JSON.stringify(result)}`);
  return result.result;
};
const snapshot = directory => {
  const files = {};
  const visit = (path, prefix = "") => {
    for (const entry of readdirSync(path, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
      const relative = prefix + entry.name;
      if (entry.isDirectory()) visit(join(path, entry.name), relative + "/");
      else if (entry.isFile()) files[relative] = createHash("sha256").update(readFileSync(join(path, entry.name))).digest("hex");
      else throw Error(`Unexpected profile entry: ${relative}`);
    }
  };
  visit(directory);
  return files;
};
const startShortcut = async (shortcut, installedExe) => {
  // Shell launching a .lnk need not inherit benchmark variables. Close the real
  // native window instead of treating a missing benchmark callback as a hang.
  const script = `
$ErrorActionPreference='Stop'
Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public class VeloxSmokeWindow { [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr window, uint message, IntPtr wParam, IntPtr lParam); }'
$p=Start-Process -FilePath $env:VELOX_SMOKE_SHORTCUT -PassThru
Write-Output $p.Id
if ($p.Path -ne $env:VELOX_SMOKE_EXE) { throw 'Shortcut launched an unexpected executable' }
$deadline=[DateTime]::UtcNow.AddSeconds(15)
do { $p.Refresh(); if($p.HasExited){throw 'App exited before a native window appeared'}; if($p.MainWindowHandle -ne 0){break}; Start-Sleep -Milliseconds 100 } while([DateTime]::UtcNow -lt $deadline)
if($p.MainWindowHandle -eq 0){throw 'Native window did not appear'}
$caption=$p.MainWindowTitle
Start-Sleep -Seconds 3
$browsers=@(Get-CimInstance Win32_Process -Filter "Name='msedgewebview2.exe'" | Where-Object { $_.ParentProcessId -eq $p.Id })
if($browsers.Count -eq 0){throw 'App did not start WebView2'}
if(-not [VeloxSmokeWindow]::PostMessage($p.MainWindowHandle,0x10,[IntPtr]::Zero,[IntPtr]::Zero)){throw 'Close message failed'}
if(-not $p.WaitForExit(10000)){throw 'Native close timed out'}
if($p.ExitCode -ne 0){throw 'Host exited with an error'}
foreach($browser in $browsers){$b=Get-Process -Id $browser.ProcessId -ErrorAction SilentlyContinue; if($b -and -not $b.WaitForExit(10000)){throw 'Browser remained running'}}
Write-Output ($caption | ConvertTo-Json -Compress)
`;
  let hostPID;
  let processOutput = "";
  try {
    const stdout = await new Promise((resolve, reject) => {
      const env = { ...process.env, VELOX_SMOKE_SHORTCUT: shortcut, VELOX_SMOKE_EXE: installedExe };
      for (const key of Object.keys(env)) if (key.startsWith("VELOX_BENCH_") || key === "VELOX_DATA_DIR") delete env[key];
      const child = execFile("powershell.exe", ["-NoProfile", "-NonInteractive", "-Command", script], {
        cwd: root, windowsHide: true, timeout: 45_000, encoding: "utf8", env,
      }, (error, stdout, stderr) => error ? reject(Error(`Shortcut launch failed: ${error.message}\n${stdout}\n${stderr}`)) : resolve(stdout));
      child.stdout.on("data", chunk => {
        processOutput += chunk;
        const match = processOutput.match(/^([1-9][0-9]*)\r?\n/);
        if (match) hostPID = Number(match[1]);
      });
    });
    const lines = stdout.trim().split(/\r?\n/);
    return { shortcutStarted: true, nativeWindowTitle: JSON.parse(lines.at(-1)), hostExitCode: 0, browserExited: true, domReadyVerified: false };
  } catch (error) {
    if (hostPID) {
      try { execute("taskkill.exe", ["/PID", String(hostPID), "/T", "/F"]); } catch {}
    }
    throw error;
  }
};

mkdirSync(work, { recursive: true });
const project = join(work, `installer-smoke-${basename(work).slice(-12)}`);
const initialized = invoke(["init", project]);
let exampleAssets;
if (options["--example"]) {
  const example = options["--example"];
  const manifest = JSON.parse(readFileSync(join(example, "velox.json"), "utf8"));
  const assets = resolve(example, manifest.assets.root);
  if (dirname(assets) !== example || basename(assets) !== "web") throw Error("Example must use its own web directory");
  exampleAssets = snapshot(assets);
  cpSync(assets, join(project, "web"), { recursive: true });
  if (JSON.stringify(exampleAssets) !== JSON.stringify(snapshot(join(project, "web")))) throw Error("Example asset copy mismatch");
  manifest.app.id = initialized.appId;
  manifest.app.name += " - Installer Smoke";
  writeFileSync(join(project, "velox.json"), JSON.stringify(manifest, null, 2) + "\n");
} else {
  writeFileSync(join(project, "web/app.js"), 'requestAnimationFrame(() => requestAnimationFrame(() => window.__veloxReady("dom-2raf")));\n');
}
const first = invoke(["build", "--config", join(project, "velox.json"), "--out", join(work, "first"), "--installer"]);
const second = invoke(["build", "--config", join(project, "velox.json"), "--out", join(work, "second"), "--installer"]);
if (!first.installer || first.installer.sha256 !== second.installer?.sha256) throw Error("Setup determinism failed");
const setup = join(work, "first", first.installer.file);
const id = initialized.appId;
const installed = join(process.env.LOCALAPPDATA, "Programs/Velox", id);
const shortcut = join(process.env.APPDATA, "Microsoft/Windows/Start Menu/Programs/Velox", `${id}.lnk`);
const registryPath = `HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\Velox.${id}`;
const profile = join(process.env.LOCALAPPDATA, "Velox/profiles", id);
if (existsSync(installed) || existsSync(shortcut) || existsSync(profile)) throw Error("Test identity already exists; preserved without changes");
let registered = false;
try { execute("reg.exe", ["query", registryPath]); registered = true; } catch (error) {
  if (error.status !== 1) throw error;
}
if (registered) throw Error("Test registry identity already exists; preserved without changes");
mkdirSync(profile, { recursive: true });
writeFileSync(join(profile, "draft-preservation-marker.json"), JSON.stringify({ text: "Disposable recovery preservation marker" }) + "\n", { flag: "wx" });
const document = join(work, "keep.md");
writeFileSync(document, "# Keep this document\n", { flag: "wx" });
let helperDirectory;
try {
  execute(setup, ["--silent"]);
  const installedExe = join(installed, "app", `${id}.exe`);
  if (!existsSync(installedExe)) throw Error("Installed executable missing");
  if (!existsSync(shortcut)) throw Error("Start Menu shortcut missing");
  invoke(["inspect", join(installed, "app")]);
  const registry = execute("reg.exe", ["query", registryPath]);
  if (!registry.includes("DisplayName") || !registry.includes("UninstallString")) throw Error("Missing uninstall registration");
  const startup = await startShortcut(shortcut, installedExe);
  const beforeRemoval = snapshot(profile);
  helperDirectory = execute(join(installed, "uninstall.exe"), ["--uninstall", id, "--silent"]);
  const deadline = Date.now() + 15_000;
  while (existsSync(installed) && Date.now() < deadline) await new Promise(resolve => setTimeout(resolve, 150));
  if (existsSync(installed)) throw Error("Uninstall did not complete");
  if (existsSync(shortcut)) throw Error("Start Menu shortcut remains");
  let registrationRemains = false;
  try { execute("reg.exe", ["query", registryPath]); registrationRemains = true; } catch (error) {
    if (error.status !== 1) throw error;
  }
  if (registrationRemains) throw Error("Uninstall registry entry remains");
  if (readFileSync(document, "utf8") !== "# Keep this document\n") throw Error("User document changed");
  const afterRemoval = snapshot(profile);
  if (JSON.stringify(beforeRemoval) !== JSON.stringify(afterRemoval)) throw Error("Recovery profile changed during removal");
  if (exampleAssets && JSON.stringify(exampleAssets) !== JSON.stringify(snapshot(join(options["--example"], "web")))) throw Error("Source example changed during smoke");
  const evidence = {
    appId: id, installerSha256: first.installer.sha256, deterministic: true,
    cli, example: options["--example"] ?? null, releaseVersion: first.releaseVersion,
    installedStartup: startup, uninstallComplete: true, documentsPreserved: true,
    profileFilesPreserved: Object.keys(afterRemoval).length, profileSnapshot: afterRemoval,
    sourceExampleUnchanged: Boolean(exampleAssets), manualUIVerified: false, actualDraftRestoreVerified: false,
    indexedDBFilesPreserved: Object.keys(afterRemoval).filter(path => path.includes("/IndexedDB/")).length,
  };
  writeFileSync(join(work, "result.json"), JSON.stringify(evidence, null, 2) + "\n");
  const { profileSnapshot, ...summary } = evidence;
  console.log(JSON.stringify({ ...summary, receipt: join(work, "result.json") }));
} catch (error) {
  writeFileSync(join(work, "failure.json"), JSON.stringify({ message: error.message, appId: id, cli }, null, 2) + "\n");
  throw error;
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
  if (!existsSync(installed) && existsSync(profile)) {
    const profileRoot = resolve(process.env.LOCALAPPDATA, "Velox/profiles");
    if (dirname(resolve(profile)).toLowerCase() !== profileRoot.toLowerCase() || basename(profile) !== id) throw Error("Unsafe test profile cleanup path");
    try { renameSync(profile, join(work, "preserved-profile")); }
    catch (error) { console.error(`Test profile preserved at ${profile}: ${error.message}`); }
  }
}
