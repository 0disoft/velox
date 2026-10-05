import { createHash } from "node:crypto";
import { createServer } from "node:net";
import { spawn, spawnSync } from "node:child_process";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve, join } from "node:path";

const root = resolve(import.meta.dir, "..");
const release = resolve(root, process.env.VELOX_DRAFT_RELEASE_DIR ?? ".cache/draft-recovery-bundle/velox-windows-x64");
const manifest = JSON.parse(await readFile(join(release, "release-manifest.json"), "utf8"));
const hashes: Record<string, string> = {};
for (const file of ["velox.exe", "velox-host.exe"]) {
  const bytes = await readFile(join(release, file));
  const digest = createHash("sha256").update(bytes).digest("hex");
  const entry = manifest.artifacts.find((item: { file: string }) => item.file === file);
  if (!entry || entry.sha256 !== digest || entry.bytes !== bytes.length) throw Error("Artifact mismatch: " + file);
  hashes[file] = digest;
}
const work = join(root, ".cache", "text-editor-draft-" + Date.now());
const project = join(work, "my-editor"), profile = join(work, "profile");
await mkdir(work, { recursive: true });
const cli = join(release, "velox.exe");
const init = spawnSync(cli, ["init", project, "--template", "text-editor", "--json"], { windowsHide: true, encoding: "utf8" });
if (init.status !== 0 || !JSON.parse(init.stdout).ok) throw Error("Init failed");
const wait = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
const text = "\ud55c\uae00 draft\nVelox recovery test";
type Message = { id?: number; method?: string; error?: { message: string }; result?: { result?: { value?: any } } };
const results: { mode: string; passed: boolean; closeExitCode?: number | null; cleanupExitCode?: number | null; error?: string }[] = [];
for (const mode of ["write", "restore-clear", "cleared-relaunch"]) {
  const server = createServer();
  await new Promise<void>(resolve => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  if (!address || typeof address === "string") throw Error("No diagnostic port");
  const port = address.port;
  await new Promise<void>((resolve, reject) => server.close(error => error ? reject(error) : resolve()));
  const child = spawn(cli, ["run", "--config", join(project, "velox.json"), "--json"], {
    cwd: root, env: { ...process.env, VELOX_DATA_DIR: profile,
      WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-address=127.0.0.1 --remote-debugging-port=" + port },
    stdio: ["ignore", "pipe", "pipe"], windowsHide: true,
  });
  let output = "", ws: WebSocket | undefined, nextID = 0;
  for (const stream of [child.stdout, child.stderr]) stream.on("data", bytes => { output = (output + bytes.toString()).slice(-32768); });
  const result: typeof results[number] = { mode, passed: false };
  const pending = new Map<number, (message: Message) => void>();
  try {
    let target: { webSocketDebuggerUrl: string } | undefined;
    const deadline = Date.now() + 30000;
    while (!target && Date.now() < deadline) {
      if (child.exitCode !== null) throw Error("Host exited before inspection");
      try {
        const pages = await (await fetch("http://127.0.0.1:" + port + "/json/list", { signal: AbortSignal.timeout(500) })).json();
        target = pages.find((page: { type: string; url: string }) => page.type === "page" && new URL(page.url).hostname.endsWith(".app.invalid"));
      } catch {}
      if (!target) await wait(100);
    }
    if (!target) throw Error("No private draft target");
    ws = new WebSocket(target.webSocketDebuggerUrl);
    await new Promise<void>((resolve, reject) => {
      const timer = setTimeout(() => reject(Error("WebSocket timeout")), 5000);
      ws!.onopen = () => { clearTimeout(timer); resolve(); };
      ws!.onerror = () => { clearTimeout(timer); reject(Error("WebSocket failed")); };
    });
    const call = (method: string, params: object = {}) => new Promise<Message>((resolve, reject) => {
      const id = ++nextID;
      const timer = setTimeout(() => { pending.delete(id); reject(Error("CDP timeout: " + method)); }, 5000);
      pending.set(id, message => { clearTimeout(timer); message.error ? reject(Error(message.error.message)) : resolve(message); });
      ws!.send(JSON.stringify({ id, method, params }));
    });
    ws.onmessage = event => {
      const message: Message = JSON.parse(String(event.data));
      if (message.method === "Page.javascriptDialogOpening") void call("Page.handleJavaScriptDialog", { accept: true }).catch(() => {});
      if (message.id && pending.has(message.id)) { pending.get(message.id)!(message); pending.delete(message.id); }
    };
    const evaluate = async (expression: string) => (await call("Runtime.evaluate", { expression, returnByValue: true, awaitPromise: true })).result?.result?.value;
    async function until(expression: string) {
      const deadline = Date.now() + 8000;
      while (Date.now() < deadline) {
        if (await evaluate(expression)) return;
        await wait(100);
      }
      const state = await evaluate('({draft:document.querySelector("#draft-state")?.textContent,status:document.querySelector("#status")?.textContent,length:document.querySelector("#editor")?.value.length,active:document.activeElement?.id})');
      throw Error("Draft assertion timed out: " + expression + " state=" + JSON.stringify(state));
    }
    await call("Page.enable");
    if (mode === "write") {
      await until('document.querySelector("#editor") && !document.querySelector("#editor").readOnly');
      await evaluate('document.querySelector("#editor").focus()');
      await call("Input.dispatchMouseEvent", { type: "mousePressed", x: 200, y: 250, button: "left", clickCount: 1 });
      await call("Input.dispatchMouseEvent", { type: "mouseReleased", x: 200, y: 250, button: "left", clickCount: 1 });
      await call("Input.insertText", { text });
      await until('document.querySelector("#draft-state").textContent === "Draft stored"');
      const valid = await evaluate('(async()=>{const d=await window.EditorDrafts.load();return d.text===' + JSON.stringify(text) + '&&Object.keys(d).sort().join(",")==="name,schemaVersion,text,updatedAt"})()');
      if (!valid) throw Error("Draft content/authority boundary mismatch");
    } else if (mode === "restore-clear") {
      await until('document.querySelector("#recovery-dialog")?.open');
      await evaluate('document.querySelector("#recovery-dialog button[value=recover]").click()');
      await until('document.querySelector("#draft-state").textContent === "Draft stored"');
      if (!await evaluate('document.querySelector("#editor").value===' + JSON.stringify(text) + '&&document.querySelector("#save-state").dataset.dirty==="true"')) throw Error("Restored draft not dirty or literal");
      await evaluate('document.querySelector("#new-document").click()');
      await until('document.querySelector("#discard-dialog").open');
      await evaluate('document.querySelector("#discard-dialog button[value=discard]").click()');
      await until('document.querySelector("#status").textContent === "New document."');
      if (await evaluate('window.EditorDrafts.load()') !== null) throw Error("Draft was not durably cleared");
    } else {
      await until('document.querySelector("#editor") && !document.querySelector("#editor").readOnly');
      if (!await evaluate('!document.querySelector("#recovery-dialog").open&&document.querySelector("#editor").value===""')) throw Error("Discarded draft resurrected");
    }
    await call("Page.close");
    const closeDeadline = Date.now() + 10000;
    while (child.exitCode === null && Date.now() < closeDeadline) await wait(100);
    result.closeExitCode = child.exitCode;
    if (child.exitCode !== 0 || !JSON.parse(output).ok) throw Error("Host did not close with one successful envelope");
    result.passed = true;
  } catch (error) { result.error = error instanceof Error ? error.message : String(error); }
  finally {
    ws?.close();
    if (child.pid && child.exitCode === null) {
      result.cleanupExitCode = spawnSync("taskkill.exe", ["/PID", String(child.pid), "/T", "/F"], { windowsHide: true }).status;
      result.passed = false;
    } else result.cleanupExitCode = child.exitCode;
    await writeFile(join(work, mode + ".log"), output);
    results.push(result);
  }
  if (!result.passed) break;
}
const receipt = { version: manifest.releaseVersion, hashes, results, passed: results.length === 3 && results.every(result => result.passed) };
await writeFile(join(work, "result.json"), JSON.stringify(receipt, null, 2));
console.log(JSON.stringify(receipt));
console.log("draft-evidence=" + join(work, "result.json"));
if (!receipt.passed) process.exitCode = 1;
