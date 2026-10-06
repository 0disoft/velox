import { createRequire } from "node:module";
import { createHash } from "node:crypto";
import { createServer } from "node:net";
import { createServer as createHTTPServer } from "node:http";
import { spawn, spawnSync } from "node:child_process";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve, join } from "node:path";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const { chromium } = require("playwright");
const args = process.argv.slice(2);
if (args.length > 1 || args.some(arg => !["--manual", "--manual-check"].includes(arg))) {
  throw Error("Usage: node scripts/tray-app-smoke.mjs [--manual|--manual-check]");
}
const manual = args.length === 1;
const manualCheck = args[0] === "--manual-check";
const root = resolve(fileURLToPath(new URL("..", import.meta.url)));
const bin = resolve(root, process.env.VELOX_TRAY_STARTER_BIN_DIR ?? ".cache/manifest-watch-bin");
const work = join(root, ".cache", "tray-starter-" + Date.now());
const project = join(work, "my-tray"), output = join(work, "output");
await mkdir(work, { recursive: true });
const digest = bytes => createHash("sha256").update(bytes).digest("hex");
const cli = join(bin, "velox.exe");
const hashes = { cli: digest(await readFile(cli)), host: digest(await readFile(join(bin, "velox-host.exe"))) };
function runCLI(args) {
  const child = spawnSync(cli, args, { cwd: root, encoding: "utf8", windowsHide: true, timeout: 20000 });
  if (child.error || child.status !== 0) throw Error("CLI failed: " + args[0] + ": " + (child.error?.message ?? child.stderr));
  const envelope = JSON.parse(child.stdout);
  if (!envelope.ok) throw Error("CLI envelope failed: " + args[0]);
  return envelope.result;
}
const generated = runCLI(["init", project, "--template", "tray-app", "--json"]);
const configPath = join(project, "velox.json");
const manifest = JSON.parse(await readFile(configPath, "utf8"));
if (JSON.stringify(manifest.security.permissions) !== '["notification.show"]' || !manifest.window.tray || !manifest.app.singleInstance) throw Error("Unexpected generated permissions/settings");
runCLI(["validate", "--config", configPath, "--json"]);
const built = runCLI(["build", "--config", configPath, "--out", output, "--json"]);
runCLI(["inspect", join(output, manifest.app.id + ".zip"), "--json"]);
const exe = join(output, manifest.app.id, manifest.app.id + ".exe");
if (digest(await readFile(exe)) !== hashes.host) throw Error("Generated app modified the host");
const result = { startedAtUtc: new Date().toISOString(), mode: args[0] ?? "automated", hashes, generatedFiles: generated.files,
  build: built, views: [], native: {}, passed: false };
const wait = ms => new Promise(resolve => setTimeout(resolve, ms));
let preview, previewServer, nativeBrowser, child;
let stdout = "", stderr = "", spawnError;
try {
  const previewFiles = new Map();
  for (const [file, type] of [["index.html", "text/html"], ["style.css", "text/css"], ["app.js", "text/javascript"], ["bell.svg", "image/svg+xml"]]) {
    previewFiles.set("/" + file, { type, data: await readFile(join(project, "web", file)) });
  }
  previewServer = createHTTPServer((request, response) => {
    const file = previewFiles.get(request.url === "/" ? "/index.html" : request.url);
    if (!file) { response.writeHead(404).end(); return; }
    response.writeHead(200, { "Content-Type": file.type }).end(file.data);
  });
  await new Promise(resolve => previewServer.listen(0, "127.0.0.1", resolve));
  const previewURL = "http://127.0.0.1:" + previewServer.address().port;
  preview = await chromium.launch({ channel: "msedge", headless: true });
  for (const colorScheme of ["light", "dark"]) {
    for (const viewport of [{ width: 620, height: 480 }, { width: 360, height: 540 }]) {
      const context = await preview.newContext({ colorScheme, viewport });
      await context.addInitScript(() => {
        window.__trayCalls = [];
        window.velox = { invoke: async (method, params) => { window.__trayCalls.push({ method, params }); return null; } };
      });
      const page = await context.newPage();
      await page.goto(previewURL);
      await page.waitForFunction(() => document.documentElement.dataset.velox === "ready");
      const send = page.getByRole("button", { name: "Send notification" });
      if (!await send.isDisabled()) throw Error("Empty message can submit");
      await page.getByRole("textbox", { name: "Message", exact: true }).fill("\uD55C\uAE00 <literal message>");
      await page.getByLabel("Kind", { exact: true }).selectOption("warning");
      await send.focus();
      await page.keyboard.press("Space");
      await page.getByRole("status").filter({ hasText: "Request accepted." }).waitFor();
      const measured = await page.evaluate(async () => {
        await document.querySelector("#send-notification img").decode();
        const button = document.querySelector("#send-notification").getBoundingClientRect();
        return { overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
          button: { width: button.width, height: button.height }, calls: window.__trayCalls,
          iconWidth: document.querySelector("#send-notification img").naturalWidth,
          iconMask: getComputedStyle(document.querySelector("#send-notification"), "::before").maskImage,
          text: document.querySelector("#message").value,
          focused: document.activeElement.id };
      });
      if (measured.overflow || measured.button.width !== 40 || measured.button.height !== 40 ||
          measured.iconWidth !== 24 || !measured.iconMask.includes("bell.svg") || measured.calls.length !== 1 || measured.calls[0].method !== "notification.show" ||
          measured.calls[0].params.kind !== "warning" || measured.text !== "\uD55C\uAE00 <literal message>" || measured.focused !== "send-notification") throw Error("Preview layout/interaction failed");
      const screenshot = join(work, `preview-${colorScheme}-${viewport.width}.png`);
      await page.screenshot({ path: screenshot, fullPage: true });
      result.views.push({ colorScheme, viewport, ...measured, screenshot });
      await context.close();
    }
  }
  await preview.close();
  preview = undefined;
  await new Promise((resolve, reject) => previewServer.close(error => error ? reject(error) : resolve()));
  previewServer = undefined;

  const server = createServer();
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  const port = server.address().port;
  await new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve()));
  const env = { ...process.env, VELOX_DATA_DIR: join(work, "profile"),
    WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-address=127.0.0.1 --remote-debugging-port=" + port };
  child = spawn(exe, [], { cwd: root, env, windowsHide: !manual, stdio: ["ignore", "pipe", "pipe"] });
  child.on("error", error => { spawnError = error; });
  child.stdout.on("data", bytes => { stdout = (stdout + bytes.toString()).slice(-32768); });
  child.stderr.on("data", bytes => { stderr = (stderr + bytes.toString()).slice(-32768); });
  const readyDeadline = Date.now() + 20000;
  let endpointReady = false;
  while (!endpointReady && Date.now() < readyDeadline) {
    if (spawnError) throw spawnError;
    if (child.exitCode !== null) throw Error("Packaged host exited before inspection");
    try {
      const response = await fetch("http://127.0.0.1:" + port + "/json/list", { signal: AbortSignal.timeout(500) });
      const pages = await response.json();
      endpointReady = pages.some(page => page.type === "page" && page.url.includes(".app.invalid"));
    } catch {}
    if (!endpointReady) await wait(100);
  }
  if (!endpointReady) throw Error("No isolated native tray target");
  nativeBrowser = await chromium.connectOverCDP("http://127.0.0.1:" + port);
  const page = nativeBrowser.contexts().flatMap(context => context.pages()).find(page => page.url().includes(".app.invalid"));
  if (!page) throw Error("Missing native page");
  await page.waitForFunction(() => document.documentElement.dataset.velox === "ready");
  await page.getByRole("textbox", { name: "Message", exact: true }).waitFor({ state: "visible" });
  result.native.inputVisible = true;
  if (manual) {
    result.native.manualChecks = manualCheck ? "not-performed" : "pending-user-observation";
    result.native.screenshot = join(work, "native-manual-ready.png");
    await page.screenshot({ path: result.native.screenshot });
    await writeFile(join(work, "result.json"), JSON.stringify(result, null, 2));
    if (!manualCheck) {
      console.log("manual-tray-ready=" + exe);
      console.log("manual-tray-receipt=" + join(work, "result.json"));
      const deadline = Date.now() + 600000;
      while (child.exitCode === null && child.signalCode === null && Date.now() < deadline) await wait(100);
      if (child.signalCode !== null) throw Error("Manual tray process terminated by signal " + child.signalCode);
      if (child.exitCode === null) throw Error("Manual tray check timed out after 10 minutes");
    }
  } else {
    await page.evaluate(() => { window.__trayDocument = "same-document"; });
    await page.getByRole("textbox", { name: "Message", exact: true }).fill("Velox tray starter verification.");
    await page.getByRole("button", { name: "Send notification" }).click();
    await page.getByRole("status").filter({ hasText: "Request accepted." }).waitFor();
    result.native.requestAccepted = true;
    // The second invocation uses the exact same isolated app/profile identity.
    const second = spawnSync(exe, [], { cwd: root, env, windowsHide: true, encoding: "utf8", timeout: 10000 });
    if (second.error || second.status !== 0 || child.exitCode !== null) throw Error("Single-instance activation failed");
    const retained = await page.evaluate(() => ({ marker: window.__trayDocument, text: document.querySelector("#message").value }));
    if (retained.marker !== "same-document" || retained.text !== "Velox tray starter verification.") throw Error("Second launch replaced the document");
    result.native.singleInstanceRetainedDocument = true;
  }
  if (child.exitCode === null) {
    const session = await page.context().newCDPSession(page);
    await session.send("Page.close");
  }
  const closeDeadline = Date.now() + 10000;
  while (child.exitCode === null && Date.now() < closeDeadline) await wait(100);
  result.native.closeExitCode = child.exitCode;
  if (child.exitCode !== 0) throw Error("Packaged tray app did not close normally");
  if (digest(await readFile(cli)) !== hashes.cli || digest(await readFile(exe)) !== hashes.host) throw Error("Input/output binary changed during smoke");
  result.passed = true;
} catch (error) {
  result.error = error instanceof Error ? error.message : String(error);
} finally {
  await preview?.close().catch(() => {});
  if (previewServer) await new Promise(resolve => previewServer.close(resolve));
  await nativeBrowser?.close().catch(() => {});
  if (child?.pid && child.exitCode === null && child.signalCode === null) {
    const stop = spawnSync("taskkill.exe", ["/PID", String(child.pid), "/T", "/F"], { windowsHide: true, encoding: "utf8" });
    result.native.cleanupExitCode = stop.status;
    result.passed = false;
  } else if (child) result.native.cleanupExitCode = child.exitCode;
  result.finishedAtUtc = new Date().toISOString();
  await writeFile(join(work, "stdout.log"), stdout);
  await writeFile(join(work, "stderr.log"), stderr);
  await writeFile(join(work, "result.json"), JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result));
  console.log("tray-starter-evidence=" + join(work, "result.json"));
}
if (!result.passed) process.exitCode = 1;
