import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { createRequire } from "node:module";
import { createServer } from "node:http";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve, join, extname, sep } from "node:path";
import { fileURLToPath } from "node:url";

const { chromium } = createRequire(import.meta.url)("playwright");
const root = fileURLToPath(new URL("..", import.meta.url));
const output = join(root, ".cache", "editor-storage-failure-" + Date.now());
const project = join(output, "editor");
await mkdir(output, { recursive: true });
const generated = JSON.parse(execFileSync("go", ["run", "./cmd/velox", "init", project, "--template", "text-editor", "--json"],
  { cwd: root, encoding: "utf8", windowsHide: true }));
assert.equal(generated.ok, true);
const apps = { editor: join(project, "web"), notes: join(root, "examples/file-notes/web") };
const types = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".ttf": "font/ttf" };
const server = createServer(async (request, response) => {
  try {
    const pathname = decodeURIComponent(new URL(request.url, "http://localhost").pathname);
    const [, app, ...segments] = pathname.split("/");
    const web = apps[app];
    if (!web) { response.writeHead(404).end(); return; }
    const path = resolve(web, segments.join("/") || "index.html");
    if (!path.startsWith(web + sep)) { response.writeHead(403).end(); return; }
    const bytes = await readFile(path);
    response.writeHead(200, { "Content-Type": types[extname(path)] ?? "application/octet-stream" });
    response.end(bytes);
  } catch { response.writeHead(404).end(); }
});
await new Promise(done => server.listen(0, "127.0.0.1", done));
const url = "http://127.0.0.1:" + server.address().port;
const result = { nativeBridge: "mocked", storageFailure: "injected UnknownError", generatedProject: project, views: [], passed: false };
async function rawEditorDraft(page, value, write = false) {
  return page.evaluate(({ value, write }) => new Promise((resolve, reject) => {
    const request = indexedDB.open("velox.text-editor", 1);
    request.onerror = () => reject(Error("Fixture database could not open"));
    request.onsuccess = () => {
      const database = request.result;
      const transaction = database.transaction("drafts", write ? "readwrite" : "readonly");
      const store = transaction.objectStore("drafts");
      const operation = write ? store.put(value, "current") : store.get("current");
      let result;
      operation.onsuccess = () => { result = operation.result; };
      transaction.oncomplete = () => { database.close(); resolve(result); };
      transaction.onerror = transaction.onabort = () => { database.close(); reject(Error("Fixture transaction failed")); };
    };
  }), { value, write });
}
let browser;
try {
  browser = await chromium.launch({ channel: "msedge", headless: true });
  for (const app of Object.keys(apps)) {
    for (const phase of ["startup", "write"]) {
      for (const view of [
        { width: 1080, height: 760, colorScheme: "light" },
        { width: 360, height: 640, colorScheme: "light" },
        { width: 360, height: 640, colorScheme: "dark" },
      ]) {
        const context = await browser.newContext({ viewport: { width: view.width, height: view.height }, colorScheme: view.colorScheme });
        try {
          await context.addInitScript(({ phase }) => {
            window.__writes = [];
            window.__cancelSave = true;
            window.__draftUnavailable = phase === "startup";
            const actualDatabase = window.indexedDB;
            Object.defineProperty(window, "indexedDB", { value: {
              open(...args) {
                if (window.__draftUnavailable) {
                  throw new DOMException("Internal error opening backing store for indexedDB.open.", "UnknownError");
                }
                return actualDatabase.open(...args);
              },
            } });
            window.velox = {
              invoke: async () => null,
              saveTextAs: async (text, name) => {
                if (window.__cancelSave) return { cancelled: true };
                window.__writes.push({ text, name });
                return { name: "kept.txt", target: 1 };
              },
              saveTextTo: async () => { throw Error("Unexpected connected save"); },
            };
          }, { phase });
          const errors = [], failures = [];
          const page = await context.newPage();
          page.on("pageerror", error => errors.push(error.message));
          page.on("response", response => { if (response.status() >= 400) failures.push(response.url()); });
          await page.goto(`${url}/${app}/`);
          await page.waitForFunction(() => !document.querySelector("#editor").readOnly);
          if (phase === "write") await page.evaluate(() => {
            window.__draftUnavailable = true;
          });
          const editor = page.locator("#editor");
          const text = "\uD55C\uAE00 draft\nKeep this text.";
          await editor.fill(text);
          await page.waitForFunction(() => document.querySelector("#draft-state").textContent.includes("Save changes to a file"));
          await editor.fill(text + " retry");
          assert.ok((await page.locator("#draft-state").textContent()).includes("Save changes to a file"));
          await page.evaluate(text => {
            const editor = document.querySelector("#editor");
            editor.dispatchEvent(new CompositionEvent("compositionstart"));
            editor.value = text + " unfinished";
            editor.dispatchEvent(new Event("input", { bubbles: true }));
          }, text);
          assert.ok((await page.locator("#draft-state").textContent()).includes("Save changes to a file"));
          await page.evaluate(text => {
            const editor = document.querySelector("#editor");
            editor.value = text;
            editor.dispatchEvent(new Event("input", { bubbles: true }));
            editor.dispatchEvent(new CompositionEvent("compositionend"));
          }, text);
          assert.ok((await page.locator("#draft-state").textContent()).includes("Save changes to a file"));
          await page.getByRole("button", { name: "Save as", exact: true }).click();
          await page.waitForFunction(() => document.querySelector("#status").textContent.toLowerCase().includes("cancel"));
          assert.equal(await editor.inputValue(), text);
          assert.equal(await page.locator("#save-state").textContent(), "Unsaved changes");
          assert.deepEqual(await page.evaluate(() => window.__writes), []);
          const screenshot = join(output, `${app}-${phase}-${view.width}-${view.colorScheme}.png`);
          await page.screenshot({ path: screenshot, fullPage: true });
          const layout = await page.evaluate(() => {
            const editor = document.querySelector("#editor").getBoundingClientRect();
            const notice = document.querySelector("#draft-state").getBoundingClientRect();
            return {
              overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
              editorWidth: editor.width, editorHeight: editor.height,
              noticeBelowEditor: notice.top >= editor.bottom,
              noticeWithinPage: notice.right <= document.documentElement.clientWidth && notice.left >= 0,
            };
          });
          assert.equal(layout.overflow, false);
          assert.ok(layout.editorWidth > 200 && layout.editorHeight > 150);
          assert.equal(layout.noticeBelowEditor, true);
          assert.equal(layout.noticeWithinPage, true);
          await page.evaluate(() => { window.__cancelSave = false; });
          await page.getByRole("button", { name: "Save as", exact: true }).click();
          await page.waitForFunction(() => document.querySelector("#save-state").textContent === "Saved to file");
          assert.equal((await page.evaluate(() => window.__writes))[0].text, text);
          assert.equal(await editor.inputValue(), text);
          assert.ok((await page.locator("#draft-state").textContent()).includes("Save changes to a file"));
          await page.evaluate(() => { window.__draftUnavailable = false; });
          const recovered = text + "\nRecovery retry";
          await editor.fill(recovered);
          assert.ok((await page.locator("#draft-state").textContent()).includes("Save changes to a file"));
          await page.waitForFunction(() => /^(Draft saved|Draft stored)$/.test(document.querySelector("#draft-state").textContent));
          const draft = await page.evaluate(app => (app === "editor" ? window.EditorDrafts : window.FileNotesStorage).load(), app);
          assert.equal(draft.text, recovered);
          assert.deepEqual(errors, []);
          assert.deepEqual(failures, []);
          result.views.push({ app, phase, view, layout, screenshot, canceledThenSaved: true,
            warningRetainedDuringTypingAndSyntheticIME: true, recoveryConfirmedInRealIndexedDB: true });
        } finally { await context.close(); }
      }
    }
  }
  result.protectedDrafts = [];
  const valid = { schemaVersion: 1, name: "protected.txt", text: "protected original", updatedAt: 123 };
  for (const candidate of [
    { kind: "future-format", value: { ...valid, schemaVersion: 2 } },
    { kind: "invalid-text", value: { ...valid, text: { invalid: true } } },
    { kind: "authority-field", value: { ...valid, target: 42 } },
    { kind: "oversized-text", value: { ...valid, text: "x".repeat(2 * 1024 * 1024 + 1) } },
  ]) {
    const context = await browser.newContext({ viewport: { width: 360, height: 640 } });
    try {
      await context.addInitScript(() => {
        const open = indexedDB.open.bind(indexedDB);
        window.__draftOpens = 0;
        indexedDB.open = (...args) => { window.__draftOpens++; return open(...args); };
        window.__writes = [];
        window.velox = {
          invoke: async method => method === "file.openText" ? { name: "opened.txt", text: "Opened content" } : null,
          saveTextAs: async text => { window.__writes.push(text); return { name: "exported.txt", target: 1 }; },
          saveTextTo: async () => { throw Error("Unexpected connected save"); },
        };
      });
      const page = await context.newPage();
      const errors = [];
      page.on("pageerror", error => errors.push(error.message));
      await page.goto(`${url}/editor/`);
      await page.waitForFunction(() => !document.querySelector("#editor").readOnly);
      await rawEditorDraft(page, candidate.value, true);
      await page.reload();
      await page.waitForFunction(() => !document.querySelector("#editor").readOnly &&
        document.querySelector("#draft-state").textContent.includes("kept unchanged"));
      const before = await page.evaluate(() => window.__draftOpens);
      await page.locator("#editor").fill("New exportable text");
      await page.waitForFunction(count => window.__draftOpens > count, before);
      assert.deepEqual(await rawEditorDraft(page), candidate.value);
      await page.getByRole("button", { name: "Save as", exact: true }).click();
      await page.locator("#status").filter({ hasText: "File saved. Draft cleanup unavailable." }).waitFor();
      assert.deepEqual(await page.evaluate(() => window.__writes), ["New exportable text"]);
      assert.equal(await page.locator("#save-state").textContent(), "Saved to file");
      assert.deepEqual(await rawEditorDraft(page), candidate.value);
      await page.getByRole("button", { name: "Open", exact: true }).click();
      await page.locator("#status").filter({ hasText: "File opened. Draft cleanup unavailable." }).waitFor();
      assert.equal(await page.locator("#editor").inputValue(), "Opened content");
      assert.deepEqual(await rawEditorDraft(page), candidate.value);
      await page.getByRole("button", { name: "New", exact: true }).click();
      await page.locator("#status").filter({ hasText: "New document. Draft cleanup unavailable." }).waitFor();
      assert.equal(await page.locator("#editor").inputValue(), "");
      assert.deepEqual(await rawEditorDraft(page), candidate.value);
      const layout = await page.evaluate(() => {
        const notice = document.querySelector("#draft-state").getBoundingClientRect();
        return { overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
          noticeWithinPage: notice.left >= 0 && notice.right <= document.documentElement.clientWidth };
      });
      assert.equal(layout.overflow, false);
      assert.equal(layout.noticeWithinPage, true);
      assert.ok((await page.locator("#draft-state").textContent()).includes("kept unchanged"));
      assert.deepEqual(errors, []);
      result.protectedDrafts.push({ kind: candidate.kind, layout, autoSavePreserved: true,
        saveOpenAndNewCleanupPreserved: true, nativeResponses: "mocked", pageErrors: errors });
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
