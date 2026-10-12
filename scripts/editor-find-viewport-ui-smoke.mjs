import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { createRequire } from "node:module";
import { createServer } from "node:http";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { join, resolve, extname, sep } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const output = join(root, ".cache", "editor-find-viewport-" + Date.now());
const project = join(output, "editor");
await mkdir(output, { recursive: true });
const generated = JSON.parse(execFileSync("go", ["run", "./cmd/velox", "init", project, "--template", "text-editor", "--json"],
  { cwd: root, encoding: "utf8", windowsHide: true, timeout: 30000 }));
assert.equal(generated.ok, true);
const web = join(project, "web");
const server = createServer(async (request, response) => {
  try {
    const path = resolve(web, "." + decodeURIComponent(new URL(request.url, "http://localhost").pathname));
    if (!path.startsWith(web + sep)) { response.writeHead(403).end(); return; }
    const file = path.endsWith(sep) || path === web ? join(path, "index.html") : path;
    const bytes = await readFile(file);
    response.writeHead(200, { "Content-Type": { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml" }[extname(file)] ?? "application/octet-stream" });
    response.end(bytes);
  } catch { response.writeHead(404).end(); }
});
await new Promise(done => server.listen(0, "127.0.0.1", done));
const url = "http://127.0.0.1:" + server.address().port + "/index.html";
const { chromium } = createRequire(import.meta.url)("playwright");
const result = { nativeBridge: "mocked; no file operations", generatedProject: project, cases: [], passed: false };

async function measuredMatch(page) {
  return page.evaluate(() => {
    const editor = document.querySelector("#editor");
    const probe = document.createElement("div"), style = getComputedStyle(editor);
    for (const name of ["fontFamily", "fontSize", "fontWeight", "fontStyle", "lineHeight", "letterSpacing", "wordSpacing", "tabSize", "wordBreak", "overflowWrap", "textIndent", "paddingTop", "paddingRight", "paddingBottom", "paddingLeft", "direction", "textAlign"]) probe.style[name] = style[name];
    Object.assign(probe.style, { position: "fixed", left: "-20000px", top: "0", visibility: "hidden", boxSizing: "border-box", width: `${editor.clientWidth}px`,
      whiteSpace: editor.wrap === "off" ? "pre" : "pre-wrap", border: "0" });
    probe.textContent = editor.value;
    document.body.appendChild(probe);
    try {
      const range = document.createRange();
      range.setStart(probe.firstChild, editor.selectionStart);
      range.setEnd(probe.firstChild, editor.selectionEnd);
      const box = probe.getBoundingClientRect(), match = range.getClientRects()[0];
      const left = match.left - box.left - editor.scrollLeft;
      const top = match.top - box.top - editor.scrollTop;
      return { wrap: editor.wrap, left, top, right: left + match.width, bottom: top + match.height,
        width: editor.clientWidth, height: editor.clientHeight, scrollLeft: editor.scrollLeft, scrollTop: editor.scrollTop,
        selected: editor.value.slice(editor.selectionStart, editor.selectionEnd),
        visible: left >= -1 && top >= -1 && left + match.width <= editor.clientWidth + 1 && top + match.height <= editor.clientHeight + 1 };
    } finally { probe.remove(); }
  });
}

let browser;
try {
  browser = await chromium.launch({ channel: "msedge", headless: true });
  for (const viewport of [{ width: 1080, height: 760 }, { width: 360, height: 640 }]) {
    for (const sample of [
      { name: "single-long-line", text: "HEAD " + "x".repeat(220) + " NEEDLE", wrap: false },
      { name: "multiline-unwrapped", text: "HEAD\n" + "x".repeat(220) + " NEEDLE\n" + "other line\n".repeat(65), wrap: false },
      { name: "vertical-unwrapped", text: "other line\n".repeat(65) + "NEEDLE", wrap: false },
      { name: "wrapped-long-text", text: "word ".repeat(700) + "NEEDLE", wrap: true },
      { name: "rtl-unwrapped", text: "\u05d0".repeat(220) + " \u05de\u05d7\u05d8", query: "\u05de\u05d7\u05d8", wrap: false, direction: "rtl" },
    ]) {
      const context = await browser.newContext({ viewport });
      try {
        await context.addInitScript(() => {
          window.velox = { invoke: async () => null, saveTextAs: async () => ({ cancelled: true }), saveTextTo: async () => ({ cancelled: true }) };
        });
        const page = await context.newPage(), errors = [];
        page.on("pageerror", error => errors.push(error.message));
        await page.goto(url);
        await page.waitForFunction(() => !document.querySelector("#editor").readOnly);
        if (sample.direction) await page.locator("#editor").evaluate((editor, dir) => { editor.dir = dir; }, sample.direction);
        await page.locator("#editor").fill(sample.text);
        if (!sample.wrap) await page.getByRole("button", { name: "Word wrap", exact: true }).click();
        await page.getByRole("button", { name: "Find in document", exact: true }).click();
        await page.locator("#find-input").fill(sample.query ?? "NEEDLE");
        const measured = await measuredMatch(page);
        result.cases.push({ sample: sample.name, viewport, measured, pageErrors: errors });
        await page.screenshot({ path: join(output, `${sample.name}-${viewport.width}.png`) });
        assert.equal(measured.selected, sample.query ?? "NEEDLE");
        assert.equal(measured.visible, true, `${sample.name} at ${viewport.width}: ${JSON.stringify(measured)}`);
        assert.deepEqual(errors, []);
        assert.equal(await page.locator("#editor").inputValue(), sample.text);
        if (sample.text.startsWith("HEAD")) {
          await page.locator("#find-input").fill("HEAD");
          const leading = await measuredMatch(page);
          assert.equal(leading.selected, "HEAD");
          assert.equal(leading.visible, true, `Return to leading match: ${JSON.stringify(leading)}`);
          result.cases.at(-1).leadingMatch = leading;
        }
      } finally { await context.close(); }
    }
  }
  result.passed = true;
} catch (error) { result.error = String(error); process.exitCode = 1; }
finally {
  await browser?.close();
  await new Promise(done => server.close(done));
  await writeFile(join(output, "result.json"), JSON.stringify(result, null, 2) + "\n");
  console.log(JSON.stringify({ receipt: join(output, "result.json"), ...result }));
}
