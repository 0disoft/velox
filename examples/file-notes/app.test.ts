import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const model = await readFile(new URL("./web/model.js", import.meta.url), "utf8");
const app = await readFile(new URL("./web/app.js", import.meta.url), "utf8");
const tick = () => new Promise((resolve) => setImmediate(resolve));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

function harness(options: { draft?: any; load?: Promise<any>; open?: () => Promise<any[]>; save?: () => Promise<any> } = {}) {
  const nodes = new Map<string, any>();
  const events = new Map<string, Function>();
  const timers = new Map<number, Function>();
  let timer = 0;
  let persisted: any;
  const ready = deferred<void>();
  function node(id: string) {
    if (!nodes.has(id)) nodes.set(id, {
      value: "", textContent: "", disabled: false, readOnly: false, returnValue: "", open: false,
      addEventListener(event: string, fn: Function) { events.set(`${id}:${event}`, fn); },
      focus() {}, showModal() { this.open = true; },
    });
    return nodes.get(id);
  }
  const context: any = {
    document: { querySelector: node, title: "" }, indexedDB: {},
    FileNotesStorage: { load: () => options.load ?? Promise.resolve(options.draft), save: async (state: any) => { persisted = { ...state }; return true; } },
    showOpenFilePicker: options.open, showSaveFilePicker: options.save,
    setTimeout(fn: Function) { timers.set(++timer, fn); return timer; },
    clearTimeout(id: number) { timers.delete(id); },
    requestAnimationFrame(fn: Function) { fn(); },
    addEventListener(event: string, fn: Function) { events.set(`window:${event}`, fn); },
    __veloxReady() { ready.resolve(); },
  };
  context.window = context;
  runInNewContext(model, context);
  runInNewContext(app, context);
  return {
    node, ready: ready.promise,
    click(id: string) { return events.get(`#${id}:click`)?.(); },
    edit(text: string) { node("#editor").value = text; events.get("#editor:input")?.(); },
    async flushDraft() { for (const fn of timers.values()) fn(); timers.clear(); await tick(); return persisted; },
    unload() { let prevented = false; events.get("window:beforeunload")?.({ preventDefault() { prevented = true; } }); return prevented; },
  };
}

test("saving a snapshot never marks edits made during the write as saved", async () => {
  const closed = deferred<void>();
  let written = "";
  const handle = { name: "note.md", queryPermission: async () => "granted", createWritable: async () => ({ write: async (text: string) => { written = text; }, close: () => closed.promise }) };
  const ui = harness({ save: async () => handle });
  await ui.ready;
  ui.edit("first");
  const saving = ui.click("save-document");
  await tick();
  expect(written).toBe("first");
  expect(ui.node("#new-document").disabled).toBe(true);
  ui.click("new-document");
  ui.edit("second");
  closed.resolve();
  await saving;
  expect(ui.node("#editor").value).toBe("second");
  expect(ui.node("#save-state").textContent).toBe("Unsaved changes");
  expect(ui.unload()).toBe(true);
  const draft = await ui.flushDraft();
  expect(draft).toMatchObject({ text: "second", savedText: "first" });
  const reopened = harness({ draft });
  await reopened.ready;
  expect(reopened.node("#editor").value).toBe("second");
  expect(reopened.node("#save-state").textContent).toBe("Unsaved changes");
});

test("save picker cancellation preserves the draft and permits a later save", async () => {
  let canceled = true;
  let written = "";
  const handle = { name: "retry.md", queryPermission: async () => "granted", createWritable: async () => ({ write: async (text: string) => { written = text; }, close: async () => {} }) };
  const ui = harness({ save: async () => {
    if (canceled) throw Object.assign(new Error("cancel"), { name: "AbortError" });
    return handle;
  } });
  await ui.ready;
  ui.edit("keep me");
  await ui.click("save-as-document");
  expect(ui.node("#status").textContent).toBe("Save canceled.");
  expect(ui.node("#editor").value).toBe("keep me");
  expect(ui.unload()).toBe(true);
  expect(ui.node("#save-document").disabled).toBe(false);
  expect((await ui.flushDraft()).text).toBe("keep me");
  canceled = false;
  ui.edit("keep me after cancel");
  await ui.click("save-as-document");
  expect(written).toBe("keep me after cancel");
  expect(ui.node("#save-state").textContent).toBe("Saved to file");
  expect(ui.unload()).toBe(false);
});

test("denied write permission preserves edits and permits retry when the browser grants access", async () => {
  let writes = 0;
  let permission = "denied";
  let written = "";
  const ui = harness({ save: async () => ({ name: "note.md", queryPermission: async () => "prompt", requestPermission: async () => permission, createWritable: async () => { writes++; return { write: async (text: string) => { written = text; }, close: async () => {} }; } }) });
  await ui.ready;
  ui.edit("keep me");
  await ui.click("save-document");
  expect(writes).toBe(0);
  expect(ui.node("#status").textContent).toBe("Save blocked: File access was denied. Your text is still in the editor.");
  expect(ui.unload()).toBe(true);
  expect(ui.node("#editor").value).toBe("keep me");
  expect(ui.node("#save-document").disabled).toBe(false);
  expect((await ui.flushDraft()).text).toBe("keep me");
  // Model a later browser grant; the application must not override a denial.
  permission = "granted";
  ui.edit("keep me after denial");
  await ui.click("save-document");
  expect(writes).toBe(1);
  expect(written).toBe("keep me after denial");
  expect(ui.node("#save-state").textContent).toBe("Saved to file");
  expect(ui.unload()).toBe(false);
});

test("picker denial does not retry, discard text, or claim a saved profile denial", async () => {
  let calls = 0;
  const ui = harness({ save: async () => {
    calls++;
    throw Object.assign(new Error("The request is not allowed by the platform"), { name: "NotAllowedError" });
  } });
  await ui.ready;
  ui.edit("unsaved text");
  await ui.click("save-as-document");
  expect(calls).toBe(1);
  expect(ui.node("#status").textContent).toBe("Save blocked: File access was denied. Your text is still in the editor.");
  expect(ui.node("#save-state").textContent).toBe("Unsaved changes");
  expect(ui.node("#save-document").disabled).toBe(false);
  const draft = await ui.flushDraft();
  const reopened = harness({ draft });
  await reopened.ready;
  expect(reopened.node("#editor").value).toBe("unsaved text");
  expect(reopened.unload()).toBe(true);
});

test("security-context errors and denied Open are classified without changing the document", async () => {
  for (const name of ["SecurityError", "NotAllowedError"]) {
    const ui = harness({ draft: { schemaVersion: 1, text: "original", savedText: "original" }, open: async () => {
      throw Object.assign(new Error("blocked"), { name });
    } });
    await ui.ready;
    ui.click("open-document");
    await tick();
    expect(ui.node("#status").textContent).toContain(name === "SecurityError" ? "unavailable in this context" : "File access was denied");
    expect(ui.node("#editor").value).toBe("original");
  }
});

test("canceled write to an existing handle is not reported as a failure", async () => {
  let canceled = false;
  const handle = { name: "note.md", queryPermission: async () => "granted", createWritable: async () => {
    if (canceled) throw Object.assign(new Error("cancel"), { name: "AbortError" });
    return { write: async () => {}, close: async () => {} };
  } };
  const ui = harness({ save: async () => handle });
  await ui.ready;
  await ui.click("save-as-document");
  ui.edit("new text");
  canceled = true;
  await ui.click("save-document");
  expect(ui.node("#status").textContent).toBe("Save canceled.");
  expect(ui.unload()).toBe(true);
});

test("failed close aborts the stream and retains unsaved text", async () => {
  let aborted = false;
  const ui = harness({ save: async () => ({ name: "note.md", queryPermission: async () => "granted", createWritable: async () => ({ write: async () => {}, close: async () => { throw new Error("disk full"); }, abort: async () => { aborted = true; } }) }) });
  await ui.ready;
  ui.edit("keep me");
  await ui.click("save-document");
  expect(aborted).toBe(true);
  expect(ui.node("#status").textContent).toContain("disk full");
  expect(ui.unload()).toBe(true);
});

test("restore locks editing until persisted content is ready", async () => {
  const load = deferred<any>();
  const ui = harness({ load: load.promise });
  expect(ui.node("#editor").readOnly).toBe(true);
  expect(ui.node("#open-document").disabled).toBe(true);
  load.resolve({ schemaVersion: 1, text: "restored", savedText: "" });
  await ui.ready;
  expect(ui.node("#editor").value).toBe("restored");
  expect(ui.node("#editor").readOnly).toBe(false);
});

test("opening a file establishes a saved baseline and permits a later save", async () => {
  let written = "";
  const handle = { name: "opened.md", getFile: async () => ({ name: "opened.md", size: 5, text: async () => "hello" }), queryPermission: async () => "granted", createWritable: async () => ({ write: async (text: string) => { written = text; }, close: async () => {} }) };
  const ui = harness({ open: async () => [handle] });
  await ui.ready;
  ui.click("open-document");
  await tick();
  expect(ui.node("#editor").value).toBe("hello");
  expect(ui.unload()).toBe(false);
  ui.edit("changed");
  await ui.click("save-document");
  expect(written).toBe("changed");
  expect(ui.unload()).toBe(false);
});

test("open cancellation and oversized files do not replace the current document", async () => {
  for (const open of [async () => { throw Object.assign(new Error("cancel"), { name: "AbortError" }); }, async () => [{ getFile: async () => ({ size: 2 * 1024 * 1024 + 1 }) }]]) {
    const ui = harness({ draft: { schemaVersion: 1, text: "original", savedText: "original" }, open });
    await ui.ready;
    ui.click("open-document");
    await tick();
    expect(ui.node("#editor").value).toBe("original");
    expect(ui.node("#editor").readOnly).toBe(false);
  }
});
