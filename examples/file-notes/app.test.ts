import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const model = await readFile(new URL("./web/model.js", import.meta.url), "utf8");
const app = await readFile(new URL("./web/app.js", import.meta.url), "utf8");
const storage = await readFile(new URL("./web/storage.js", import.meta.url), "utf8");
const find = await readFile(new URL("./web/find.js", import.meta.url), "utf8");
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
  title?: (title: string) => Promise<any>;
} = {}) {
  const nodes = new Map<string, any>();
  const events = new Map<string, Function>();
  const timers = new Map<number, Function>();
  let timer = 0;
  let persisted: any;
  const titles: string[] = [];
  const ready = deferred<void>();
  function node(id: string) {
    if (!nodes.has(id)) nodes.set(id, {
      value: "", textContent: "", disabled: false, readOnly: false, returnValue: "", open: false, hidden: false,
      selectionStart: 0, selectionEnd: 0, attributes: {} as Record<string, string>,
      addEventListener(event: string, fn: Function) { events.set(`${id}:${event}`, fn); },
      focus() { context.document.activeElement = this; }, showModal() { this.open = true; },
      select() { this.selectionStart = 0; this.selectionEnd = this.value.length; },
      setSelectionRange(start: number, end: number) { this.selectionStart = start; this.selectionEnd = end; },
      setAttribute(name: string, value: string) { this.attributes[name] = value; },
      click() { if (!this.disabled) return events.get(`${id}:click`)?.(); },
    });
    return nodes.get(id);
  }
  const context: any = {
    document: { querySelector: node, title: "" }, indexedDB: {},
    FileNotesStorage: { load: () => options.load ?? Promise.resolve(options.draft), save: async (state: any) => { persisted = { ...state }; } },
    velox: options.unavailable ? undefined : {
      invoke(method: string, params?: { target?: number; title?: string }) {
        if (method === "window.setTitle") {
          titles.push(params!.title!);
          return options.title?.(params!.title!) ?? Promise.resolve(null);
        }
        if (method === "file.openText") return options.open?.() ?? Promise.resolve({ cancelled: true });
        if (method === "file.releaseSaveTarget") return options.release?.(params!.target!) ?? Promise.resolve({});
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
  runInNewContext(find, context);
  runInNewContext(app, context);
  return {
    node, titles, document: context.document, ready: ready.promise,
    click(id: string) { return node(`#${id}`).click(); },
    key(code: string, options: any = {}) {
      const event = { code, ctrlKey: true, altKey: false, metaKey: false, shiftKey: false,
        repeat: false, isComposing: false, keyCode: 0, defaultPrevented: false,
        ...options, preventDefault() { this.defaultPrevented = true; } };
      events.get("window:keydown")?.(event);
      return event;
    },
    compose(active: boolean) { events.get(`#editor:composition${active ? "start" : "end"}`)?.(); },
    composeSearch(active: boolean) { events.get(`#find-input:composition${active ? "start" : "end"}`)?.(); },
    discard(accepted: boolean) {
      node("#discard-dialog").open = false;
      node("#discard-dialog").returnValue = accepted ? "discard" : "cancel";
      events.get("#discard-dialog:close")?.();
    },
    edit(text: string) { node("#editor").value = text; events.get("#editor:input")?.(); },
    query(text: string) { node("#find-input").value = text; events.get("#find-input:input")?.(); },
    searchKey(key: string, options: any = {}) {
      const event = { key, target: node("#find-input"), ctrlKey: false, altKey: false, metaKey: false,
        shiftKey: false, repeat: false, isComposing: false, defaultPrevented: false,
        ...options, preventDefault() { this.defaultPrevented = true; } };
      events.get("window:keydown")?.(event);
      return event;
    },
    async flushDraft() { for (const fn of timers.values()) fn(); timers.clear(); await tick(); return persisted; },
    unload() { let prevented = false; events.get("window:beforeunload")?.({ preventDefault() { prevented = true; } }); return prevented; },
  };
}

test("native title follows dirty, saved, renamed, open and new transitions without per-keystroke calls", async () => {
  let saves = 0;
  const ui = harness({
    save: async () => ({ name: ++saves === 1 ? "note.md" : "renamed.txt", target: saves }),
    open: async () => ({ name: "opened.md", text: "opened" }),
  });
  await ui.ready;
  expect(ui.titles).toEqual(["Untitled.md · Velox File Notes"]);
  ui.edit("first");
  ui.edit("second");
  expect(ui.titles).toEqual(["Untitled.md · Velox File Notes", "• Untitled.md · Velox File Notes"]);
  ui.edit("");
  expect(ui.titles.at(-1)).toBe("Untitled.md · Velox File Notes");
  ui.edit("write");
  await ui.click("save-document");
  expect(ui.titles.at(-1)).toBe("note.md · Velox File Notes");
  await ui.click("save-as-document");
  expect(ui.titles.at(-1)).toBe("renamed.txt · Velox File Notes");
  await ui.click("open-document");
  await tick();
  expect(ui.titles.at(-1)).toBe("opened.md · Velox File Notes");
  await ui.click("new-document");
  expect(ui.titles.at(-1)).toBe("Untitled.md · Velox File Notes");
  expect(ui.document.title).toBe(ui.titles.at(-1));
});

test("restored dirty title and canceled save keep the correct caption", async () => {
  const ui = harness({ draft: { schemaVersion: 1, name: "복구.md", text: "edit", savedText: "" } });
  await ui.ready;
  expect(ui.titles).toEqual(["• 복구.md · Velox File Notes"]);
  await ui.click("save-as-document");
  expect(ui.titles).toEqual(["• 복구.md · Velox File Notes"]);
});

test("unavailable or rejected native titles never block editing or saving", async () => {
  for (const code of ["METHOD_NOT_FOUND", "PERMISSION_DENIED", "NATIVE_OPERATION_FAILED"]) {
    const ui = harness({
      title: async () => { throw failure(code); },
      save: async () => ({ name: "saved.md", target: 1 }),
    });
    await ui.ready;
    ui.edit("first");
    ui.edit("second");
    await tick();
    expect(ui.titles).toHaveLength(2);
    await ui.click("save-document");
    expect(ui.node("#status").textContent).toBe("saved.md saved.");
    expect(ui.unload()).toBe(false);
  }
  const browser = harness({ unavailable: true });
  await browser.ready;
  browser.edit("text");
  expect(browser.document.title).toBe("• Untitled.md · Velox File Notes");
  expect(browser.titles).toHaveLength(0);
});

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

test("keyboard Save and Save as share native selection and connected-save flows", async () => {
  const selections: any[] = [], writes: any[] = [];
  const ui = harness({
    save: async (text, name) => { selections.push({ text, name }); return { name: "note.md", target: 7 }; },
    saveTo: async (text, target) => { writes.push({ text, target }); return { name: "note.md", target }; },
  });
  await ui.ready;
  ui.edit("first");
  expect(ui.key("KeyS", { key: "\u3134" }).defaultPrevented).toBe(true);
  await tick();
  expect(selections).toEqual([{ text: "first", name: "Untitled.md" }]);
  ui.edit("second");
  ui.key("KeyS");
  await tick();
  expect(writes).toEqual([{ text: "second", target: 7 }]);
  ui.key("KeyS", { shiftKey: true });
  await tick();
  expect(selections[1]).toEqual({ text: "second", name: "note.md" });
  expect(ui.unload()).toBe(false);
});

test("keyboard Open and New preserve discard confirmation and cannot replace a pending action", async () => {
  let opens = 0;
  const ui = harness({ open: async () => { opens++; return { cancelled: true }; } });
  await ui.ready;
  ui.edit("keep this");
  ui.key("KeyO");
  expect(ui.node("#discard-dialog").open).toBe(true);
  ui.key("KeyN");
  ui.key("KeyS");
  expect(opens).toBe(0);
  ui.discard(false);
  await tick();
  expect(ui.node("#editor").value).toBe("keep this");
  ui.key("KeyO");
  ui.key("KeyN");
  ui.discard(true);
  await tick();
  expect(opens).toBe(1);
  expect(ui.node("#editor").value).toBe("keep this");
  ui.key("KeyN");
  ui.discard(true);
  await tick();
  expect(ui.node("#editor").value).toBe("");
  expect(ui.node("#status").textContent).toBe("New document created.");
});

test("shortcuts suppress browser defaults without duplicating restoration or pending file operations", async () => {
  const load = deferred<any>(), saved = deferred<any>();
  let saves = 0;
  const ui = harness({ load: load.promise, save: async () => { saves++; return saved.promise; } });
  expect(ui.key("KeyS").defaultPrevented).toBe(true);
  expect(saves).toBe(0);
  load.resolve(null);
  await ui.ready;
  ui.key("KeyS");
  for (const code of ["KeyS", "KeyO", "KeyN"]) expect(ui.key(code).defaultPrevented).toBe(true);
  expect(saves).toBe(1);
  saved.resolve({ cancelled: true });
  await tick();
  expect(ui.key("KeyS", { repeat: true }).defaultPrevented).toBe(true);
  expect(saves).toBe(1);
});

test("IME and unrelated modifier combinations are left untouched", async () => {
  let saves = 0;
  const ui = harness({ save: async () => { saves++; return { cancelled: true }; } });
  await ui.ready;
  for (const options of [{ isComposing: true }, { keyCode: 229 }, { ctrlKey: false },
    { altKey: true }, { metaKey: true }, { defaultPrevented: true }]) {
    const event = ui.key("KeyS", options);
    expect(event.defaultPrevented).toBe(!!options.defaultPrevented);
  }
  for (const code of ["KeyO", "KeyN"]) expect(ui.key(code, { shiftKey: true }).defaultPrevented).toBe(false);
  expect(ui.key("KeyZ").defaultPrevented).toBe(false);
  ui.compose(true);
  expect(ui.key("KeyS").defaultPrevented).toBe(false);
  expect(saves).toBe(0);
  ui.compose(false);
  ui.key("KeyS");
  await tick();
  expect(saves).toBe(1);
});

test("find is literal, wraps in both directions and does not change dirty state or draft", async () => {
  const ui = harness();
  await ui.ready;
  ui.edit("\ud55c\uae00 aa \ud55c\uae00\n\ud83d\ude42 [x] [x]");
  const before = await ui.flushDraft();
  expect(ui.key("KeyF").defaultPrevented).toBe(true);
  expect(ui.node("#find-bar").hidden).toBe(false);
  ui.query("\ud55c\uae00");
  expect(ui.node("#find-count").textContent).toBe("1 / 2");
  expect(ui.node("#editor").selectionStart).toBe(0);
  ui.searchKey("Enter");
  expect(ui.node("#find-count").textContent).toBe("2 / 2");
  expect(ui.node("#editor").selectionStart).toBe(6);
  ui.searchKey("Enter");
  expect(ui.node("#editor").selectionStart).toBe(0);
  ui.searchKey("Enter", { shiftKey: true });
  expect(ui.node("#editor").selectionStart).toBe(6);
  ui.query("[x]");
  expect(ui.node("#find-count").textContent).toBe("1 / 2");
  expect(ui.node("#editor").selectionStart).toBe(12);
  expect(await ui.flushDraft()).toEqual(before);
  expect(ui.unload()).toBe(true);
  ui.searchKey("Escape");
  expect(ui.node("#find-bar").hidden).toBe(true);
});

test("find refreshes after edits and New, with empty query and no-result controls", async () => {
  const ui = harness();
  await ui.ready;
  ui.edit("one one");
  ui.key("KeyF");
  ui.query("one");
  ui.edit("two");
  expect(ui.node("#find-count").textContent).toBe("0 / 0");
  expect(ui.node("#find-next").disabled).toBe(true);
  ui.query("");
  expect(ui.node("#find-count").textContent).toBe("");
  ui.query("two");
  expect(ui.node("#find-count").textContent).toBe("1 / 1");
  ui.key("KeyN");
  expect(ui.searchKey("Escape").defaultPrevented).toBe(false);
  expect(ui.node("#find-bar").hidden).toBe(false);
  ui.discard(true);
  await tick();
  expect(ui.node("#find-count").textContent).toBe("0 / 0");
});

test("find guards repeated Enter, IME and pending writes", async () => {
  const saved = deferred<any>();
  const ui = harness({ save: () => saved.promise });
  await ui.ready;
  ui.edit("one one");
  ui.key("KeyF");
  ui.query("one");
  ui.searchKey("Enter", { repeat: true });
  ui.searchKey("Enter", { isComposing: true });
  ui.searchKey("Enter", { keyCode: 229 });
  expect(ui.node("#editor").selectionStart).toBe(0);
  const saving = ui.click("save-document");
  ui.click("find-next");
  ui.searchKey("Enter");
  expect(ui.node("#editor").selectionStart).toBe(0);
  saved.resolve({ cancelled: true });
  await saving;
  ui.searchKey("Enter");
  expect(ui.node("#editor").selectionStart).toBe(4);
});

test("find refreshes a query changed during saving and defers IME query input until completion", async () => {
  const saved = deferred<any>();
  const ui = harness({ save: () => saved.promise });
  await ui.ready;
  ui.edit("one one two");
  ui.key("KeyF");
  ui.query("one");
  const saving = ui.click("save-document");
  ui.query("two");
  saved.resolve({ cancelled: true });
  await saving;
  expect(ui.node("#find-count").textContent).toBe("0 / 1");
  ui.searchKey("Enter");
  expect(ui.node("#editor").selectionStart).toBe(8);
  ui.composeSearch(true);
  ui.query("one");
  ui.searchKey("Enter");
  expect(ui.node("#editor").selectionStart).toBe(8);
  ui.composeSearch(false);
  expect(ui.node("#find-count").textContent).toBe("1 / 2");
  expect(ui.node("#editor").selectionStart).toBe(0);
});
