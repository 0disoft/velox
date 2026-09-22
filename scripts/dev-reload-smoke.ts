import { createHash } from "node:crypto";
import { createServer } from "node:net";
import { spawn, spawnSync } from "node:child_process";
import { cp, mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve, join } from "node:path";

type State = { html: string; js: string; css: string; origin: string };
type Message = { id?: number; error?: { message: string }; result?: { result?: { value?: State } } };
const root = resolve(import.meta.dir, "..");
const release = resolve(root, process.env.VELOX_RELOAD_RELEASE_DIR ?? "dist/release/velox-windows-x64");
const manifest = JSON.parse(await readFile(join(release, "release-manifest.json"), "utf8"));
const hashes: Record<string, string> = {};
for (const file of ["velox.exe", "velox-host.exe"]) {
  const entry = manifest.artifacts.find((item: { file: string }) => item.file === file);
  const bytes = await readFile(join(release, file));
  const digest = createHash("sha256").update(bytes).digest("hex");
  if (!entry || entry.bytes !== bytes.length || entry.sha256 !== digest) throw Error("Artifact mismatch: " + file);
  hashes[file] = digest;
}
const work = join(root, ".cache", "normal-reload-" + Date.now());
const project = join(work, "project"), web = join(project, "web");
await mkdir(work, { recursive: true });
await cp(join(root, "examples/file-notes"), project, { recursive: true });
const index = join(web, "index.html"), original = await readFile(index, "utf8");
if (!original.includes("</body>")) throw Error("Missing HTML body");
const phases = [
  { name: "before", color: "rgb(200, 10, 20)" },
  { name: "after-one", color: "rgb(10, 120, 30)" },
  { name: "after-two", color: "rgb(20, 30, 180)" },
];
async function change(phase: typeof phases[number]) {
  await writeFile(index, original.replace("</body>",
    '<div id="reload-proof" data-phase="' + phase.name + '"></div><link rel="stylesheet" href="reload-proof.css"><script src="reload-proof.js"></script></body>'));
  await writeFile(join(web, "reload-proof.js"), 'document.getElementById("reload-proof").textContent=' + JSON.stringify(phase.name) + ";");
  await writeFile(join(web, "reload-proof.css"), "#reload-proof{color:" + phase.color + "}");
}
await change(phases[0]);
const server = createServer();
await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
const address = server.address();
if (!address || typeof address === "string") throw Error("No local diagnostic port");
const port = address.port;
await new Promise<void>((resolve, reject) => server.close(error => error ? reject(error) : resolve()));
const result: {
  version: string; hashes: Record<string, string>; startedAtUtc: string; finishedAtUtc?: string;
  before?: State; cycles: { phase: string; ignoreCache: false; observed?: State; passed: boolean }[];
  passed: boolean; error?: string; cleanupExitCode?: number | null;
} = { version: manifest.releaseVersion, hashes, startedAtUtc: new Date().toISOString(), cycles: [], passed: false };
const child = spawn(join(release, "velox.exe"), ["run", "--config", join(project, "velox.json"), "--debug", "--json"], {
  cwd: root, env: { ...process.env, VELOX_DATA_DIR: join(work, "profile"),
    WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-address=127.0.0.1 --remote-debugging-port=" + port },
  stdio: ["ignore", "pipe", "pipe"], windowsHide: true,
});
let output = "", spawnError: Error | undefined, ws: WebSocket | undefined, id = 0;
child.on("error", error => { spawnError = error; });
for (const stream of [child.stdout, child.stderr]) stream.on("data", bytes => { output = (output + bytes.toString()).slice(-32768); });
const wait = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
const pending = new Map<number, (message: Message) => void>();
try {
  let target: { webSocketDebuggerUrl: string } | undefined;
  const deadline = Date.now() + 20000;
  while (Date.now() < deadline) {
    if (spawnError) throw spawnError;
    try {
      const pages = await (await fetch("http://127.0.0.1:" + port + "/json/list", { signal: AbortSignal.timeout(500) })).json();
      target = pages.find((page: { type: string; url: string }) => page.type === "page" && new URL(page.url).hostname.endsWith(".app.invalid"));
      if (target) break;
    } catch {}
    await wait(100);
  }
  if (!target) throw Error("No private reload target");
  ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise<void>((resolve, reject) => {
    const timer = setTimeout(() => reject(Error("WebSocket timeout")), 5000);
    ws!.onopen = () => { clearTimeout(timer); resolve(); };
    ws!.onerror = () => { clearTimeout(timer); reject(Error("WebSocket error")); };
  });
  ws.onmessage = event => {
    const message: Message = JSON.parse(String(event.data));
    if (message.id && pending.has(message.id)) {
      pending.get(message.id)!(message);
      pending.delete(message.id);
    }
  };
  const call = (method: string, params: object = {}) => new Promise<Message>((resolve, reject) => {
    const number = ++id;
    const timer = setTimeout(() => { pending.delete(number); reject(Error("CDP timeout: " + method)); }, 5000);
    pending.set(number, message => {
      clearTimeout(timer);
      message.error ? reject(Error(message.error.message)) : resolve(message);
    });
    ws!.send(JSON.stringify({ id: number, method, params }));
  });
  const state = async () => {
    const response = await call("Runtime.evaluate", {
      expression: '(()=>{const n=document.getElementById("reload-proof");return n?{html:n.dataset.phase,js:n.textContent,css:getComputedStyle(n).color,origin:location.origin}:null})()',
      returnByValue: true,
    });
    return response.result?.result?.value;
  };
  const matches = (value: State | undefined, phase: typeof phases[number]) =>
    value?.html === phase.name && value.js === phase.name && value.css === phase.color;
  async function observe(phase: typeof phases[number], origin?: string) {
    const deadline = Date.now() + 6000;
    let value: State | undefined;
    do {
      try { value = await state(); } catch { value = undefined; }
      if (matches(value, phase) && (!origin || value?.origin === origin)) break;
      await wait(100);
    } while (Date.now() < deadline);
    return value;
  }
  result.before = await observe(phases[0]);
  if (!matches(result.before, phases[0])) throw Error("Baseline not rendered");
  for (const phase of phases.slice(1)) {
    await change(phase);
    // This regression must pass without a test-side cache override or hard reload.
    await call("Page.reload", { ignoreCache: false });
    const observed = await observe(phase, result.before!.origin);
    const passed = matches(observed, phase) && observed?.origin === result.before!.origin;
    result.cycles.push({ phase: phase.name, ignoreCache: false, observed, passed });
    if (!passed) throw Error("Normal reload failed: " + phase.name);
  }
  for (const [file, digest] of Object.entries(hashes)) {
    if (createHash("sha256").update(await readFile(join(release, file))).digest("hex") !== digest) throw Error("Artifact changed during test");
  }
  result.passed = true;
} catch (error) {
  result.error = error instanceof Error ? error.message : String(error);
} finally {
  ws?.close();
  if (child.pid && child.exitCode === null) {
    const stop = spawnSync("taskkill.exe", ["/PID", String(child.pid), "/T", "/F"], { windowsHide: true, encoding: "utf8" });
    result.cleanupExitCode = stop.status;
    if (stop.status !== 0) result.passed = false;
  } else {
    result.passed = false;
  }
  result.finishedAtUtc = new Date().toISOString();
  await writeFile(join(work, "runner.log"), output);
  await writeFile(join(work, "result.json"), JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result));
  console.log("reload-evidence=" + join(work, "result.json"));
}
if (!result.passed) process.exitCode = 1;
