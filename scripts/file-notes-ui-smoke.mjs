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
async function waitForDraft(page, text, savedText) {
  const deadline = Date.now() + 5000;
  let draft;
  while (Date.now() < deadline) {
    draft = await page.evaluate(() => window.FileNotesStorage.load());
    if (draft?.text === text && (savedText === undefined || draft.savedText === savedText)) return draft;
    await new Promise(resolve => setTimeout(resolve, 30));
  }
  throw new Error("The expected recovery snapshot did not commit within five seconds.");
}
async function rawDraft(page, value, write = false) {
  return page.evaluate(({ value, write }) => new Promise((resolve, reject) => {
    const request = indexedDB.open("dev.velox.filenotes", 1);
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
      const draft = await waitForDraft(page, "one\r\ntwo\r\n\uD55C\uAE00\r\n", "one\r\ntwo\r\n");
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
  const recoveryContext = await browser.newContext();
  try {
    await recoveryContext.addInitScript(() => {
      let storage;
      // Delay the storage boundary, but retain the actual IndexedDB operations.
      Object.defineProperty(window, "FileNotesStorage", {
        get: () => storage,
        set: value => {
          storage = Object.freeze({ ...value, save: async state => {
            if (window.__holdDraft) {
              await new Promise(resolve => { window.__resumeDraft = resolve; });
            }
            return value.save(state);
          } });
        },
      });
      window.velox = {
        invoke: async method => method === "file.openText"
          ? { name: "opened.md", text: "opened\r\ntext" } : null,
        saveTextAs: async () => ({ name: "saved.md", target: 1 }),
      };
    });
    const page = await recoveryContext.newPage();
    const cdp = await recoveryContext.newCDPSession(page);
    const errors = [];
    page.on("pageerror", error => errors.push(error.message));
    await page.goto(url);
    await page.waitForFunction(() => !document.querySelector("#editor").readOnly);
    const checks = [];
    for (const action of ["Save", "New", "Open"]) {
      const dirty = "old draft before " + action;
      await page.locator("#editor").fill(dirty);
      await waitForDraft(page, dirty);
      await page.evaluate(() => { window.__holdDraft = true; });
      await page.getByRole("button", { name: action, exact: true }).click();
      if (action !== "Save") await page.getByRole("button", { name: "Discard changes", exact: true }).click();
      await page.waitForFunction(() => typeof window.__resumeDraft === "function");
      const expected = action === "Save" ? dirty : action === "New" ? "" : "opened\r\ntext";
      assert.equal(await page.locator("#editor").inputValue(), expected.replace(/\r\n/g, "\n"));
      assert.notEqual(await page.locator("#save-state").textContent(), "Unsaved changes");
      const oldDraft = await page.evaluate(() => window.FileNotesStorage.load());
      assert.equal(oldDraft.text, dirty);
      assert.notEqual(oldDraft.savedText, dirty);
      assert.equal(await page.getByRole("button", { name: "Save", exact: true }).isEnabled(), true);

      let dialogs = 0;
      await page.evaluate(() => { window.__reloadMarker = "retained"; });
      let deadline;
      const canceled = new Promise((resolve, reject) => {
        deadline = setTimeout(() => reject(new Error("Pending recovery did not request beforeunload consent.")), 5000);
        page.once("dialog", async dialog => {
          try {
            dialogs++;
            assert.equal(dialog.type(), "beforeunload");
            await dialog.dismiss();
            resolve();
          } catch (error) { reject(error); }
        });
      });
      try {
        await cdp.send("Page.reload");
        await canceled;
      } finally { clearTimeout(deadline); }
      assert.equal(dialogs, 1);
      assert.equal(await page.evaluate(() => window.__reloadMarker), "retained");
      assert.equal(await page.locator("#editor").inputValue(), expected.replace(/\r\n/g, "\n"));

      await page.evaluate(() => {
        window.__holdDraft = false;
        window.__resumeDraft();
        delete window.__resumeDraft;
      });
      await waitForDraft(page, expected, expected);
      await page.waitForFunction(() => {
        const event = new Event("beforeunload", { cancelable: true });
        window.dispatchEvent(event);
        return !event.defaultPrevented;
      });
      const unexpected = async dialog => { dialogs++; await dialog.dismiss(); };
      page.on("dialog", unexpected);
      await page.reload();
      page.off("dialog", unexpected);
      await page.waitForFunction(() => !document.querySelector("#editor").readOnly);
      assert.equal(dialogs, 1);
      assert.equal(await page.locator("#editor").inputValue(), expected.replace(/\r\n/g, "\n"));
      assert.equal(await page.locator("#save-state").textContent(), "No save target");
      checks.push({ action, pendingRecoveryWarned: true, canceledReloadPreservedText: true,
        committedReloadUnprompted: true, cleanBaselineRestored: true });
    }
    assert.deepEqual(errors, []);
    result.cleanRecoveryTransitions = { delay: "controlled storage boundary", checks, pageErrors: errors };
  } finally { await recoveryContext.close(); }
  const protectedDrafts = [
    { kind: "future-format", value: { schemaVersion: 2, text: "Future draft", extra: { preserved: true } } },
    { kind: "invalid-text", value: { schemaVersion: 1, text: { invalid: true }, savedText: "old text" } },
    { kind: "oversized-text", value: { schemaVersion: 1, text: "x".repeat(2 * 1024 * 1024 + 1) } },
  ];
  result.protectedDrafts = [];
  for (const candidate of protectedDrafts) {
    const viewport = { width: 360, height: 640 };
    const context = await browser.newContext({ viewport });
    try {
      await context.addInitScript(() => {
        const open = indexedDB.open.bind(indexedDB);
        window.__draftOpens = 0;
        indexedDB.open = (...args) => { window.__draftOpens++; return open(...args); };
        window.__writes = [];
        window.velox = { invoke: async () => null, saveTextAs: async text => {
          window.__writes.push(text);
          return { name: "exported.md", target: 1 };
        } };
      });
      const page = await context.newPage();
      const errors = [];
      page.on("pageerror", error => errors.push(error.message));
      await page.goto(url);
      await page.waitForFunction(() => !document.querySelector("#editor").readOnly);
      await rawDraft(page, candidate.value, true);
      await page.reload();
      await page.waitForFunction(() => !document.querySelector("#editor").readOnly &&
        document.querySelector("#draft-state").textContent.includes("kept unchanged"));
      const before = await page.evaluate(() => window.__draftOpens);
      await page.locator("#editor").fill("New editable text");
      await page.waitForFunction(count => window.__draftOpens > count, before);
      assert.deepEqual(await rawDraft(page), candidate.value);
      assert.ok((await page.locator("#draft-state").textContent()).includes("kept unchanged"));
      const layout = await page.evaluate(() => {
        const notice = document.querySelector("#draft-state").getBoundingClientRect();
        return { overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
          noticeWithinPage: notice.left >= 0 && notice.right <= document.documentElement.clientWidth };
      });
      assert.equal(layout.overflow, false);
      assert.equal(layout.noticeWithinPage, true);
      const screenshot = join(output, "protected-" + candidate.kind + ".png");
      await page.screenshot({ path: screenshot, fullPage: true });
      await page.getByRole("button", { name: "Save as", exact: true }).click();
      await page.locator("#status").filter({ hasText: "exported.md saved." }).waitFor();
      assert.deepEqual(await page.evaluate(() => window.__writes), ["New editable text"]);
      assert.equal(await page.locator("#save-state").textContent(), "Saved to file");
      assert.deepEqual(await rawDraft(page), candidate.value);
      await page.getByRole("button", { name: "New", exact: true }).click();
      await page.waitForFunction(() => document.querySelector("#editor").value === "" &&
        !document.querySelector("#new-document").disabled);
      assert.deepEqual(await rawDraft(page), candidate.value);
      assert.deepEqual(errors, []);
      result.protectedDrafts.push({ kind: candidate.kind, viewport, layout, screenshot, startupNotice: true, editingEnabled: true,
        autoSaveAndNewPreserveOriginal: true, mockedNativeExportWorks: true, pageErrors: errors });
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
    assert.equal(await page.locator("#draft-state").textContent(), "Draft recovery unavailable. Save changes to a file.");
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
