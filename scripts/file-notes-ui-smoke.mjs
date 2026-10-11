import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { createServer } from "node:http";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve, join, extname, sep } from "node:path";
import { fileURLToPath } from "node:url";

const { chromium } = createRequire(import.meta.url)("playwright");
const root = fileURLToPath(new URL("..", import.meta.url));
const web = join(root, "examples/file-notes/web");
const output = join(root, ".cache", "file-notes-readiness-" + Date.now());
await mkdir(output, { recursive: true });
const types = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".woff2": "font/woff2", ".ttf": "font/ttf", ".png": "image/png" };
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
const result = { nativeDialogs: "mocked", draftStorage: "real IndexedDB", views: [], passed: false };
let browser;
try {
  browser = await chromium.launch({ channel: "msedge", headless: true });
  for (const viewport of [{ width: 1080, height: 760 }, { width: 360, height: 640 }]) {
    const context = await browser.newContext({ viewport });
    try {
      await context.addInitScript(() => {
        window.__writes = [];
        window.velox = {
          invoke: async method => {
            if (method === "file.openText") return { name: "lines.md", text: "one\r\ntwo\r\n" };
            return null;
          },
          saveTextAs: async (text, name) => { window.__writes.push({ text, name }); return { name: "saved.md", target: 1 }; },
          saveTextTo: async (text, target) => { window.__writes.push({ text, target }); return { name: "saved.md", target }; },
        };
      });
      const errors = [];
      context.on("page", page => page.on("pageerror", error => errors.push(error.message)));
      const page = await context.newPage();
      await page.goto(url);
      const editor = page.getByRole("textbox", { name: "Markdown document" });
      await page.waitForFunction(() => !document.querySelector("#editor").readOnly);
      await page.getByRole("button", { name: "Open", exact: true }).click();
      await page.getByRole("status").filter({ hasText: "lines.md opened." }).waitFor();
      assert.equal(await editor.inputValue(), "one\ntwo\n");
      assert.equal(await page.locator("#save-state").textContent(), "No save target");
      await editor.fill("one\ntwo\n\uD55C\uAE00\n");
      await page.waitForFunction(() => document.querySelector("#draft-state").textContent === "Draft saved");
      const draft = await page.evaluate(() => window.FileNotesStorage.load());
      assert.equal(draft.text, "one\r\ntwo\r\n\uD55C\uAE00\r\n");
      assert.equal(draft.savedText, "one\r\ntwo\r\n");
      await editor.press("Control+f");
      const find = page.getByRole("searchbox", { name: "Find text" });
      await find.fill("two");
      assert.equal(await page.locator("#find-count").textContent(), "1 / 1");
      await find.press("Escape");
      assert.equal(await page.locator("#find-bar").isVisible(), false);
      await page.evaluate(() => document.fonts.ready);
      assert.equal(await page.evaluate(() => document.fonts.check('16px "Velox Noto Sans KR"', "\uD55C\uAE00")), true);
      await editor.press("Control+f");
      await find.fill("");
      await page.getByRole("button", { name: "Close find", exact: true }).click();
      assert.equal(await page.locator("#find-bar").isVisible(), false);
      const screenshot = join(output, "editor-" + viewport.width + ".png");
      await page.screenshot({ path: screenshot, fullPage: true });
      const layout = await page.evaluate(() => ({
        overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
        width: document.querySelector("#editor").getBoundingClientRect().width,
        height: document.querySelector("#editor").getBoundingClientRect().height,
        font: getComputedStyle(document.querySelector("#editor")).fontFamily,
      }));
      assert.equal(layout.overflow, false);
      assert.ok(layout.width > 200 && layout.height > 150);
      // A new page in the same browser context checks committed recovery without
      // consenting to close the original unsaved document.
      const reopened = await context.newPage();
      await reopened.goto(url);
      await reopened.waitForFunction(() => !document.querySelector("#editor").readOnly);
      assert.equal(await reopened.locator("#editor").inputValue(), "one\ntwo\n\uD55C\uAE00\n");
      assert.equal(await reopened.locator("#save-state").textContent(), "Unsaved changes");
      await reopened.getByRole("button", { name: "Save", exact: true }).click();
      await reopened.getByRole("status").filter({ hasText: "saved.md saved." }).waitFor();
      const writes = await reopened.evaluate(() => window.__writes);
      assert.deepEqual(writes, [{ text: draft.text, name: "lines.md" }]);
      assert.equal(await reopened.locator("#save-state").textContent(), "Saved to file");
      assert.deepEqual(errors, []);
      result.views.push({ viewport, layout, screenshot, restoredAndSaved: true });
    } finally { await context.close(); }
  }
  const context = await browser.newContext();
  try {
    await context.addInitScript(() => {
      Object.defineProperty(window, "indexedDB", { value: { open() {
        const request = {};
        queueMicrotask(() => request.onblocked());
        return request;
      } } });
    });
    const page = await context.newPage();
    await page.goto(url);
    await page.waitForFunction(() => !document.querySelector("#editor").readOnly);
    assert.equal(await page.locator("#draft-state").textContent(), "Draft recovery unavailable");
    await page.getByRole("textbox", { name: "Markdown document" }).fill("Still editable");
    assert.equal(await page.locator("#editor").inputValue(), "Still editable");
    result.blockedRecoveryUnlocksEditor = true;
  } finally { await context.close(); }
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
