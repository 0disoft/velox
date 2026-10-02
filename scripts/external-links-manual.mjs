import fs from "node:fs";
import path from "node:path";
import { createHash, randomUUID } from "node:crypto";
import { spawn, execFileSync } from "node:child_process";

const mode = process.argv[2];
if (!["allowed", "denied"].includes(mode) || process.argv.length !== 3) {
  throw new Error("Expected exactly allowed or denied mode");
}
const root = process.cwd();
const fixture = path.join(root, "tests/fixtures/external-links");
const host = path.join(root, "dist/velox-host.exe");
const hash = (file) => createHash("sha256").update(fs.readFileSync(file)).digest("hex");
const sourceCommit = execFileSync("git", ["rev-parse", "HEAD"], { encoding: "utf8", windowsHide: true }).trim();
const owned = path.join(root, ".cache/external-links-manual", `${Date.now()}-${randomUUID()}`);
const web = path.join(owned, "web");
fs.mkdirSync(web, { recursive: true });
const files = ["index.html", "style.css", "app.js"];
const fixtureHashes = {};
for (const name of files) {
  const source = path.join(fixture, "web", name);
  fixtureHashes[name] = hash(source);
  fs.copyFileSync(source, path.join(web, name));
}
fs.copyFileSync(path.join(root, "assets/branding/velox.png"), path.join(web, "velox.png"));
const sourceConfig = path.join(fixture, `${mode}.runtime.json`);
const config = path.join(owned, "velox.runtime.json");
fixtureHashes.config = hash(sourceConfig);
fs.copyFileSync(sourceConfig, config);
const receipt = { mode, sourceCommit, hostSHA256: hash(host), fixtureHashes, startedAt: new Date().toISOString(), profile: path.join(owned, "profile") };
fs.writeFileSync(path.join(owned, "receipt.json"), `${JSON.stringify(receipt, null, 2)}\n`);
const child = spawn(host, ["--config", config], {
  cwd: owned,
  env: { ...process.env, VELOX_DATA_DIR: receipt.profile },
  windowsHide: false,
  stdio: ["ignore", "ignore", "pipe"],
});
let timedOut = false;
let diagnostic = "";
child.stderr.on("data", (data) => { diagnostic = (diagnostic + data.toString()).slice(-4096); });
child.on("spawn", () => {
  receipt.hostPID = child.pid;
  console.log(JSON.stringify({ started: true, mode, pid: child.pid, receipt: path.join(owned, "receipt.json") }));
});
const deadline = setTimeout(() => { timedOut = true; child.kill(); }, 8 * 60 * 1000);
try {
  const code = await new Promise((resolve, reject) => {
    child.once("error", reject);
    child.once("close", resolve);
  });
  Object.assign(receipt, { exitedAt: new Date().toISOString(), exitCode: code, timedOut });
  fs.writeFileSync(path.join(owned, "receipt.json"), `${JSON.stringify(receipt, null, 2)}\n`);
  console.log(JSON.stringify({ exited: true, mode, code, timedOut }));
  if (timedOut || code !== 0) {
    process.stderr.write(diagnostic);
    process.exitCode = 1;
  }
} finally {
  clearTimeout(deadline);
  if (child.exitCode === null && child.signalCode === null) child.kill();
}
