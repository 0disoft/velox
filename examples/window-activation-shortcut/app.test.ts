import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const app = await readFile(new URL("./web/app.js", import.meta.url), "utf8");

function harness() {
  const nodes = new Map<string, any>();
  const events = new Map<string, Function>();
  let ready = false;
  function node(selector: string) {
    if (!nodes.has(selector)) nodes.set(selector, {
      value: selector === "#note" ? "Activation shortcut test." : "", textContent: "",
      addEventListener(name: string, callback: Function) { events.set(`${selector}:${name}`, callback); },
    });
    return nodes.get(selector);
  }
  const window = {
    addEventListener(name: string, callback: Function) { events.set(`window:${name}`, callback); },
    __veloxReady(phase: string) { expect(phase).toBe("dom-2raf"); ready = true; },
    get velox() { throw Error("Activation example must not invoke IPC"); },
    get localStorage() { throw Error("Activation example must not persist notes"); },
  };
  runInNewContext(app, { window, document: { querySelector: node }, requestAnimationFrame(callback: Function) { callback(); } });
  return { node, get ready() { return ready; },
    input(value: string) { node("#note").value = value; events.get("#note:input")!(); },
    focus() { events.get("window:focus")!(); } };
}

test("input stays local, preserves Unicode and counts UTF-16 units", () => {
  const ui = harness();
  expect(ui.ready).toBe(true);
  expect(ui.node("#note-count").value).toBe("25 / 2048");
  const note = "\uD55C\uAE00 \uD83D\uDE80\n<script> plain note";
  ui.input(note);
  expect(ui.node("#note").value).toBe(note);
  expect(ui.node("#note-count").value).toBe(`${note.length} / 2048`);
  expect(ui.node("#status").textContent).toBe("Note changed.");
  ui.input("x".repeat(2048));
  expect(ui.node("#note-count").value).toBe("2048 / 2048");
});

test("repeated window focus does not reset the note or count", () => {
  const ui = harness();
  ui.input("Disposable note");
  ui.focus();
  ui.focus();
  expect(ui.node("#status").textContent).toBe("Window focused.");
  expect(ui.node("#note").value).toBe("Disposable note");
  expect(ui.node("#note-count").value).toBe("15 / 2048");
});

test("manifest opts into only a native activation shortcut and host tray", async () => {
  const manifest = JSON.parse(await readFile(new URL("./velox.json", import.meta.url), "utf8"));
  expect(manifest.app.id).toBe("dev.velox.activation");
  expect(manifest.window.activationShortcut).toBe("Ctrl+Alt+Shift+V");
  expect(manifest.window.tray).toBe(true);
  expect(manifest.security.permissions).toEqual([]);
});
