import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./text-editor/app.js", import.meta.url), "utf8");
const tick = async () => { for (let i = 0; i < 12; i++) await Promise.resolve(); };

async function harness(native = true, options: { draft?: unknown; load?: Promise<unknown> } = {}) {
  const nodes = new Map<string, any>();
  function node(id: string) {
    if (!nodes.has(id)) nodes.set(id, {
      value: "", textContent: "", dataset: {}, disabled: false, open: false, returnValue: "",
      listeners: {},
      addEventListener(event: string, fn: Function) { this.listeners[event] = fn; },
      click() { if (!this.disabled) this.listeners.click?.(); },
      focus() {}, showModal() { this.open = true; },
      close(value: string) { this.open = false; this.returnValue = value; this.listeners.close?.(); },
    });
    return nodes.get(id);
  }
  const calls: any[] = [];
  const listeners: any = {};
  let openResult: any = { cancelled: true };
  let saveResult: any = { cancelled: false, name: "saved.txt", target: 1 };
  let failure: any = null;
  let wait: Promise<unknown> | null = null;
  let storedDraft: any = options.draft ?? null;
  let draftFailure = false;
  let draftWait: Promise<unknown> | null = null;
  const draftCalls: any[] = [];
  const timers = new Map<number, Function>();
  let timerID = 0;
  const window: any = { addEventListener: (event: string, fn: Function) => { listeners[event] = fn; } };
  window.EditorDrafts = {
    async load() { return options.load ? await options.load : storedDraft; },
    async save(snapshot: any) {
      draftCalls.push({ method: "save", snapshot });
      if (draftWait) await draftWait;
      if (draftFailure) throw new Error("storage failed");
      storedDraft = structuredClone(snapshot);
    },
    async clear() {
      draftCalls.push({ method: "clear" });
      if (draftFailure) throw new Error("storage failed");
      storedDraft = null;
    },
  };
  async function record(method: string, params: unknown) {
    calls.push({ method, params });
    if (failure) throw failure;
    if (wait) return await wait;
    return method === "file.openText" ? openResult : method === "release" ? null : saveResult;
  }
  if (native) window.velox = {
    invoke: (method: string, params: unknown) => record(method, params),
    saveTextAs: (text: string, name: string) => record("as", { text, name }),
    saveTextTo: (text: string, target: number) => record("to", { text, target }),
  };
  const document: any = { title: "Editor", querySelector: node };
  runInNewContext(source, { document, window, setTimeout: (fn: Function) => { timers.set(++timerID, fn); return timerID; }, clearTimeout: (id: number) => timers.delete(id) });
  await tick();
  return {
    node, calls, listeners, document,
    edit(text: string) { node("#editor").value = text; node("#editor").listeners.input(); },
    async click(id: string) { node(`#${id}-document`).click(); await tick(); },
    setOpen: (value: unknown) => { openResult = value; },
    setSave: (value: unknown) => { saveResult = value; },
    setFailure: (value: unknown) => { failure = value; },
    setWait: (value: Promise<unknown> | null) => { wait = value; },
    draftCalls, storedDraft: () => storedDraft,
    setDraftFailure: (value: boolean) => { draftFailure = value; },
    setDraftWait: (value: Promise<unknown> | null) => { draftWait = value; },
    async flushDraft() { for (const [id, fn] of timers) { timers.delete(id); fn(); } await tick(); },
  };
}

test("no startup native access; first save connects and next save reuses target", async () => {
  const ui = await harness();
  expect(ui.calls).toEqual([]);
  ui.edit("\ud55c\uae00\n<script>literal</script>");
  await ui.click("save");
  expect(ui.calls[0].method).toBe("as");
  expect(ui.calls[0].params.text).toBe(ui.node("#editor").value);
  expect(ui.node("#save-state").textContent).toBe("Saved to file");
  ui.edit("changed");
  await ui.click("save");
  expect(ui.calls[1]).toEqual({ method: "to", params: { text: "changed", target: 1 } });
  await ui.click("save-as");
  expect(ui.calls[2].method).toBe("as");
});

test("save cancellation and native errors retain dirty text and connected target", async () => {
  const ui = await harness();
  await ui.click("save");
  ui.edit("keep");
  ui.setSave({ cancelled: true });
  await ui.click("save-as");
  expect(ui.node("#editor").value).toBe("keep");
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
  for (const code of ["PERMISSION_DENIED", "FILE_CHANGED", "SAVE_RECOVERY_REQUIRED"]) {
    ui.setFailure(Object.assign(new Error("blocked"), { code }));
    await ui.click("save");
    expect(ui.calls.at(-1).params.target).toBe(1);
    expect(ui.node("#editor").value).toBe("keep");
    expect(ui.node("#editor").readOnly).toBe(false);
  }
  ui.setFailure(Object.assign(new Error("expired"), { code: "SAVE_TARGET_INVALID" }));
  await ui.click("save");
  ui.setFailure(null);
  await ui.click("save");
  expect(ui.calls.at(-1).method).toBe("as");
});

test("replacement requires discard, open cancellation preserves text, New releases target", async () => {
  const ui = await harness();
  await ui.click("save");
  ui.edit("keep");
  await ui.click("new");
  ui.node("#discard-dialog").close("cancel");
  await tick();
  expect(ui.node("#editor").value).toBe("keep");
  await ui.click("open");
  ui.node("#discard-dialog").close("discard");
  await tick();
  expect(ui.node("#editor").value).toBe("keep");
  expect(ui.calls.at(-1).method).toBe("file.openText");
  ui.setOpen({ cancelled: false, name: "<b>file.txt</b>", text: "opened" });
  await ui.click("open");
  ui.node("#discard-dialog").close("discard");
  await tick();
  expect(ui.calls.at(-1)).toEqual({ method: "file.releaseSaveTarget", params: { target: 1 } });
  expect(ui.node("#document-name").textContent).toBe("<b>file.txt</b>");
  expect(ui.node("#editor").value).toBe("opened");
  await ui.click("new");
  expect(ui.node("#editor").value).toBe("");
});

test("pending save freezes editing, suppresses duplicates and prevents unloading", async () => {
  const ui = await harness();
  let finish: Function = () => {};
  ui.setWait(new Promise((resolve) => { finish = resolve; }));
  ui.node("#save-document").click();
  expect(ui.node("#editor").readOnly).toBe(true);
  await ui.click("save-as");
  await ui.click("new");
  expect(ui.calls).toHaveLength(1);
  let blocked = false;
  ui.listeners.beforeunload({ preventDefault() { blocked = true; } });
  expect(blocked).toBe(true);
  finish({ cancelled: true });
  await tick();
  expect(ui.node("#editor").readOnly).toBe(false);
});

test("shortcuts respect IME, repeats and discard dialog; dirty text prevents close", async () => {
  const ui = await harness();
  const key = { ctrlKey: true, code: "KeyS", preventDefault() {} };
  ui.listeners.keydown({ ...key, isComposing: true });
  ui.listeners.keydown({ ...key, keyCode: 229 });
  ui.listeners.keydown({ ...key, repeat: true });
  expect(ui.calls).toHaveLength(0);
  ui.listeners.keydown(key);
  await tick();
  expect(ui.calls).toHaveLength(1);
  ui.edit("dirty");
  let blocked = false;
  ui.listeners.beforeunload({ preventDefault() { blocked = true; } });
  expect(blocked).toBe(true);
  await ui.click("new");
  ui.listeners.keydown(key);
  expect(ui.calls).toHaveLength(1);
});

test("browser preview disables native controls without a picker fallback", async () => {
  const ui = await harness(false);
  await ui.click("save");
  await ui.click("open");
  expect(ui.calls).toEqual([]);
  expect(ui.node("#open-document").disabled).toBe(true);
  expect(ui.node("#status").textContent).toBe("Native file access unavailable.");
});

test("restored draft is literal, unsaved and saves through a fresh picker", async () => {
  const ui = await harness(true, { draft: { name: "<b>draft.txt</b>", text: "\ud55c\uae00\n<script>literal</script>" } });
  expect(ui.node("#recovery-dialog").open).toBe(true);
  expect(ui.node("#draft-name").textContent).toBe("<b>draft.txt</b>");
  expect(ui.node("#editor").readOnly).toBe(true);
  expect(ui.calls).toEqual([]);
  let canceled = false;
  ui.node("#recovery-dialog").listeners.cancel({ preventDefault() { canceled = true; } });
  expect(canceled).toBe(true);
  ui.node("#recovery-dialog").close("recover");
  await tick();
  expect(ui.node("#editor").value).toBe("\ud55c\uae00\n<script>literal</script>");
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
  await ui.flushDraft();
  expect(ui.node("#draft-state").textContent).toBe("Draft stored");
  let blocked = false;
  ui.listeners.beforeunload({ preventDefault() { blocked = true; } });
  expect(blocked).toBe(true);
  await ui.click("save");
  expect(ui.calls[0].method).toBe("as");
  expect(ui.storedDraft()).toBeNull();
});

test("empty restored draft remains dirty and failed discard preserves candidate", async () => {
  const ui = await harness(true, { draft: { name: "emptied.txt", text: "" } });
  ui.setDraftFailure(true);
  ui.node("#recovery-dialog").close("discard");
  await tick();
  expect(ui.node("#recovery-dialog").open).toBe(true);
  expect(ui.storedDraft().name).toBe("emptied.txt");
  expect(ui.node("#draft-state").textContent).toBe("Draft recovery unavailable");
  ui.setDraftFailure(false);
  ui.node("#recovery-dialog").close("recover");
  await tick();
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
  await ui.flushDraft();
  expect(ui.storedDraft().text).toBe("");
});

test("startup load freezes editing and explicit discard completes before unlock", async () => {
  let finish: Function = () => {};
  const ui = await harness(true, { load: new Promise(resolve => { finish = resolve; }) });
  expect(ui.node("#editor").readOnly).toBe(true);
  await ui.click("new");
  expect(ui.draftCalls).toEqual([]);
  finish({ name: "old.txt", text: "old" });
  await tick();
  expect(ui.node("#recovery-dialog").open).toBe(true);
  ui.node("#recovery-dialog").close("discard");
  await tick();
  expect(ui.node("#editor").readOnly).toBe(false);
  expect(ui.node("#draft-state").textContent).toBe("No draft");
  expect(ui.draftCalls).toEqual([{ method: "clear" }]);
});

test("debounced writes serialize before clear and never resurrect discarded text", async () => {
  const ui = await harness();
  let finish: Function = () => {};
  ui.setDraftWait(new Promise(resolve => { finish = resolve; }));
  ui.edit("old text");
  await ui.flushDraft();
  expect(ui.node("#draft-state").textContent).toBe("Saving draft...");
  await ui.click("new");
  ui.node("#discard-dialog").close("discard");
  await tick();
  expect(ui.node("#editor").readOnly).toBe(true);
  expect(ui.draftCalls.map(call => call.method)).toEqual(["save"]);
  finish(); await tick();
  expect(ui.draftCalls.map(call => call.method)).toEqual(["save", "clear"]);
  expect(ui.storedDraft()).toBeNull();
  expect(ui.node("#editor").value).toBe("");
});

test("storage failure keeps editing available and never weakens close protection", async () => {
  const ui = await harness();
  ui.setDraftFailure(true);
  ui.edit("keep text");
  await ui.flushDraft();
  expect(ui.node("#draft-state").textContent).toBe("Draft recovery unavailable");
  expect(ui.node("#editor").readOnly).toBe(false);
  let blocked = false;
  ui.listeners.beforeunload({ preventDefault() { blocked = true; } });
  expect(blocked).toBe(true);
  await ui.click("save");
  expect(ui.node("#status").textContent).toBe("File saved. Draft cleanup unavailable.");
  ui.setOpen({ cancelled: false, name: "new.txt", text: "opened" });
  await ui.click("open");
  expect(ui.node("#editor").value).toBe("opened");
  await ui.click("new");
  expect(ui.node("#editor").value).toBe("");
});

test("IME defers drafts until composition ends and rapid edits coalesce", async () => {
  const ui = await harness();
  ui.node("#editor").listeners.compositionstart();
  ui.edit("\ud55c");
  await ui.flushDraft();
  expect(ui.draftCalls).toEqual([]);
  ui.edit("\ud55c\uae00");
  ui.node("#editor").listeners.compositionend();
  ui.edit("\ud55c\uae00 text");
  await ui.flushDraft();
  expect(ui.draftCalls).toHaveLength(1);
  expect(ui.storedDraft().text).toBe("\ud55c\uae00 text");
});
