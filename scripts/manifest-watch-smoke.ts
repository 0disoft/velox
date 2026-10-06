import { createHash, randomUUID } from "node:crypto";
import { createServer } from "node:net";
import { spawn, spawnSync } from "node:child_process";
import { cp, mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import { resolve, join } from "node:path";

const root = resolve(import.meta.dir, "..");
const bin = resolve(root, process.env.VELOX_MANIFEST_WATCH_BIN_DIR ?? ".cache/manifest-watch-bin");
const json = process.argv.includes("--json");
const hashes: Record<string, string> = {};
const sha256 = (bytes: Uint8Array) => createHash("sha256").update(bytes).digest("hex");
for (const file of ["velox.exe", "velox-host.exe"]) hashes[file] = sha256(await readFile(join(bin, file)));
const metadata = JSON.parse(await readFile(join(bin, "velox-host.json"), "utf8"));
const hostBytes = await readFile(join(bin, "velox-host.exe"));
if (metadata.host.sha256 !== hashes["velox-host.exe"] || metadata.host.bytes !== hostBytes.length) throw Error("Host metadata mismatch");

const work = join(root, ".cache", "manifest-watch-" + Date.now());
const project = join(work, "project");
await mkdir(work, { recursive: true });
await cp(join(root, "examples/file-notes"), project, { recursive: true });
const configPath = join(project, "velox.json");
const config = JSON.parse(await readFile(configPath, "utf8"));
config.app.id = "dev.velox.manifest-watch-smoke";
config.app.name = "Velox Manifest Watch Test";
config.window.rememberState = false;
config.window.tray = false;
await writeFile(configPath, JSON.stringify(config, null, 2));

const server = createServer();
await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
const address = server.address();
if (!address || typeof address === "string") throw Error("No diagnostic port");
const port = address.port;
await new Promise<void>((resolve, reject) => server.close(error => error ? reject(error) : resolve()));

type State = { documentId: string; text: string; saveState: string; origin: string; debugBinding: boolean };
type Message = { id?: number; method?: string; error?: { message: string }; result?: { result?: { value?: unknown }; exceptionDetails?: unknown } };
const result: {
  mode: string; hashes: Record<string, string>; hostVersion: string; startedAtUtc: string;
  before?: State; cycles: { phase: string; observed: State; runtimeConfigSHA256: string }[];
  noticeCount?: number; errorCount?: number; closeExitCode?: number | null; cleanupExitCode?: number | null;
  temporaryConfigRemoved?: boolean; dialogCount?: number; passed: boolean; error?: string; finishedAtUtc?: string;
  envelopeOK?: boolean;
} = { mode: json ? "source-cli-watch-json-with-debug-off" : "source-cli-watch-with-debug-off", hashes, hostVersion: metadata.releaseVersion,
  startedAtUtc: new Date().toISOString(), cycles: [], passed: false };
const child = spawn(join(bin, "velox.exe"), ["run", "--config", configPath, "--watch", ...(json ? ["--json"] : [])], {
  cwd: root, env: { ...process.env, VELOX_DATA_DIR: join(work, "profile"),
    WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-address=127.0.0.1 --remote-debugging-port=" + port },
  stdio: ["ignore", "pipe", "pipe"], windowsHide: true,
});
let stdout = "", stderr = "", spawnError: Error | undefined, ws: WebSocket | undefined, nextID = 0;
let dialogCount = 0;
const pending = new Map<number, (message: Message) => void>();
child.on("error", error => { spawnError = error; });
child.stdout.on("data", bytes => { stdout = (stdout + bytes.toString()).slice(-32768); });
child.stderr.on("data", bytes => { stderr = (stderr + bytes.toString()).slice(-32768); });
const wait = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
const notices = () => stderr.split("\n").filter(line => line.includes("manifest changed; restart required")).length;
const errors = () => stderr.split("\n").filter(line => line.includes("watch: manifest error:")).length;

try {
  let target: { webSocketDebuggerUrl: string } | undefined;
  const deadline = Date.now() + 20000;
  while (!target && Date.now() < deadline) {
    if (spawnError) throw spawnError;
    if (child.exitCode !== null) throw Error("CLI exited before inspection");
    try {
      const pages = await (await fetch("http://127.0.0.1:" + port + "/json/list", { signal: AbortSignal.timeout(500) })).json();
      target = pages.find((page: { type: string; url: string }) => page.type === "page" && new URL(page.url).hostname.endsWith(".app.invalid"));
    } catch {}
    if (!target) await wait(100);
  }
  if (!target) throw Error("No isolated manifest-watch target");
  ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise<void>((resolve, reject) => {
    const timer = setTimeout(() => reject(Error("WebSocket timeout")), 5000);
    ws!.onopen = () => { clearTimeout(timer); resolve(); };
    ws!.onerror = () => { clearTimeout(timer); reject(Error("WebSocket failed")); };
  });
  ws.onmessage = event => {
    const message: Message = JSON.parse(String(event.data));
    if (message.method === "Page.javascriptDialogOpening") dialogCount++;
    if (message.id && pending.has(message.id)) {
      pending.get(message.id)!(message);
      pending.delete(message.id);
    }
  };
  const call = (method: string, params: object = {}) => new Promise<Message>((resolve, reject) => {
    const id = ++nextID;
    const timer = setTimeout(() => { pending.delete(id); reject(Error("CDP timeout: " + method)); }, 5000);
    pending.set(id, message => { clearTimeout(timer); message.error ? reject(Error(message.error.message)) : resolve(message); });
    ws!.send(JSON.stringify({ id, method, params }));
  });
  const state = async () => {
    const response = await call("Runtime.evaluate", { expression: '({documentId:window.__manifestWatchDocument,text:document.querySelector("#editor").value,saveState:document.querySelector("#save-state").textContent,origin:location.origin,debugBinding:typeof window.__veloxDevDiagnostic==="function"})', returnByValue: true });
    const value = response.result?.result?.value as State | undefined;
    if (!value) throw Error("No editor state");
    return value;
  };
  await call("Page.enable");
  const readyDeadline = Date.now() + 8000;
  let ready = false;
  while (!ready && Date.now() < readyDeadline) {
    const response = await call("Runtime.evaluate", { expression: 'document.readyState==="complete"&&document.querySelector("#save-state")?.textContent==="No save target"', returnByValue: true });
    ready = response.result?.result?.value === true;
    if (!ready) await wait(100);
  }
  if (!ready) throw Error("Editor did not finish initialization");
  const marker = randomUUID();
  const setup = await call("Runtime.evaluate", { expression: '(()=>{window.__manifestWatchDocument=' + JSON.stringify(marker) + ';document.querySelector("#editor").focus();})()' });
  if (setup.result?.exceptionDetails) throw Error("Editor setup evaluation failed");
  await call("Input.insertText", { text: "manifest watch unsaved text" });
  await wait(1200);
  result.before = await state();
  if (result.before.documentId !== marker || result.before.text !== "manifest watch unsaved text" || result.before.saveState !== "Unsaved changes" || result.before.debugBinding) throw Error("Dirty editor baseline failed");
  const runtimeFiles = (await readdir(project)).filter(file => file.startsWith(".velox-run-") && file.endsWith(".json"));
  if (runtimeFiles.length !== 1) throw Error("Expected one temporary runtime config");
  const runtimePath = join(project, runtimeFiles[0]);
  const runtimeDigest = sha256(await readFile(runtimePath));

  async function observe(phase: string, expectedNotices: number, expectedErrors: number) {
    const deadline = Date.now() + 8000;
    while ((notices() < expectedNotices || errors() < expectedErrors) && Date.now() < deadline) {
      if (child.exitCode !== null) throw Error("CLI exited during " + phase);
      await wait(100);
    }
    await wait(1500);
    const observed = await state();
    const digest = sha256(await readFile(runtimePath));
    if (notices() !== expectedNotices || errors() !== expectedErrors) throw Error("Missing or repeated diagnostic: " + phase);
    if (JSON.stringify(observed) !== JSON.stringify(result.before) || digest !== runtimeDigest || dialogCount !== 0 || child.exitCode !== null) throw Error("Manifest edit changed running app: " + phase);
    result.cycles.push({ phase, observed, runtimeConfigSHA256: digest });
  }

  config.app.name = "Changed Name Requires Restart";
  config.window.width = 900;
  config.security.permissions = [];
  await writeFile(configPath, JSON.stringify(config, null, 2));
  await observe("valid-settings-edit", 1, 0);
  await writeFile(configPath, "{");
  await observe("invalid-json", 1, 1);
  config.app.name = "Corrected Name Requires Restart";
  await writeFile(configPath, JSON.stringify(config, null, 2));
  await observe("corrected-settings", 2, 1);
  result.noticeCount = notices();
  result.errorCount = errors();
  result.dialogCount = dialogCount;

  // Clear only the isolated fixture before normal close, avoiding a consent prompt.
  await call("Runtime.evaluate", { expression: '(()=>{const e=document.querySelector("#editor");e.value="";e.dispatchEvent(new Event("input",{bubbles:true}));})()' });
  await wait(1200);
  await call("Page.close");
  const closeDeadline = Date.now() + 10000;
  while (child.exitCode === null && Date.now() < closeDeadline) await wait(100);
  result.closeExitCode = child.exitCode;
  if (child.exitCode !== 0) throw Error("CLI did not close normally");
  if (json) {
    const envelope = JSON.parse(stdout);
    result.envelopeOK = envelope.ok === true && envelope.command === "run" && envelope.result.exitCode === 0;
    if (!result.envelopeOK) throw Error("Stdout did not contain one successful run envelope");
  }
  result.temporaryConfigRemoved = !(await readdir(project)).includes(runtimeFiles[0]);
  if (!result.temporaryConfigRemoved) throw Error("Temporary config remained");
  for (const [file, digest] of Object.entries(hashes)) {
    if (sha256(await readFile(join(bin, file))) !== digest) throw Error("Binary changed during smoke: " + file);
  }
  result.passed = true;
} catch (error) {
  result.error = error instanceof Error ? error.message : String(error);
} finally {
  ws?.close();
  if (child.pid && child.exitCode === null) {
    const stop = spawnSync("taskkill.exe", ["/PID", String(child.pid), "/T", "/F"], { windowsHide: true, encoding: "utf8" });
    result.cleanupExitCode = stop.status;
    result.passed = false;
  } else result.cleanupExitCode = child.exitCode;
  result.finishedAtUtc = new Date().toISOString();
  await writeFile(join(work, "stdout.log"), stdout);
  await writeFile(join(work, "stderr.log"), stderr);
  await writeFile(join(work, "result.json"), JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result));
  console.log("manifest-watch-evidence=" + join(work, "result.json"));
}
if (!result.passed) process.exitCode = 1;
