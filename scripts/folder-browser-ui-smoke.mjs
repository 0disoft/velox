import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { createRequire } from "node:module";
import { createServer } from "node:http";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve, join, extname, sep } from "node:path";
import { fileURLToPath } from "node:url";

const exampleOnly = process.argv[2] === "--example-long-names";
if (process.argv.length > 3 || process.argv.length === 3 && !exampleOnly) {
  throw Error("Usage: node scripts/folder-browser-ui-smoke.mjs [--example-long-names]");
}
const { chromium } = createRequire(import.meta.url)("playwright");
const root = fileURLToPath(new URL("..", import.meta.url));
const output = join(root, ".cache", "folder-browser-readiness-" + Date.now());
const project = exampleOnly ? join(root, "examples", "folder-browser") : join(output, "folder-browser");
await mkdir(output, { recursive: true });
if (!exampleOnly) {
  const generated = JSON.parse(execFileSync("go", ["run", "./cmd/velox", "init", project, "--template", "folder-browser", "--json"],
    { cwd: root, encoding: "utf8", windowsHide: true }));
  assert.equal(generated.ok, true);
}
const web = join(project, "web");
const types = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" };
const server = createServer(async (request, response) => {
  try {
    const pathname = decodeURIComponent(new URL(request.url, "http://localhost").pathname);
    const path = resolve(web, "." + (pathname === "/" ? "/index.html" : pathname));
    if (!path.startsWith(web + sep)) { response.writeHead(403).end(); return; }
    const bytes = await readFile(path);
    response.writeHead(200, { "Content-Type": types[extname(path)] ?? "application/octet-stream" });
    response.end(bytes);
  } catch { response.writeHead(404).end(); }
});
await new Promise(done => server.listen(0, "127.0.0.1", done));
const url = "http://127.0.0.1:" + server.address().port;
const result = { nativeBridge: "mocked", generatedProject: exampleOnly ? null : project,
  project, exampleOnly, views: [], passed: false };
let browser;
try {
  browser = await chromium.launch({ channel: "msedge", headless: true });
  for (const view of [
    { width: 1080, height: 760, colorScheme: "light" },
    { width: 360, height: 640, colorScheme: "light" },
    { width: 360, height: 640, colorScheme: "dark" },
  ]) {
    const context = await browser.newContext({ viewport: { width: view.width, height: view.height }, colorScheme: view.colorScheme });
    try {
      await context.addInitScript(() => {
        window.__calls = [];
        window.__mode = {};
        let target = 0;
        const listing = { entries: [
          { name: "notes-\uD55C\uAE00.txt", kind: "file" },
          { name: "\uD55C".repeat(251) + ".txt", kind: "file" },
          { name: "child", kind: "directory" },
        ], skipped: 1, truncated: false };
        const failure = code => Object.assign(new Error("The folder operation failed."), { code });
        window.velox = { invoke: async (method, params) => {
          window.__calls.push({ method, params });
          if (method === "folder.select") {
            if (window.__mode.selectNull) throw null;
            if (window.__mode.cancel) return { cancelled: true };
            return { name: "\uD55C".repeat(255), target: ++target };
          }
          if (method === "folder.list") {
            if (window.__mode.listFailure) throw failure("NATIVE_OPERATION_FAILED");
            if (window.__mode.pendingList) return new Promise(done => { window.__finishList = () => done(listing); });
            return listing;
          }
          if (method === "folder.openText") {
            if (window.__mode.openFailure) throw failure(window.__mode.openFailure);
            return { name: params.name, text: "\uD55C\uAE00 text\n<script>literal</script>", bytes: 40 };
          }
          if (method === "folder.release") {
            if (window.__mode.releaseFailure) throw failure("NATIVE_OPERATION_FAILED");
            return null;
          }
          throw new Error("unexpected method");
        } };
      });
      const errors = [], failures = [];
      const page = await context.newPage();
      page.on("pageerror", error => errors.push(error.message));
      page.on("response", response => { if (response.status() >= 400) failures.push(response.url()); });
      await page.goto(url);
      const select = page.getByRole("button", { name: "Select folder", exact: true });
      const refresh = page.getByRole("button", { name: "Refresh list", exact: true });
      const release = page.getByRole("button", { name: "Release folder", exact: true });
      const file = () => page.getByRole("button", { name: "notes-\uD55C\uAE00.txt", exact: true });
      const preview = page.getByRole("textbox", { name: "File contents", exact: true });
      const loaded = () => page.getByRole("status").filter({ hasText: "List loaded." }).waitFor();
      assert.deepEqual(await page.evaluate(() => window.__calls), []);
      assert.equal(await preview.getAttribute("readonly"), "");
      await select.focus();
      await select.press("Enter");
      await loaded();
      await file().click();
      await page.getByRole("status").filter({ hasText: "Text loaded." }).waitFor();
      assert.equal(await preview.inputValue(), "\uD55C\uAE00 text\n<script>literal</script>");
      const longName = "\uD55C".repeat(251) + ".txt";
      await page.getByRole("button", { name: longName, exact: true }).click();
      await page.getByRole("status").filter({ hasText: "Text loaded." }).waitFor();
      const longLayout = await page.evaluate(() => ({
        previewHeight: document.querySelector("#preview").clientHeight,
        fileName: document.querySelector("#file-name").textContent,
        fileTitle: document.querySelector("#file-name").title,
        folderTitle: document.querySelector("#folder-name").title,
        folderHeight: document.querySelector("#folder-name").clientHeight,
        overflow: document.documentElement.scrollWidth > innerWidth,
        request: window.__calls.at(-1),
      }));
      const longScreenshot = join(output, `long-name-${view.width}-${view.colorScheme}.png`);
      await page.screenshot({ path: longScreenshot });
      result.views.push({ view, longLayout, longScreenshot });
      assert.equal(longLayout.fileName, longName);
      assert.equal(longLayout.fileTitle, longName);
      assert.equal(longLayout.folderTitle, "\uD55C".repeat(255));
      assert.ok(longLayout.folderHeight <= 52, "Folder heading exceeded two lines");
      assert.equal(longLayout.request.method, "folder.openText");
      assert.equal(longLayout.request.params.name, longName);
      assert.equal(longLayout.overflow, false);
      assert.ok(longLayout.previewHeight > 100, `Long-name preview collapsed: ${JSON.stringify(longLayout)}`);
      if (exampleOnly) {
        assert.deepEqual(errors, []);
        assert.deepEqual(failures, []);
        assert.equal(await page.locator("#copy").isEnabled(), true);
        continue;
      }
      await page.evaluate(() => { window.__mode.cancel = true; });
      await select.click();
      await page.getByRole("status").filter({ hasText: "Selection canceled." }).waitFor();
      assert.equal(await preview.inputValue(), "\uD55C\uAE00 text\n<script>literal</script>");
      await page.evaluate(() => {
        window.__mode.cancel = false;
        window.__mode.listFailure = true;
        window.__oldRow = document.querySelector(".file-name");
      });
      await refresh.click();
      await page.getByRole("status").filter({ hasText: "NATIVE_OPERATION_FAILED" }).waitFor();
      assert.equal(await preview.inputValue(), "");
      assert.equal(await page.locator("#entries tr").count(), 0);
      assert.equal(await page.locator("#count").textContent(), "");
      assert.equal(await refresh.isEnabled(), true);
      const before = await page.evaluate(() => window.__calls.length);
      await page.evaluate(() => window.__oldRow.dispatchEvent(new MouseEvent("click")));
      assert.equal(await page.evaluate(() => window.__calls.length), before);
      const failedScreenshot = join(output, `failed-${view.width}-${view.colorScheme}.png`);
      await page.screenshot({ path: failedScreenshot });
      await page.evaluate(() => { window.__mode.listFailure = false; });
      await refresh.click();
      await loaded();
      await page.evaluate(() => { window.__mode.pendingList = true; });
      await refresh.click();
      await page.getByRole("status").filter({ hasText: "Loading list..." }).waitFor();
      assert.equal(await select.isDisabled(), true);
      await preview.focus();
      await page.evaluate(() => { window.__mode.pendingList = false; window.__finishList(); });
      await loaded();
      assert.equal(await preview.evaluate(node => node === document.activeElement), true);
      await page.evaluate(() => { window.__mode.openFailure = "FOLDER_TARGET_INVALID"; });
      await file().click();
      await page.getByRole("status").filter({ hasText: "FOLDER_TARGET_INVALID" }).waitFor();
      assert.equal(await page.locator("#folder-name").textContent(), "No folder selected");
      assert.equal(await refresh.isDisabled(), true);
      assert.equal(await select.evaluate(node => node === document.activeElement), true);
      await page.evaluate(() => { delete window.__mode.openFailure; });
      await select.press("Enter");
      await loaded();
      await page.evaluate(() => { window.__mode.openFailure = "UNSUPPORTED_FILE"; });
      await file().click();
      await page.getByRole("status").filter({ hasText: "UNSUPPORTED_FILE" }).waitFor();
      assert.equal(await preview.inputValue(), "");
      assert.equal(await refresh.isEnabled(), true);
      await page.evaluate(() => { delete window.__mode.openFailure; });
      await file().click();
      await page.getByRole("status").filter({ hasText: "Text loaded." }).waitFor();
      const loadedScreenshot = join(output, `loaded-${view.width}-${view.colorScheme}.png`);
      await page.screenshot({ path: loadedScreenshot });
      const layout = await page.evaluate(() => ({
        overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
        preview: { width: document.querySelector("#preview").clientWidth, height: document.querySelector("#preview").clientHeight },
        mask: getComputedStyle(document.querySelector("#select"), "::before").maskImage,
      }));
      assert.equal(layout.overflow, false);
      assert.ok(layout.preview.width >= 300 && layout.preview.height > 100);
      assert.ok(layout.mask.includes("folder-open.svg"));
      await page.evaluate(() => { window.__mode.releaseFailure = true; });
      await release.click();
      await page.getByRole("status").filter({ hasText: "NATIVE_OPERATION_FAILED" }).waitFor();
      assert.equal(await release.isEnabled(), true);
      await page.evaluate(() => { window.__mode.releaseFailure = false; });
      await release.click();
      await page.getByRole("status").filter({ hasText: "Folder released." }).waitFor();
      assert.equal(await select.evaluate(node => node === document.activeElement), true);
      assert.equal(await preview.inputValue(), "");
      await page.evaluate(() => { window.__mode.selectNull = true; });
      await select.press("Enter");
      await page.getByRole("status").filter({ hasText: "ERROR: The folder operation failed." }).waitFor();
      assert.equal(await select.isEnabled(), true);
      await page.reload();
      assert.deepEqual(await page.evaluate(() => window.__calls), []);
      assert.equal(await page.locator("#folder-name").textContent(), "No folder selected");
      assert.deepEqual(errors, []);
      assert.deepEqual(failures, []);
      Object.assign(result.views.at(-1), { layout, failedScreenshot, loadedScreenshot, errors, failures,
        explicitRetry: true, focusRecovery: true, noRestoredFolderGrant: true });
    } finally { await context.close(); }
  }
  result.passed = true;
} catch (error) {
  result.error = String(error);
  throw error;
} finally {
  await browser?.close();
  await new Promise(done => server.close(done));
  await writeFile(join(output, "result.json"), JSON.stringify(result, null, 2) + "\n");
  console.log(JSON.stringify({ output, ...result }));
}
