import { createHash } from "node:crypto";
import { createServer } from "node:net";
import { spawn, spawnSync } from "node:child_process";
import { cp, mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve, join } from "node:path";

type State = { html: string; js: string; css: string; origin: string };
type Message = { id?: number; method?: string; params?: { type?: string }; error?: { message: string }; result?: { result?: { value?: State | string } } };
const root = resolve(import.meta.dir, "..");
const release = resolve(root, process.env.VELOX_RELOAD_RELEASE_DIR ?? "dist/release/velox-windows-x64");
const watch = process.env.VELOX_RELOAD_WATCH === "1";
const visualAssets = process.env.VELOX_RELOAD_VISUAL_ASSETS === "1";
if (visualAssets && !watch) throw Error("Visual asset smoke requires watch mode");
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
const image = join(web, "watch-proof.svg"), font = join(web, "fonts", "watch-proof.ttf");
const visualMarkup = visualAssets
  ? '<link rel="stylesheet" href="watch-proof.css"><img id="watch-image" src="watch-proof.svg"><script src="watch-proof.js"></script>'
  : "";
async function changeImage(color: string) {
  await writeFile(image, '<svg xmlns="http://www.w3.org/2000/svg" width="2" height="2"><rect width="2" height="2" fill="' + color + '"/></svg>');
}
if (visualAssets) {
  await changeImage("red");
  await cp(join(web, "fonts", "NotoSansKR.ttf"), font);
  await writeFile(join(web, "watch-proof.css"), '@font-face{font-family:WatchProofFont;src:url("fonts/watch-proof.ttf")}#watch-image{position:fixed;bottom:0;right:0;width:2px;height:2px}');
  await writeFile(join(web, "watch-proof.js"), "window.__watchAssetDocument=crypto.randomUUID();");
}
const phases = [
  { name: "before", color: "rgb(200, 10, 20)" },
  { name: "after-one", color: "rgb(10, 120, 30)" },
  { name: "after-two", color: "rgb(20, 30, 180)" },
];
async function change(phase: typeof phases[number]) {
  await writeFile(index, original.replace("</body>",
    '<div id="reload-proof" data-phase="' + phase.name + '"></div><link rel="stylesheet" href="reload-proof.css"><script src="reload-proof.js"></script>' + visualMarkup + '</body>'));
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
type VisualState = { documentId: string; imagePixel: number[]; fontLoaded: boolean; fontWidth: number; origin: string };
const result: {
  version: string; hashes: Record<string, string>; startedAtUtc: string; finishedAtUtc?: string;
  before?: State; cycles: { phase: string; ignoreCache: false; observed?: State; passed: boolean }[];
  mode: string; canceledReloadPreservedInput?: boolean; normalCloseExitCode?: number | null;
  passed: boolean; error?: string; cleanupExitCode?: number | null;
  visualAssets?: { before: VisualState; imageOnly: VisualState; fontOnly: VisualState };
} = { mode: watch ? "watch-with-debug-off" : "manual-debug", version: manifest.releaseVersion, hashes, startedAtUtc: new Date().toISOString(), cycles: [], passed: false };
const child = spawn(join(release, "velox.exe"), ["run", "--config", join(project, "velox.json"), watch ? "--watch" : "--debug", "--json"], {
  cwd: root, env: { ...process.env, VELOX_DATA_DIR: join(work, "profile"),
    WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-address=127.0.0.1 --remote-debugging-port=" + port },
  stdio: ["ignore", "pipe", "pipe"], windowsHide: true,
});
let output = "", spawnError: Error | undefined, ws: WebSocket | undefined, id = 0;
child.on("error", error => { spawnError = error; });
for (const stream of [child.stdout, child.stderr]) stream.on("data", bytes => { output = (output + bytes.toString()).slice(-32768); });
const wait = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
const pending = new Map<number, (message: Message) => void>();
let beforeUnloadOpen = false;
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
    if (message.method === "Page.javascriptDialogOpening" && message.params?.type === "beforeunload") beforeUnloadOpen = true;
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
    return response.result?.result?.value as State | undefined;
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
    if (!watch) await call("Page.reload", { ignoreCache: false });
    const observed = await observe(phase, result.before!.origin);
    const passed = matches(observed, phase) && observed?.origin === result.before!.origin;
    result.cycles.push({ phase: phase.name, ignoreCache: false, observed, passed });
    if (!passed) throw Error("Normal reload failed: " + phase.name);
  }
  if (visualAssets) {
    const visualState = async () => {
      const response = await call("Runtime.evaluate", {
        expression: `(async()=>{
          const loaded=await document.fonts.load('20px "WatchProofFont"');
          const image=document.getElementById("watch-image");await image.decode();
          const canvas=document.createElement("canvas");canvas.width=2;canvas.height=2;
          const context=canvas.getContext("2d");context.drawImage(image,0,0);
          const imagePixel=Array.from(context.getImageData(0,0,1,1).data);
          context.font='20px "WatchProofFont"';
          return {documentId:window.__watchAssetDocument,imagePixel,fontLoaded:loaded.length===1,
            fontWidth:context.measureText("WWWWiiii0123456789").width,origin:location.origin};
        })()`,
        awaitPromise: true, returnByValue: true,
      });
      return response.result?.result?.value as unknown as VisualState | undefined;
    };
    async function observeVisual(previous: VisualState, matches: (value: VisualState) => boolean) {
      const deadline = Date.now() + 8000;
      do {
        try {
          const value = await visualState();
          if (value && value.documentId !== previous.documentId && value.origin === previous.origin && value.fontLoaded && matches(value)) return value;
        } catch {}
        await wait(100);
      } while (Date.now() < deadline);
      throw Error("Visual asset edit did not render through automatic reload");
    }
    const before = await visualState();
    if (!before?.fontLoaded || !before.documentId || before.imagePixel.join(",") !== "255,0,0,255") throw Error("Visual baseline not rendered: " + JSON.stringify(before));
    // Change only the referenced asset, keeping HTML, CSS and JS untouched.
    await changeImage("lime");
    const imageOnly = await observeVisual(before, value => value.imagePixel.join(",") === "0,255,0,255");
    await cp(join(process.env.WINDIR ?? "C:/Windows", "Fonts", "arial.ttf"), font);
    const fontOnly = await observeVisual(imageOnly, value => value.fontWidth !== imageOnly.fontWidth && value.imagePixel.join(",") === "0,255,0,255");
    result.visualAssets = { before, imageOnly, fontOnly };
  }
  if (watch) {
    await call("Page.enable");
    // Trusted input gives beforeunload the same sticky activation as editing
    // the real textarea. No test-side navigation or cache override is used.
    await call("Input.dispatchMouseEvent", { type: "mousePressed", x: 200, y: 300, button: "left", clickCount: 1 });
    await call("Input.dispatchMouseEvent", { type: "mouseReleased", x: 200, y: 300, button: "left", clickCount: 1 });
    await call("Input.insertText", { text: "watch protection" });
    const blocked = { name: "after-cancel", color: "rgb(30, 140, 160)" };
    await change(blocked);
    const deadline = Date.now() + 8000;
    while (!beforeUnloadOpen && Date.now() < deadline) await wait(100);
    if (!beforeUnloadOpen) throw Error("Watch reload did not obtain beforeunload consent");
    await call("Page.handleJavaScriptDialog", { accept: false });
    const old = await state();
    const editor = await call("Runtime.evaluate", { expression: 'document.querySelector("#editor").value', returnByValue: true });
    result.canceledReloadPreservedInput = matches(old, phases[2]) && editor.result?.result?.value === "watch protection";
    if (!result.canceledReloadPreservedInput) throw Error("Canceled reload discarded input");
    await call("Runtime.evaluate", { expression: '(()=>{const e=document.querySelector("#editor");e.value="";e.dispatchEvent(new Event("input",{bubbles:true}));})()' });
    await wait(1200);
    const retry = { name: "after-retry", color: "rgb(30, 140, 160)" };
    await change(retry);
    const observed = await observe(retry, result.before!.origin);
    const passed = matches(observed, retry) && observed?.origin === result.before!.origin;
    result.cycles.push({ phase: retry.name, ignoreCache: false, observed, passed });
    if (!passed) throw Error("Watch did not recover after canceled reload");
    await call("Page.close");
    const closeDeadline = Date.now() + 10000;
    while (child.exitCode === null && Date.now() < closeDeadline) await wait(100);
    result.normalCloseExitCode = child.exitCode;
    if (child.exitCode !== 0) throw Error("Watch process did not exit normally");
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
  } else if (watch && child.exitCode === 0) {
    result.cleanupExitCode = 0;
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
