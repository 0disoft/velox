import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { createHash, randomUUID } from "node:crypto";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { createServer } from "node:net";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const checkCleanTransition = process.argv[4] === "--check-clean-transition";
if (process.platform !== "win32" ||
    !(process.argv.length === 4 || (process.argv.length === 5 && checkCleanTransition))) {
  throw Error("Usage on Windows: node scripts/file-notes-packaged-recovery.mjs <velox.exe> <portable-directory> [--check-clean-transition]");
}
const root = fileURLToPath(new URL("..", import.meta.url));
const cli = resolve(process.argv[2]), directory = resolve(process.argv[3]);
const inspected = spawnSync(cli, ["inspect", directory, "--json"], {
  cwd: root, windowsHide: true, encoding: "utf8", timeout: 20000,
});
assert.equal(inspected.status, 0, inspected.error?.message ?? inspected.stderr);
const envelope = JSON.parse(inspected.stdout);
assert.equal(envelope.ok, true);
const artifact = envelope.result;
assert.equal(artifact.app.id, "dev.velox.filenotes");
const executable = join(directory, artifact.app.id + ".exe");
const digest = bytes => createHash("sha256").update(bytes).digest("hex");
assert.equal(digest(await readFile(executable)), artifact.hostSha256);
const { chromium } = createRequire(import.meta.url)("playwright");
const work = join(root, ".cache", "packaged-recovery-" + randomUUID());
// Preserve the isolated profile and receipt; never rename/delete a live profile.
const profile = join(work, "profile");
await mkdir(profile, { recursive: true });
const expected = "\uD55C\uAE00 draft recovery\nSecond line <literal>\n";
const result = { schemaVersion: "velox.packaged-file-notes-recovery/v1", artifact,
  startedAtUtc: new Date().toISOString(), profile, instrumentedCDP: true,
  nativeFileDialogs: "not-tested", launches: [], passed: false };
const wait = ms => new Promise(done => setTimeout(done, ms));
const launches = [], browsers = [];

function nativeClose(pid) {
  const command = `Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public static class VeloxClose { public delegate bool Callback(IntPtr h, IntPtr p); [DllImport("user32.dll")] public static extern bool EnumWindows(Callback c, IntPtr p); [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid); [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h); [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr h, uint m, IntPtr w, IntPtr l); public static bool Close(uint target) { IntPtr found=IntPtr.Zero; EnumWindows((h,p)=>{uint id;GetWindowThreadProcessId(h,out id);if(id==target && IsWindowVisible(h))found=h;return true;},IntPtr.Zero);return found!=IntPtr.Zero && PostMessage(found,0x10,IntPtr.Zero,IntPtr.Zero); } }'; if(-not [VeloxClose]::Close(${pid})){throw 'Owned visible window unavailable'}`;
  const closed = spawnSync("pwsh", ["-NoProfile", "-NonInteractive", "-Command", command], {
    windowsHide: true, encoding: "utf8", timeout: 15000,
  });
  assert.equal(closed.status, 0, closed.error?.message ?? closed.stderr);
}

async function observeBrowser(pid) {
  // Acquire a process handle while the identified browser is alive. Waiting on
  // that handle cannot confuse an exited browser with later PID reuse.
  const monitor = spawn("pwsh", ["-NoProfile", "-NonInteractive", "-Command",
    `$ErrorActionPreference='Stop'; $p=[System.Diagnostics.Process]::GetProcessById(${pid}); try { $null=$p.Handle; Write-Output 'ready'; if(-not $p.WaitForExit(30000)){throw 'Browser exit timeout'}; Write-Output 'exited' } finally { $p.Dispose() }`],
  { windowsHide: true, stdio: ["ignore", "pipe", "pipe"] });
  const observation = { monitor, output: "", error: "", exited: false };
  monitor.stdout.on("data", bytes => { observation.output += bytes; });
  monitor.stderr.on("data", bytes => { observation.error += bytes; });
  monitor.on("error", error => { observation.error += error.message; });
  observation.done = new Promise(done => monitor.on("close", code => {
    observation.exited = code === 0 && observation.output.includes("exited");
    done();
  }));
  const deadline = Date.now() + 10000;
  while (!observation.output.includes("ready") && Date.now() < deadline) {
    if (monitor.exitCode !== null || observation.error) break;
    await wait(25);
  }
  if (!observation.output.includes("ready")) {
    monitor.kill();
    await observation.done;
    throw Error("Could not retain browser exit handle: " + observation.error);
  }
  return observation;
}

async function launch() {
  const server = createServer();
  await new Promise(done => server.listen(0, "127.0.0.1", done));
  const port = server.address().port;
  await new Promise(done => server.close(done));
  const env = { ...process.env, VELOX_DATA_DIR: profile,
    LOCALAPPDATA: join(work, "local-app-data"),
    WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS: "--remote-debugging-address=127.0.0.1 --remote-debugging-port=" + port };
  for (const key of Object.keys(env)) {
    if (/^VELOX_(BENCHMARK|STARTUP)/i.test(key) || key === "WEBVIEW2_USER_DATA_FOLDER") delete env[key];
  }
  const child = spawn(executable, [], { cwd: directory, env,
    windowsHide: false, stdio: ["ignore", "pipe", "pipe"] });
  const item = { child, stdout: "", stderr: "", error: null, observation: null };
  launches.push(item);
  child.stdout.on("data", bytes => { item.stdout = (item.stdout + bytes).slice(-32768); });
  child.stderr.on("data", bytes => { item.stderr = (item.stderr + bytes).slice(-32768); });
  child.on("error", error => { item.error = error; });
  item.done = new Promise(done => child.on("close", done));
  const endpoint = "http://127.0.0.1:" + port;
  const deadline = Date.now() + 20000;
  let ready = false;
  while (!ready && Date.now() < deadline) {
    if (item.error) throw item.error;
    if (child.exitCode !== null) throw Error("Host exited before UI: " + item.stderr);
    try {
      const response = await fetch(endpoint + "/json/list", { signal: AbortSignal.timeout(500) });
      ready = (await response.json()).some(page => page.type === "page" && page.url.includes(".app.invalid"));
    } catch {}
    if (!ready) await wait(50);
  }
  assert.equal(ready, true, "Packaged WebView2 target unavailable");
  const browser = await chromium.connectOverCDP(endpoint);
  browsers.push(browser);
  const session = await browser.newBrowserCDPSession();
  result.webView2Version = (await session.send("Browser.getVersion")).product;
  const processes = (await session.send("SystemInfo.getProcessInfo")).processInfo;
  const pid = processes.find(process => process.type === "browser")?.id;
  assert.ok(Number.isSafeInteger(pid) && pid > 0, "No observed browser PID");
  item.observation = await observeBrowser(pid);
  item.browserPID = pid;
  const page = browser.contexts().flatMap(context => context.pages()).find(page => page.url().includes(".app.invalid"));
  assert.ok(page, "Missing packaged page");
  page.setDefaultTimeout(10000);
  await page.waitForFunction(() => document.querySelector("#editor")?.readOnly === false);
  item.page = page;
  return item;
}

async function closeWithConsent(item, accept) {
  const dialogEvent = item.page.waitForEvent("dialog");
  dialogEvent.catch(() => {});
  nativeClose(item.child.pid);
  const dialog = await dialogEvent;
  assert.equal(dialog.type(), "beforeunload");
  if (accept) await dialog.accept(); else await dialog.dismiss();
}

async function waitHost(item) {
  let timer;
  try {
    await Promise.race([item.done, new Promise((_, reject) => {
      timer = setTimeout(() => reject(Error("Host did not exit after accepted close")), 10000);
    })]);
  } finally { clearTimeout(timer); }
  assert.equal(item.child.exitCode, 0, "Nonzero packaged host exit");
  assert.equal(item.child.signalCode, null);
}

async function closeClean(item) {
  let dialogs = 0;
  const unexpected = async dialog => { dialogs++; await dialog.dismiss(); };
  item.page.on("dialog", unexpected);
  try {
    nativeClose(item.child.pid);
    await waitHost(item);
    assert.equal(dialogs, 0, "A settled clean document unexpectedly requested consent");
  } finally { item.page.off("dialog", unexpected); }
}

async function closeNewDocument(item) {
  await item.page.evaluate(() => new Promise((resolve, reject) => {
    const opened = indexedDB.open("dev.velox.filenotes", 1);
    opened.onerror = () => reject(Error("Could not open the controlled draft transaction"));
    opened.onsuccess = () => {
      const database = opened.result;
      const transaction = database.transaction("drafts", "readwrite");
      const store = transaction.objectStore("drafts");
      const deadline = Date.now() + 15000;
      let first = true;
      window.__holdDraftTransaction = true;
      window.__draftTransactionReleased = false;
      transaction.oncomplete = () => { database.close(); window.__draftTransactionReleased = true; };
      transaction.onerror = () => { database.close(); reject(Error("Controlled draft transaction failed")); };
      transaction.onabort = transaction.onerror;
      // Reads keep this actual readwrite transaction alive without changing a row.
      const read = () => {
        const request = store.get("current");
        request.onsuccess = () => {
          if (first) { first = false; resolve(); }
          if (window.__holdDraftTransaction && Date.now() < deadline) read();
        };
      };
      read();
    };
  }));
  await item.page.getByRole("button", { name: "New", exact: true }).click();
  await item.page.getByRole("button", { name: "Discard changes", exact: true }).click();
  await item.page.waitForFunction(() => document.querySelector("#editor").value === "" &&
    !document.querySelector("#new-document").disabled && !document.querySelector("#editor").readOnly);
  assert.equal(await item.page.locator("#editor").inputValue(), "");
  let consent;
  let timer;
  const handled = new Promise((resolve, reject) => {
    timer = setTimeout(() => reject(Error("Clean transition neither closed nor offered consent")), 10000);
    consent = async dialog => {
      try {
        assert.equal(dialog.type(), "beforeunload");
        await dialog.dismiss();
        resolve("canceled");
      } catch (error) { reject(error); }
    };
  });
  item.page.on("dialog", consent);
  let outcome;
  try {
    nativeClose(item.child.pid);
    outcome = await Promise.race([handled, item.done.then(() => "exited")]);
  } finally {
    clearTimeout(timer);
    item.page.off("dialog", consent);
  }
  if (outcome === "canceled") {
    assert.equal(item.child.exitCode, null);
    assert.equal(await item.page.locator("#editor").inputValue(), "");
    assert.equal(await item.page.evaluate(() => window.__draftTransactionReleased), false);
    await item.page.evaluate(() => { window.__holdDraftTransaction = false; });
    await item.page.waitForFunction(() => document.querySelector("#draft-state").textContent === "Draft saved");
    const saved = await item.page.evaluate(() => window.FileNotesStorage.load());
    assert.equal(saved.text, "");
    assert.equal(saved.savedText, "");
    await closeClean(item);
  } else {
    await waitHost(item);
    throw Error("The pending clean draft closed without consent under an active transaction lock");
  }
  return { firstClose: outcome, pendingCloseCanceled: true,
    storageDelay: "actual readwrite transaction held by read-only requests", automaticReleaseMs: 15000 };
}

try {
  const first = await launch();
  const editor = first.page.getByRole("textbox", { name: "Markdown document" });
  await editor.click();
  await editor.fill(expected);
  await first.page.waitForFunction(() => document.querySelector("#draft-state").textContent === "Draft saved");
  const saved = await first.page.evaluate(() => window.FileNotesStorage.load());
  assert.equal(saved.text, expected);
  assert.equal(saved.savedText, "");
  await closeWithConsent(first, false);
  assert.equal(await editor.inputValue(), expected);
  assert.equal(first.child.exitCode, null, "Canceled close exited the host");
  result.cancelRetainsText = true;
  await closeWithConsent(first, true);
  await waitHost(first);
  // Relaunch before waiting for the old browser. This is the real same-profile
  // flow, not recovery made easier by waiting for all native locks to disappear.
  const second = await launch();
  assert.equal(await second.page.locator("#editor").inputValue(), expected);
  assert.equal(await second.page.locator("#save-state").textContent(), "Unsaved changes");
  assert.ok((await second.page.locator("#status").textContent()).startsWith("Draft restored from"));
  const recovered = await second.page.evaluate(() => window.FileNotesStorage.load());
  assert.equal(recovered.text, expected);
  assert.equal(recovered.savedText, "");
  result.immediateRelaunchRecoversCommittedDraft = true;
  if (checkCleanTransition) {
    result.cleanTransition = await closeNewDocument(second);
    const third = await launch();
    assert.equal(await third.page.locator("#editor").inputValue(), "");
    assert.equal(await third.page.locator("#save-state").textContent(), "No save target");
    const clean = await third.page.evaluate(() => window.FileNotesStorage.load());
    assert.equal(clean.text, "");
    assert.equal(clean.savedText, "");
    result.cleanTransition.relaunchHasNoDiscardedDraft = true;
    await third.page.locator("#editor").click();
    await closeClean(third);
    result.cleanTransition.restoredCleanCloseUnprompted = true;
  } else {
    await second.page.locator("#editor").click();
    await closeWithConsent(second, true);
    await waitHost(second);
  }
  for (const item of launches) {
    await item.observation.done;
    assert.equal(item.observation.exited, true, item.observation.error);
  }
  assert.equal(digest(await readFile(executable)), artifact.hostSha256);
  const finalInspection = spawnSync(cli, ["inspect", directory, "--json"], {
    cwd: root, windowsHide: true, encoding: "utf8", timeout: 20000,
  });
  assert.equal(finalInspection.status, 0, finalInspection.stderr);
  assert.deepEqual(JSON.parse(finalInspection.stdout).result, artifact);
  result.passed = true;
} catch (error) {
  result.error = String(error);
  process.exitCode = 1;
} finally {
  for (const item of launches) {
    if (item.child.pid && item.child.exitCode === null && item.child.signalCode === null) {
      spawnSync("taskkill.exe", ["/PID", String(item.child.pid), "/T", "/F"], {
        windowsHide: true, encoding: "utf8", timeout: 15000,
      });
    }
    await item.done;
    if (item.observation) await item.observation.done;
    result.launches.push({ hostPID: item.child.pid, hostExitCode: item.child.exitCode,
      hostSignal: item.child.signalCode, browserPID: item.browserPID,
      browserExitObserved: item.observation?.exited ?? false });
    await writeFile(join(work, "host-" + result.launches.length + ".log"), item.stdout + item.stderr);
  }
  for (const browser of browsers) await browser.close().catch(() => {});
  result.finishedAtUtc = new Date().toISOString();
  await writeFile(join(work, "result.json"), JSON.stringify(result, null, 2) + "\n");
  console.log(JSON.stringify({ receipt: join(work, "result.json"), ...result }));
}
