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
            window.velox = {
              invoke: async () => null,
              saveTextAs: async (text, name) => {
                if (window.__cancelSave) return { cancelled: true };
                window.__writes.push({ text, name });
                return { name: "kept.txt", target: 1 };
              },
              saveTextTo: async () => { throw Error("Unexpected connected save"); },
            };
            if (phase === "startup") Object.defineProperty(window, "indexedDB", { value: {
              open() { throw new DOMException("Internal error opening backing store for indexedDB.open.", "UnknownError"); },
            } });
          }, { phase });
          const errors = [], failures = [];
          const page = await context.newPage();
          page.on("pageerror", error => errors.push(error.message));
          page.on("response", response => { if (response.status() >= 400) failures.push(response.url()); });
          await page.goto(`${url}/${app}/`);
          await page.waitForFunction(() => !document.querySelector("#editor").readOnly);
          if (phase === "write") await page.evaluate(() => {
            Object.defineProperty(window, "indexedDB", { value: {
              open() { throw new DOMException("Internal error opening backing store for indexedDB.open.", "UnknownError"); },
            } });
          });
          const editor = page.locator("#editor");
          const text = "\uD55C\uAE00 draft\nKeep this text.";
          await editor.fill(text);
          await page.waitForFunction(() => document.querySelector("#draft-state").textContent.includes("Save changes to a file"));
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
          assert.deepEqual(errors, []);
          assert.deepEqual(failures, []);
          result.views.push({ app, phase, view, layout, screenshot, canceledThenSaved: true });
        } finally { await context.close(); }
      }
    }
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
