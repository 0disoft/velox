import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const model = await readFile(new URL("./web/model.js", import.meta.url), "utf8");
const app = await readFile(new URL("./web/app.js", import.meta.url), "utf8");
const storage = await readFile(new URL("./web/storage.js", import.meta.url), "utf8");
const tick = () => new Promise((resolve) => setImmediate(resolve));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

function failure(code: string, message = code) { return Object.assign(new Error(message), { code }); }

function harness(options: {
  draft?: any; load?: Promise<any>; open?: () => Promise<any>;
  save?: (text: string, name: string) => Promise<any>;
  saveTo?: (text: string, target: number) => Promise<any>;
  release?: (target: number) => Promise<any>; unavailable?: boolean;
} = {}) {
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
    FileNotesStorage: { load: () => options.load ?? Promise.resolve(options.draft), save: async (state: any) => { persisted = { ...state }; } },
    velox: options.unavailable ? undefined : {
      invoke(method: string, params?: { target: number }) {
        if (method === "file.openText") return options.open?.() ?? Promise.resolve({ cancelled: true });
        if (method === "file.releaseSaveTarget") return options.release?.(params!.target) ?? Promise.resolve({});
        throw new Error(`unexpected method: ${method}`);
      },
      saveTextAs: options.save ?? (async () => ({ cancelled: true })),
      saveTextTo: options.saveTo ?? (async () => { throw new Error("unexpected connected save"); }),
    },
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
    discard(accepted: boolean) {
      node("#discard-dialog").returnValue = accepted ? "discard" : "cancel";
      events.get("#discard-dialog:close")?.();
    },
    edit(text: string) { node("#editor").value = text; events.get("#editor:input")?.(); },
    async flushDraft() { for (const fn of timers.values()) fn(); timers.clear(); await tick(); return persisted; },
    unload() { let prevented = false; events.get("window:beforeunload")?.({ preventDefault() { prevented = true; } }); return prevented; },
  };
}

test("saving a snapshot never marks edits made during the native write as saved", async () => {
  const saved = deferred<any>();
  let written = "";
  const ui = harness({ save: async (text) => { written = text; return saved.promise; } });
  await ui.ready;
  ui.edit("first");
  const saving = ui.click("save-document");
  await tick();
  expect(written).toBe("first");
  expect(ui.node("#new-document").disabled).toBe(true);
  ui.click("new-document");
  ui.edit("second");
  saved.resolve({ name: "note.md", target: 1 });
  await saving;
  expect(ui.node("#editor").value).toBe("second");
  expect(ui.node("#save-state").textContent).toBe("Unsaved changes");
  expect(ui.unload()).toBe(true);
  const draft = await ui.flushDraft();
  expect(draft).toMatchObject({ text: "second", savedText: "first" });
  const reopened = harness({ draft });
  await reopened.ready;
  expect(reopened.node("#editor").value).toBe("second");
  expect(reopened.unload()).toBe(true);
});

test("Save as cancellation keeps edits and the previous connected target", async () => {
  let selections = 0;
  let connected: any;
  const ui = harness({
    save: async () => ++selections === 1 ? { name: "note.md", target: 7 } : { cancelled: true },
    saveTo: async (text, target) => { connected = { text, target }; return { name: "note.md", target }; },
  });
  await ui.ready;
  await ui.click("save-as-document");
  ui.edit("keep me");
  await ui.click("save-as-document");
  expect(ui.node("#status").textContent).toBe("Save canceled.");
  expect(ui.unload()).toBe(true);
  expect(ui.node("#save-document").disabled).toBe(false);
  await ui.click("save-document");
  expect(connected).toEqual({ text: "keep me", target: 7 });
  expect(ui.unload()).toBe(false);
});

test("native permission denial preserves text and allows an explicit retry", async () => {
  let denied = true;
  const ui = harness({ save: async () => {
    if (denied) throw failure("PERMISSION_DENIED");
    return { name: "note.md", target: 1 };
  } });
  await ui.ready;
  ui.edit("keep me");
  await ui.click("save-document");
  expect(ui.node("#status").textContent).toContain("File access was denied");
  expect(ui.node("#editor").value).toBe("keep me");
  expect(ui.unload()).toBe(true);
  expect((await ui.flushDraft()).text).toBe("keep me");
  denied = false;
  await ui.click("save-document");
  expect(ui.node("#save-state").textContent).toBe("Saved to file");
  expect(ui.unload()).toBe(false);
});

test("external-change conflicts never retry or fall back to overwriting", async () => {
  let selections = 0;
  let writes = 0;
  const ui = harness({
    save: async () => { selections++; return { name: "note.md", target: 1 }; },
    saveTo: async () => { writes++; throw failure("FILE_CHANGED"); },
  });
  await ui.ready;
  await ui.click("save-as-document");
  ui.edit("unsaved edit");
  await ui.click("save-document");
  expect(selections).toBe(1);
  expect(writes).toBe(1);
  expect(ui.node("#status").textContent).toContain("file changed outside this app");
  expect(ui.node("#editor").value).toBe("unsaved edit");
  expect(ui.unload()).toBe(true);
  expect((await ui.flushDraft()).savedText).toBe("");
});

test("an expired target is dropped without silently opening a picker", async () => {
  let selections = 0;
  const ui = harness({
    save: async () => { selections++; return { name: "note.md", target: selections }; },
    saveTo: async () => { throw failure("SAVE_TARGET_INVALID"); },
  });
  await ui.ready;
  await ui.click("save-document");
  ui.edit("still here");
  await ui.click("save-document");
  expect(selections).toBe(1);
  expect(ui.node("#status").textContent).toContain("connection expired");
  expect(ui.unload()).toBe(true);
  await ui.click("save-document");
  expect(selections).toBe(2);
  expect(ui.unload()).toBe(false);
});

test("recovery and disk errors retain the edited buffer and old saved baseline", async () => {
  for (const code of ["SAVE_RECOVERY_REQUIRED", "NATIVE_OPERATION_FAILED"]) {
    const ui = harness({ save: async () => { throw failure(code, "disk failure"); } });
    await ui.ready;
    ui.edit("keep me");
    await ui.click("save-document");
    expect(ui.node("#status").textContent).toContain("disk failure");
    expect(ui.node("#editor").value).toBe("keep me");
    expect(ui.unload()).toBe(true);
    expect(ui.node("#save-document").disabled).toBe(false);
    expect((await ui.flushDraft()).savedText).toBe("");
  }
});

test("restore locks editing and discards both legacy handles and persisted targets", async () => {
  const load = deferred<any>();
  let selections = 0;
  const ui = harness({ load: load.promise, save: async () => { selections++; return { cancelled: true }; } });
  expect(ui.node("#editor").readOnly).toBe(true);
  expect(ui.node("#open-document").disabled).toBe(true);
  load.resolve({ schemaVersion: 1, name: "restored.md", text: "restored", savedText: "", handle: {}, target: 99 });
  await ui.ready;
  expect(ui.node("#editor").value).toBe("restored");
  expect(ui.node("#editor").readOnly).toBe(false);
  await ui.click("save-document");
  expect(selections).toBe(1);
  expect(ui.node("#editor").value).toBe("restored");
});

test("Open releases the old target and requires selection on the next Save", async () => {
  const released: number[] = [];
  const selections: any[] = [];
  const ui = harness({
    open: async () => ({ name: "opened.md", text: "hello" }),
    save: async (text, name) => { selections.push({ text, name }); return { name, target: selections.length }; },
    release: async (target) => { released.push(target); },
  });
  await ui.ready;
  await ui.click("save-document");
  ui.click("open-document");
  await tick();
  expect(released).toEqual([1]);
  expect(ui.node("#editor").value).toBe("hello");
  expect(ui.node("#save-state").textContent).toBe("No save target");
  expect(ui.unload()).toBe(false);
  ui.edit("changed");
  await ui.click("save-document");
  expect(selections[1]).toEqual({ text: "changed", name: "opened.md" });
  expect(ui.unload()).toBe(false);
});

test("canceled, denied and oversized Open preserve the previous save connection", async () => {
  for (const open of [async () => ({ cancelled: true }), async () => { throw failure("PERMISSION_DENIED"); }, async () => { throw failure("PAYLOAD_TOO_LARGE"); }]) {
    let releases = 0;
    let usedTarget = 0;
    const ui = harness({
      open, save: async () => ({ name: "note.md", target: 3 }),
      saveTo: async (_, target) => { usedTarget = target; return { name: "note.md", target }; },
      release: async () => { releases++; },
    });
    await ui.ready;
    ui.edit("original");
    await ui.click("save-document");
    ui.click("open-document");
    await tick();
    expect(ui.node("#editor").value).toBe("original");
    expect(ui.node("#editor").readOnly).toBe(false);
    expect(releases).toBe(0);
    ui.edit("next");
    await ui.click("save-document");
    expect(usedTarget).toBe(3);
  }
});

test("New only releases the target after discard confirmation and clears it", async () => {
  const released: number[] = [];
  let selections = 0;
  const ui = harness({
    save: async () => ({ name: "note.md", target: ++selections }),
    release: async (target) => { released.push(target); },
  });
  await ui.ready;
  await ui.click("save-document");
  ui.edit("unsaved");
  ui.click("new-document");
  ui.discard(false);
  expect(released).toEqual([]);
  expect(ui.node("#editor").value).toBe("unsaved");
  ui.click("new-document");
  ui.discard(true);
  await tick();
  expect(released).toEqual([1]);
  expect(ui.node("#editor").value).toBe("");
  expect(ui.unload()).toBe(false);
  await ui.click("save-document");
  expect(selections).toBe(2);
});

test("release failures do not discard the document on New or Open", async () => {
  for (const action of ["new-document", "open-document"]) {
    const ui = harness({
      open: async () => ({ name: "other.md", text: "other" }),
      save: async () => ({ name: "note.md", target: 1 }),
      release: async () => { throw failure("NATIVE_OPERATION_FAILED"); },
    });
    await ui.ready;
    ui.edit("original");
    await ui.click("save-document");
    ui.click(action);
    await tick();
    expect(ui.node("#editor").value).toBe("original");
    expect(ui.node("#save-document").disabled).toBe(false);
    expect(ui.node("#status").textContent).toContain("failed");
  }
});

test("New freezes editing while its previous target is being released", async () => {
  const released = deferred<void>();
  const ui = harness({ save: async () => ({ name: "note.md", target: 1 }), release: () => released.promise });
  await ui.ready;
  await ui.click("save-document");
  ui.click("new-document");
  expect(ui.node("#editor").readOnly).toBe(true);
  expect(ui.node("#save-document").disabled).toBe(true);
  released.resolve();
  await tick();
  expect(ui.node("#editor").readOnly).toBe(false);
  expect(ui.node("#editor").value).toBe("");
});

test("missing bridge reports unavailable without discarding text or using browser pickers", async () => {
  const ui = harness({ unavailable: true });
  await ui.ready;
  ui.edit("keep me");
  await ui.click("save-document");
  expect(ui.node("#status").textContent).toBe("Native file saving is unavailable.");
  expect(ui.unload()).toBe(true);
  expect(app).not.toMatch(/showOpenFilePicker|showSaveFilePicker|createWritable/);
});

test("IndexedDB stores draft fields only, never write targets or legacy handles", async () => {
  let stored: any;
  let closed = false;
  const database = {
    transaction() {
      const transaction: any = {
        objectStore: () => ({
          put(value: any, key: string) {
            expect(key).toBe("current");
            stored = value;
            const request: any = {};
            queueMicrotask(() => { request.onsuccess(); transaction.oncomplete(); });
            return request;
          },
        }),
      };
      return transaction;
    },
    close() { closed = true; },
  };
  const context: any = {
    indexedDB: {
      open(name: string, version: number) {
        expect(name).toBe("dev.velox.filenotes");
        expect(version).toBe(1);
        const request: any = { result: database };
        queueMicrotask(() => request.onsuccess());
        return request;
      },
    },
  };
  runInNewContext(storage, context);
  const draft = { schemaVersion: 1, name: "note.md", text: "draft", savedText: "saved", updatedAt: null };
  await context.FileNotesStorage.save({ ...draft, target: 42, handle: {}, path: "private" });
  expect(stored).toEqual(draft);
  expect(closed).toBe(true);
});
