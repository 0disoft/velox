import { createHash } from "node:crypto";
import { createServer } from "node:net";
import { spawn, spawnSync } from "node:child_process";
import { cp, mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve, join } from "node:path";

const root = resolve(import.meta.dir, "..");
const release = resolve(root, process.env.VELOX_DIAGNOSTICS_RELEASE_DIR ?? ".cache/dev-diagnostics-bundle/velox-windows-x64");
const manifest = JSON.parse(await readFile(join(release, "release-manifest.json"), "utf8"));
const hashes: Record<string, string> = {};
for (const file of ["velox.exe", "velox-host.exe"]) {
  const bytes = await readFile(join(release, file));
  const digest = createHash("sha256").update(bytes).digest("hex");
  const entry = manifest.artifacts.find((item: { file: string }) => item.file === file);
  if (!entry || entry.sha256 !== digest || entry.bytes !== bytes.length) throw Error("Artifact mismatch: " + file);
  hashes[file] = digest;
}
const work = join(root, ".cache", "development-diagnostics-" + Date.now());
await mkdir(work, { recursive: true });
const wait = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
type Message = { id?: number; error?: { message: string }; result?: { result?: { value?: any } } };
type Result = { debug: boolean; bindingPresent?: boolean; envelopeOK?: boolean; diagnosticCount?: number; closeExitCode?: number | null; cleanupExitCode?: number | null; passed: boolean; error?: string };
const results: Result[] = [];
for (const debug of [false, true]) {
  const mode = debug ? "debug" : "normal";
  const project = join(work, mode);
  await cp(join(root, "examples/hello"), project, { recursive: true });
  const web = join(project, "web"), index = join(web, "index.html");
  const html = await readFile(index, "utf8");
  await writeFile(index, html.replace("</body>", '<script src="failure.js?token=SECRET_URL#SECRET_FRAGMENT"></script></body>'));
  await writeFile(join(web, "failure.js"), 'setTimeout(() => { throw new Error("SECRET_ERROR_BODY"); }, 300);\nsetTimeout(() => { Promise.reject(new Error("SECRET_REJECTION_BODY")); }, 500);');
  const server = createServer();
  await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  if (!address || typeof address === "string") throw Error("No diagnostic port");
  const port = address.port;
  await new Promise<void>((resolve, reject) => server.close(error => error ? reject(error) : resolve()));
  const child = spawn(join(release, "velox.exe"), ["run", "--config", join(project, "velox.json"), "--json", ...(debug ? ["--debug"] : [])], {
    cwd: root, env: { ...process.env, VELOX_DATA_DIR: join(project, "profile"),
      WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-address=127.0.0.1 --remote-debugging-port=" + port },
    stdio: ["ignore", "pipe", "pipe"], windowsHide: true,
  });
  let stdout = "", stderr = "", spawnError: Error | undefined, ws: WebSocket | undefined, nextID = 0;
  child.on("error", error => { spawnError = error; });
  child.stdout.on("data", bytes => { stdout = (stdout + bytes.toString()).slice(-32768); });
  child.stderr.on("data", bytes => { stderr = (stderr + bytes.toString()).slice(-32768); });
  const result: Result = { debug, passed: false };
  const pending = new Map<number, (message: Message) => void>();
  try {
    let target: { webSocketDebuggerUrl: string } | undefined;
    const deadline = Date.now() + 20000;
    while (!target && Date.now() < deadline) {
      if (spawnError) throw spawnError;
      if (child.exitCode !== null) throw Error("Host exited before inspection");
      try {
        const pages = await (await fetch("http://127.0.0.1:" + port + "/json/list", { signal: AbortSignal.timeout(500) })).json();
        target = pages.find((page: { type: string; url: string }) => page.type === "page" && new URL(page.url).hostname.endsWith(".app.invalid"));
      } catch {}
      if (!target) await wait(100);
    }
    if (!target) throw Error("No private diagnostics target");
    ws = new WebSocket(target.webSocketDebuggerUrl);
    await new Promise<void>((resolve, reject) => {
      const timer = setTimeout(() => reject(Error("WebSocket timeout")), 5000);
      ws!.onopen = () => { clearTimeout(timer); resolve(); };
      ws!.onerror = () => { clearTimeout(timer); reject(Error("WebSocket failed")); };
    });
    ws.onmessage = event => {
      const message: Message = JSON.parse(String(event.data));
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
    await wait(1000);
    const state = await call("Runtime.evaluate", { expression: '({binding:typeof window.__veloxDevDiagnostic==="function",heading:document.querySelector("h1")?.textContent,origin:location.origin})', returnByValue: true });
    result.bindingPresent = state.result?.result?.value?.binding;
    if (result.bindingPresent !== debug || !state.result?.result?.value?.heading) throw Error("Incorrect diagnostic registration or blank page");
    if (debug) {
      const logDeadline = Date.now() + 5000;
      while ((!stderr.includes("uncaught-error failure.js:1:") || !stderr.includes("unhandled-rejection <unknown>:0:0")) && Date.now() < logDeadline) await wait(100);
      if (!stderr.includes("uncaught-error failure.js:1:") || !stderr.includes("unhandled-rejection <unknown>:0:0")) throw Error("Native development metadata missing");
      // Deliberately invalid private fields and forged paths must never be echoed.
      await call("Runtime.evaluate", { expression: 'Promise.all([window.__veloxDevDiagnostic({kind:"uncaught-error",source:"failure.js",line:1,column:1,message:"SECRET_EXTRA"}),window.__veloxDevDiagnostic({kind:"uncaught-error",source:"file:///C:/SECRET_PATH",line:1,column:1}),...Array.from({length:50},()=>window.__veloxDevDiagnostic({kind:"uncaught-error",source:"failure.js",line:1,column:1}))])', awaitPromise: true });
      await wait(300);
    }
    result.diagnosticCount = stderr.split("\n").filter(line => line.startsWith("velox-debug:")).length;
    if ((debug && result.diagnosticCount !== 19) || (!debug && result.diagnosticCount !== 0)) throw Error("Diagnostic attempt budget was not enforced");
    if ((stdout + stderr).includes("SECRET_") || stderr.includes("C:/")) throw Error("Private content leaked to process output");
    await call("Page.close");
    const closeDeadline = Date.now() + 10000;
    while (child.exitCode === null && Date.now() < closeDeadline) await wait(100);
    result.closeExitCode = child.exitCode;
    if (child.exitCode !== 0) throw Error("Host did not close normally");
    const envelope = JSON.parse(stdout);
    result.envelopeOK = envelope.ok === true && envelope.command === "run" && envelope.result.exitCode === 0;
    if (!result.envelopeOK) throw Error("Stdout did not contain one successful run envelope");
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
    await writeFile(join(project, "stdout.log"), stdout);
    await writeFile(join(project, "stderr.log"), stderr);
    results.push(result);
  }
}
for (const [file, digest] of Object.entries(hashes)) {
  if (createHash("sha256").update(await readFile(join(release, file))).digest("hex") !== digest) throw Error("Artifact changed during smoke");
}
const receipt = { version: manifest.releaseVersion, hashes, results, passed: results.every(result => result.passed) };
await writeFile(join(work, "result.json"), JSON.stringify(receipt, null, 2));
console.log(JSON.stringify(receipt));
console.log("diagnostics-evidence=" + join(work, "result.json"));
if (!receipt.passed) process.exitCode = 1;
