import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./text-editor/app.js", import.meta.url), "utf8");
const findSource = await readFile(new URL("./text-editor/find.js", import.meta.url), "utf8");
const positionSource = await readFile(new URL("./text-editor/positions.js", import.meta.url), "utf8");
const tick = async () => { for (let i = 0; i < 12; i++) await Promise.resolve(); };

async function harness(native = true, options: { draft?: unknown; load?: Promise<unknown> } = {}) {
  const nodes = new Map<string, any>();
  function node(id: string) {
    if (!nodes.has(id)) nodes.set(id, {
      value: "", textContent: "", dataset: {}, disabled: false, open: false, returnValue: "", hidden: true, checked: false,
      selectionStart: 0, selectionEnd: 0,
      listeners: {},
      addEventListener(event: string, fn: Function) { this.listeners[event] = fn; },
      click() { if (!this.disabled) this.listeners.click?.(); },
      focus() { document.activeElement = this; }, select() {},
      setAttribute(name: string, value: string) { this[name] = value; },
      setSelectionRange(start: number, end: number, direction = "none") { this.selectionStart = start; this.selectionEnd = end; this.selectionDirection = direction; this.listeners.select?.(); },
      showModal() { this.open = true; },
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
  window.setTimeout = (fn: Function) => { timers.set(++timerID, fn); return timerID; };
  window.clearTimeout = (id: number) => timers.delete(id);
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
  const document: any = { title: "Editor", querySelector: node, addEventListener() {} };
  let editorValue = "";
  Object.defineProperty(node("#editor"), "value", {
    get: () => editorValue,
    set: (value: string) => { editorValue = value.replace(/\r\n?/g, "\n"); },
  });
  node("#editor").wrap = "soft";
  node("#editor").dataset.fontSize = "18";
  node("#font-size").textContent = "18 px";
  node("#word-wrap").setAttribute("aria-pressed", "true");
  runInNewContext(positionSource, { document, window, Intl });
  runInNewContext(findSource, { document, window, TextEncoder });
  runInNewContext(source, { document, window, TextEncoder, setTimeout: (fn: Function) => { timers.set(++timerID, fn); return timerID; }, clearTimeout: (id: number) => timers.delete(id) });
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

test("opened line endings stay clean and round-trip unchanged or edited", async () => {
  for (const raw of ["a\nb\n", "a\r\nb\r\n", "a\rb\r", "a\r\nb\nc\r", "no newline"]) {
    const ui = await harness();
    ui.setOpen({ cancelled: false, name: "lines.txt", text: raw });
    await ui.click("open");
    expect(ui.node("#save-state").dataset.dirty).toBe("false");
    await ui.click("save");
    expect(ui.calls.at(-1).params.text).toBe(raw);
    const eol = raw.match(/\r\n|\r|\n/)?.[0] ?? "\n";
    ui.edit(ui.node("#editor").value + "changed\n");
    await ui.click("save");
    expect(ui.calls.at(-1).params.text).toBe(ui.node("#editor").value.replace(/\n/g, eol));
    expect(ui.calls.at(-1).method).toBe("to");
    expect(ui.node("#save-state").dataset.dirty).toBe("false");
  }
});

test("draft recovery retains raw line endings and New resets to LF", async () => {
  const raw = "a\r\nb\nc\r";
  const ui = await harness(true, { draft: { name: "draft.txt", text: raw } });
  ui.node("#recovery-dialog").close("recover"); await tick();
  await ui.flushDraft();
  expect(ui.storedDraft().text).toBe(raw);
  await ui.click("save-as");
  expect(ui.calls.at(-1).params.text).toBe(raw);
  ui.edit("changed\ntext\n"); await ui.flushDraft();
  expect(ui.storedDraft().text).toBe("changed\r\ntext\r\n");
  await ui.click("new"); ui.node("#discard-dialog").close("discard"); await tick();
  ui.edit("new\ntext\n"); await ui.click("save");
  expect(ui.calls.at(-1).params.text).toBe("new\ntext\n");
});

test("CRLF expansion is size-checked before picker or connected write", async () => {
  const ui = await harness();
  ui.setOpen({ cancelled: false, name: "lines.txt", text: "a\r\n" });
  await ui.click("open");
  ui.edit("\n".repeat(1024 * 1024)); await ui.click("save");
  expect(new TextEncoder().encode(ui.calls.at(-1).params.text).length).toBe(2 * 1024 * 1024);
  ui.edit("\n".repeat(1024 * 1024 + 1));
  const calls = ui.calls.length;
  for (const id of ["save", "save-as"]) {
    await ui.click(id);
    expect(ui.calls).toHaveLength(calls);
    expect(ui.node("#status").textContent).toContain("PAYLOAD_TOO_LARGE");
    expect(ui.node("#save-state").dataset.dirty).toBe("true");
  }
  ui.edit("\uD55C".repeat(699050) + "\n"); await ui.click("save");
  expect(new TextEncoder().encode(ui.calls.at(-1).params.text).length).toBe(2 * 1024 * 1024);
  const beforeUnicodeOverflow = ui.calls.length;
  ui.edit("\uD55C".repeat(699051) + "\n"); await ui.click("save-as");
  expect(ui.calls).toHaveLength(beforeUnicodeOverflow);
  expect(ui.node("#status").textContent).toContain("PAYLOAD_TOO_LARGE");
});

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

test("find wraps literal matches and case toggle does not touch save or drafts", async () => {
  const ui = await harness();
  ui.edit("Text text TEXT [x] \ud55c\uae00");
  await ui.flushDraft();
  const writes = ui.draftCalls.length;
  await ui.click("find");
  expect(ui.node("#find-bar").hidden).toBe(false);
  expect(ui.document.activeElement).toBe(ui.node("#find-input"));
  const input = ui.node("#find-input");
  input.value = "text";
  input.listeners.input();
  expect(ui.node("#find-count").textContent).toBe("1 / 3");
  expect(ui.node("#editor").selectionStart).toBe(0);
  ui.node("#find-next").click();
  expect(ui.node("#editor").selectionStart).toBe(5);
  ui.node("#find-previous").click();
  ui.node("#find-previous").click();
  expect(ui.node("#find-count").textContent).toBe("3 / 3");
  ui.node("#find-case").checked = true;
  ui.node("#find-case").listeners.change();
  expect(ui.node("#find-count").textContent).toBe("1 / 1");
  expect(ui.node("#editor").selectionStart).toBe(5);
  input.value = "[x]"; input.listeners.input();
  expect(ui.node("#editor").selectionStart).toBe(15);
  expect(ui.node("#editor").selectionEnd).toBe(18);
  input.value = "missing"; input.listeners.input();
  expect(ui.node("#find-count").textContent).toBe("0 / 0");
  expect(ui.node("#find-next").disabled).toBe(true);
  expect(ui.node("#editor").value).toBe("Text text TEXT [x] \ud55c\uae00");
  expect(ui.calls).toEqual([]);
  expect(ui.draftCalls).toHaveLength(writes);
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
});

test("find shortcuts preserve IME and modal guards, Enter moves and Escape restores editor focus", async () => {
  const ui = await harness(false);
  ui.edit("\ud55c\uae00 \ud55c\uae00");
  const key = { ctrlKey: true, code: "KeyF", preventDefault() {} };
  ui.listeners.keydown({ ...key, isComposing: true });
  ui.listeners.keydown({ ...key, keyCode: 229 });
  ui.listeners.keydown({ ...key, repeat: true });
  expect(ui.node("#find-bar").hidden).toBe(true);
  ui.listeners.keydown(key);
  const input = ui.node("#find-input");
  input.value = "\ud55c\uae00"; input.listeners.input();
  expect(ui.node("#find-count").textContent).toBe("1 / 2");
  input.listeners.compositionstart();
  input.value = "\ud55c"; input.listeners.input();
  ui.listeners.keydown({ ctrlKey: true, code: "KeyS", target: input, preventDefault() {} });
  expect(ui.calls).toEqual([]);
  ui.listeners.keydown({ key: "Escape", isComposing: true, target: input, preventDefault() {} });
  expect(ui.node("#find-bar").hidden).toBe(false);
  expect(ui.node("#find-count").textContent).toBe("1 / 2");
  input.value = "\ud55c\uae00"; input.listeners.compositionend();
  ui.listeners.keydown({ key: "Enter", target: input, preventDefault() {} });
  expect(ui.node("#editor").selectionStart).toBe(3);
  ui.listeners.keydown({ key: "Enter", shiftKey: true, target: input, preventDefault() {} });
  expect(ui.node("#editor").selectionStart).toBe(0);
  ui.listeners.keydown({ key: "Escape", target: input, preventDefault() {} });
  expect(ui.node("#find-bar").hidden).toBe(true);
  expect(ui.document.activeElement).toBe(ui.node("#editor"));
  await ui.click("new");
  ui.listeners.keydown(key);
  expect(ui.node("#find-bar").hidden).toBe(true);
});

test("find refreshes after editor changes, recovery and document replacement", async () => {
  const ui = await harness(true, { draft: { name: "draft.txt", text: "keep keep" } });
  await ui.click("find");
  expect(ui.node("#find-bar").hidden).toBe(true);
  ui.node("#recovery-dialog").close("recover"); await tick();
  await ui.click("find");
  const input = ui.node("#find-input");
  input.value = "keep"; input.listeners.input();
  expect(ui.node("#find-count").textContent).toBe("1 / 2");
  ui.edit("keep");
  expect(ui.node("#find-count").textContent).toBe("0 / 1");
  await ui.click("save");
  expect(ui.calls[0].method).toBe("as");
  ui.setOpen({ name: "new.txt", text: "nothing", cancelled: false });
  await ui.click("open");
  expect(ui.node("#find-count").textContent).toBe("0 / 0");
  expect(ui.node("#find-next").disabled).toBe(true);
});

test("find returns original UTF-16 offsets, literal non-overlapping results and bounded dense state", () => {
  const context: any = {};
  runInNewContext(findSource, context);
  const find = context.EditorFind.find;
  expect(find("aaaa", "aa")).toEqual({ count: 2, index: 1, start: 0, length: 2 });
  expect(find("[x] .* [x]", "[x]")).toEqual({ count: 2, index: 1, start: 0, length: 3 });
  expect(find("\u0130 Text \ud83d\ude42 text", "text")).toEqual({ count: 2, index: 1, start: 2, length: 4 });
  expect(find("\ud83d\ude42\n\ud55c\uae00", "\ud55c\uae00").start).toBe(3);
  expect(find("text", "").count).toBe(0);
  const text = "x".repeat(2 << 20);
  expect(find(text, "x", text.length - 2)).toEqual({ count: text.length, index: text.length, start: text.length - 1, length: 1 });
});

test("explicit close cancels pending find composition instead of trapping the bar", async () => {
  const ui = await harness();
  ui.edit("\ud55c\uae00");
  await ui.click("find");
  const input = ui.node("#find-input");
  input.listeners.compositionstart();
  input.value = "\ud55c"; input.listeners.input();
  ui.node("#find-close").click();
  expect(ui.node("#find-bar").hidden).toBe(true);
  expect(ui.document.activeElement).toBe(ui.node("#editor"));
  input.listeners.compositionend();
  expect(ui.node("#find-bar").hidden).toBe(true);
  expect(ui.document.activeElement).toBe(ui.node("#editor"));
  await ui.click("find");
  expect(ui.node("#find-bar").hidden).toBe(false);
});

test("Escape closes stale IME state but not an actual composing key event", async () => {
  const ui = await harness(false);
  await ui.click("find");
  const input = ui.node("#find-input");
  input.listeners.compositionstart();
  ui.listeners.keydown({ key: "Escape", keyCode: 229, isComposing: true, target: input, preventDefault() {} });
  expect(ui.node("#find-bar").hidden).toBe(false);
  ui.listeners.keydown({ key: "Escape", keyCode: 229, isComposing: false, target: input, preventDefault() {} });
  expect(ui.node("#find-bar").hidden).toBe(true);
  expect(ui.document.activeElement).toBe(ui.node("#editor"));
});

test("empty query and empty document can close by button or input Escape without changing text", async () => {
  for (const text of ["", "\ud55c\uae00"]) {
    const ui = await harness(false);
    ui.edit(text);
    await ui.click("find");
    ui.node("#find-input").listeners.compositionstart();
    ui.node("#find-close").click();
    expect(ui.node("#find-bar").hidden).toBe(true);
    await ui.click("find");
    const input = ui.node("#find-input");
    input.listeners.compositionstart();
    input.listeners.blur();
    let prevented = false;
    input.listeners.keydown({ key: "Escape", target: input, preventDefault() { prevented = true; } });
    expect(prevented).toBe(true);
    expect(ui.node("#find-bar").hidden).toBe(true);
    expect(ui.node("#editor").value).toBe(text);
    expect(ui.document.activeElement).toBe(ui.node("#editor"));
  }
});

function replaceFields(ui: any, query: string, value: string) {
  ui.listeners.keydown({ ctrlKey: true, code: "KeyH", preventDefault() {} });
  const input = ui.node("#find-input");
  input.value = query; input.listeners.input();
  ui.node("#replace-input").value = value;
  ui.node("#replace-input").listeners.input();
}

test("replace current and all update dirty text and draft without changing native target", async () => {
  const ui = await harness();
  ui.edit("Text text TEXT");
  await ui.click("save");
  replaceFields(ui, "text", "\ud55c\uae00");
  expect(ui.node("#replace-row").hidden).toBe(false);
  expect(ui.node("#replace-all").title).toBe("Replace all (3)");
  ui.node("#replace-one").click();
  expect(ui.node("#editor").value).toBe("\ud55c\uae00 text TEXT");
  expect(ui.node("#editor").selectionStart).toBe(3);
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
  ui.node("#replace-all").click();
  expect(ui.node("#editor").value).toBe("\ud55c\uae00 \ud55c\uae00 \ud55c\uae00");
  expect(ui.node("#replace-all").disabled).toBe(true);
  await ui.flushDraft();
  expect(ui.storedDraft().text).toBe(ui.node("#editor").value);
  expect(ui.calls).toHaveLength(1);
  await ui.click("save");
  expect(ui.calls.at(-1)).toEqual({ method: "to", params: { text: "\ud55c\uae00 \ud55c\uae00 \ud55c\uae00", target: 1 } });
});

test("replace undo is one transaction, survives close/save and tracks current saved baseline", async () => {
  const ui = await harness();
  ui.edit("aa aa");
  await ui.click("save");
  replaceFields(ui, "aa", "bb");
  ui.node("#replace-all").click();
  ui.node("#replace-undo").click();
  expect(ui.node("#editor").value).toBe("aa aa");
  expect(ui.node("#save-state").dataset.dirty).toBe("false");
  await ui.flushDraft();
  expect(ui.storedDraft()).toBeNull();
  expect(ui.node("#replace-undo").disabled).toBe(true);
  ui.node("#replace-all").click();
  ui.setSave({ cancelled: true });
  await ui.click("save-as");
  expect(ui.node("#replace-undo").disabled).toBe(false);
  ui.setSave({ cancelled: false, name: "saved.txt", target: 1 });
  await ui.click("save");
  ui.node("#find-close").click();
  ui.listeners.keydown({ ctrlKey: true, code: "KeyZ", target: ui.node("#editor"), preventDefault() {} });
  expect(ui.node("#editor").value).toBe("aa aa");
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
  await ui.flushDraft();
  expect(ui.storedDraft().text).toBe("aa aa");
});

test("typing and successful document replacement invalidate only replacement undo", async () => {
  const ui = await harness();
  ui.edit("aa");
  replaceFields(ui, "aa", "bb");
  ui.node("#replace-all").click();
  ui.edit("bb typed");
  expect(ui.node("#replace-undo").disabled).toBe(true);
  let prevented = false;
  ui.listeners.keydown({ ctrlKey: true, code: "KeyZ", target: ui.node("#editor"), preventDefault() { prevented = true; } });
  expect(prevented).toBe(false);
  ui.edit("aa"); replaceFields(ui, "aa", "bb"); ui.node("#replace-all").click();
  await ui.click("save");
  ui.setOpen({ cancelled: true }); await ui.click("open");
  expect(ui.node("#replace-undo").disabled).toBe(false);
  ui.setOpen({ cancelled: false, name: "same.txt", text: "bb" }); await ui.click("open");
  expect(ui.node("#replace-undo").disabled).toBe(true);
  replaceFields(ui, "bb", "aa"); ui.node("#replace-all").click();
  await ui.click("new"); ui.node("#discard-dialog").close("discard"); await tick();
  expect(ui.node("#replace-undo").disabled).toBe(true);
  expect(ui.node("#find-count").textContent).toBe("0 / 0");
});

test("replace respects empty query, deletion, no-op and UTF-8 expansion rejection", async () => {
  const ui = await harness(false);
  ui.edit("keep keep"); await ui.flushDraft();
  replaceFields(ui, "", "x");
  expect(ui.node("#replace-all").disabled).toBe(true);
  replaceFields(ui, "keep", "keep"); ui.node("#replace-all").click();
  expect(ui.node("#replace-undo").disabled).toBe(true);
  expect(ui.node("#status").textContent).toBe("Document unchanged.");
  replaceFields(ui, "keep", ""); ui.node("#replace-all").click();
  expect(ui.node("#editor").value).toBe(" ");
  ui.node("#replace-undo").click();
  expect(ui.node("#editor").value).toBe("keep keep");
  replaceFields(ui, "keep", "\ud55c".repeat(400000)); ui.node("#replace-all").click();
  expect(ui.node("#editor").value).toBe("keep keep");
  expect(ui.node("#status").textContent).toContain("2 MiB");
  expect(ui.calls).toEqual([]);
});

test("replace shortcut, IME, busy and modal guards preserve document", async () => {
  const ui = await harness();
  ui.edit("aa aa");
  const shortcut = { ctrlKey: true, code: "KeyH", preventDefault() {} };
  for (const guard of [{isComposing:true}, {keyCode:229}, {repeat:true}]) ui.listeners.keydown({ ...shortcut, ...guard });
  expect(ui.node("#replace-row").hidden).toBe(true);
  replaceFields(ui, "aa", "bb");
  for (const id of ["#find-input", "#replace-input", "#editor"]) {
    ui.node(id).listeners.compositionstart();
    expect(ui.node("#replace-all").disabled).toBe(true);
    ui.node("#replace-all").click();
    expect(ui.node("#editor").value).toBe("aa aa");
    ui.node(id).listeners.compositionend();
  }
  await ui.click("new");
  ui.node("#replace-all").click();
  expect(ui.node("#editor").value).toBe("aa aa");
  ui.node("#discard-dialog").close("cancel");
  let finish: any;
  ui.setWait(new Promise(resolve => { finish = resolve; }));
  ui.node("#save-document").click();
  expect(ui.node("#replace-all").disabled).toBe(true);
  ui.node("#replace-all").click();
  expect(ui.node("#editor").value).toBe("aa aa");
  finish({ cancelled: true }); await tick();
  ui.node("#replace-all").click();
  expect(ui.node("#editor").value).toBe("bb bb");
  ui.node("#replace-input").listeners.compositionstart();
  ui.node("#find-close").click();
  ui.node("#replace-input").listeners.compositionend();
  expect(ui.node("#find-bar").hidden).toBe(true);
  expect(ui.document.activeElement).toBe(ui.node("#editor"));
});

test("replacement of a restored draft stays dirty and undo survives rejected expansion", async () => {
  const ui = await harness(true, { draft: { name: "draft.txt", text: "aa aa" } });
  ui.node("#recovery-dialog").close("recover"); await tick();
  replaceFields(ui, "aa", "bb");
  ui.node("#replace-all").click();
  replaceFields(ui, "bb", "x".repeat(2 << 20));
  ui.node("#replace-all").click();
  expect(ui.node("#editor").value).toBe("bb bb");
  expect(ui.node("#replace-undo").disabled).toBe(false);
  ui.node("#replace-undo").click();
  expect(ui.node("#editor").value).toBe("aa aa");
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
  await ui.flushDraft();
  expect(ui.storedDraft().text).toBe("aa aa");
  await ui.click("save");
  expect(ui.calls.at(-1).method).toBe("as");
});

test("position updates through find, replace, undo, open and new without extra native calls", async () => {
  const ui = await harness();
  expect(ui.node("#cursor-position").textContent).toBe("Ln 1, Col 1");
  ui.edit("\ud55c\n\ud83d\ude42\ud55c");
  ui.node("#editor").setSelectionRange(0, 5, "backward");
  expect(ui.node("#cursor-position").textContent).toBe("Ln 1, Col 1 | Selected 4");
  replaceFields(ui, "\ud55c", "abc");
  expect(ui.node("#cursor-position").textContent).toBe("Ln 1, Col 2 | Selected 1");
  ui.node("#replace-all").click();
  expect(ui.node("#cursor-position").textContent).toBe("Ln 1, Col 4 | Selected 3");
  ui.node("#replace-undo").click();
  expect(ui.node("#cursor-position").textContent).toBe("Ln 1, Col 2 | Selected 1");
  expect(ui.calls).toEqual([]);
  await ui.click("save");
  ui.setOpen({ cancelled: false, name: "next.txt", text: "next\nline" });
  await ui.click("open");
  ui.node("#editor").setSelectionRange(9, 9);
  expect(ui.node("#cursor-position").textContent).toBe("Ln 2, Col 5");
  await ui.click("new");
  expect(ui.node("#cursor-position").textContent).toBe("Ln 1, Col 1");
});

test("position defers composition and refreshes after draft recovery", async () => {
  const ui = await harness(true, { draft: { name: "draft.txt", text: "\u1112\u1161\u11ab\n\ud83d\udc69\u200d\ud83d\udcbb" } });
  ui.node("#recovery-dialog").close("recover"); await tick();
  ui.node("#editor").setSelectionRange(9, 9);
  expect(ui.node("#cursor-position").textContent).toBe("Ln 2, Col 2");
  ui.node("#editor").listeners.compositionstart();
  ui.edit("\ud55c\uae00");
  expect(ui.node("#cursor-position").textContent).toBe("Ln ..., Col ...");
  ui.node("#editor").listeners.compositionend();
  expect(ui.node("#cursor-position").textContent).toBe("Ln 1, Col 3");
  await ui.flushDraft();
  expect(ui.storedDraft().text).toBe("\ud55c\uae00");
  expect(ui.calls).toEqual([]);
});

test("word wrap preserves text, selection, logical position and draft/save authority", async () => {
  const ui = await harness();
  ui.edit("\ud55c\uae00 ".repeat(80) + "\n\ud83d\udc69\u200d\ud83d\udcbb");
  await ui.click("save");
  const editor = ui.node("#editor"), toggle = ui.node("#word-wrap");
  editor.setSelectionRange(2, 120, "backward");
  editor.scrollTop = 48; editor.scrollLeft = 12;
  const text = editor.value, position = ui.node("#cursor-position").textContent;
  const calls = ui.calls.length, drafts = ui.draftCalls.length;
  toggle.click();
  expect(editor.wrap).toBe("off");
  expect(toggle["aria-pressed"]).toBe("false");
  expect(toggle.title).toBe("Word wrap (off)");
  expect(editor.value).toBe(text);
  expect([editor.selectionStart, editor.selectionEnd, editor.selectionDirection]).toEqual([2, 120, "backward"]);
  expect([editor.scrollTop, editor.scrollLeft]).toEqual([48, 12]);
  expect(ui.node("#cursor-position").textContent).toBe(position);
  expect(ui.node("#save-state").dataset.dirty).toBe("false");
  expect(ui.document.activeElement).toBe(editor);
  toggle.click();
  expect(editor.wrap).toBe("soft");
  expect(toggle["aria-pressed"]).toBe("true");
  expect(toggle.title).toBe("Word wrap (on)");
  await tick();
  expect(ui.calls.length).toBe(calls);
  expect(ui.draftCalls.length).toBe(drafts);
  ui.edit(text + "unsaved");
  await ui.flushDraft();
  const stored = structuredClone(ui.storedDraft());
  toggle.click(); await ui.flushDraft();
  expect(ui.storedDraft()).toEqual(stored);
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
  await ui.click("save");
  expect(ui.calls.at(-1).method).toBe("to");
  expect(ui.calls.at(-1).params.target).toBe(1);
});

test("word wrap stays available without native APIs but guards IME, busy and dialogs", async () => {
  const preview = await harness(false);
  preview.node("#word-wrap").click();
  expect(preview.node("#editor").wrap).toBe("off");
  expect(preview.calls).toEqual([]);
  const ui = await harness();
  const toggle = ui.node("#word-wrap"), editor = ui.node("#editor");
  editor.listeners.compositionstart();
  expect(toggle.disabled).toBe(true);
  toggle.listeners.click();
  expect(editor.wrap).toBe("soft");
  editor.listeners.compositionend();
  expect(toggle.disabled).toBe(false);
  ui.edit("keep"); await ui.click("new");
  expect(toggle.disabled).toBe(true);
  toggle.listeners.click();
  expect(editor.wrap).toBe("soft");
  ui.node("#discard-dialog").close("cancel");
  expect(toggle.disabled).toBe(false);
  let finish: (value: unknown) => void = () => {};
  ui.setWait(new Promise(resolve => { finish = resolve; }));
  await ui.click("save");
  expect(toggle.disabled).toBe(true);
  toggle.listeners.click();
  expect(editor.wrap).toBe("soft");
  finish({cancelled:true}); await tick();
  expect(toggle.disabled).toBe(false);
  const recovery = await harness(true, {draft:{name:"draft.txt",text:"draft"}});
  expect(recovery.node("#word-wrap").disabled).toBe(true);
  recovery.node("#word-wrap").listeners.click();
  expect(recovery.node("#editor").wrap).toBe("soft");
  recovery.node("#recovery-dialog").close("recover"); await tick();
  expect(recovery.node("#word-wrap").disabled).toBe(false);
});

test("word wrap is session-only and does not clear replacement undo or change New/Open", async () => {
  const ui = await harness();
  ui.edit("one one");
  replaceFields(ui, "one", "two");
  ui.node("#replace-all").click();
  ui.node("#word-wrap").click();
  ui.node("#replace-undo").click();
  expect(ui.node("#editor").value).toBe("one one");
  expect(ui.node("#editor").wrap).toBe("off");
  await ui.click("save");
  ui.setOpen({cancelled:false,name:"next.txt",text:"next\nline"});
  await ui.click("open");
  expect(ui.node("#editor").wrap).toBe("off");
  await ui.click("new");
  expect(ui.node("#editor").wrap).toBe("off");
  const fresh = await harness();
  expect(fresh.node("#editor").wrap).toBe("soft");
});

test("editor font size stays bounded, restores default and preserves text/selection/save/draft", async () => {
  const ui = await harness();
  const editor = ui.node("#editor"), decrease = ui.node("#font-decrease"), increase = ui.node("#font-increase"), reset = ui.node("#font-reset");
  expect(reset.disabled).toBe(true);
  ui.edit("\ud55c\uae00 long line\n\ud83d\udc69\u200d\ud83d\udcbb");
  await ui.click("save");
  editor.setSelectionRange(2, 11, "backward");
  editor.scrollTop = 20; editor.scrollLeft = 14;
  const text = editor.value, position = ui.node("#cursor-position").textContent;
  const calls = ui.calls.length, drafts = ui.draftCalls.length;
  increase.click();
  expect(editor.dataset.fontSize).toBe("20");
  expect(ui.node("#font-size").textContent).toBe("20 px");
  expect(reset.disabled).toBe(false);
  expect(editor.value).toBe(text);
  expect([editor.selectionStart, editor.selectionEnd, editor.selectionDirection]).toEqual([2, 11, "backward"]);
  expect([editor.scrollTop, editor.scrollLeft]).toEqual([20, 14]);
  expect(ui.document.activeElement).toBe(editor);
  expect(ui.node("#cursor-position").textContent).toBe(position);
  expect(ui.node("#save-state").dataset.dirty).toBe("false");
  for (let i = 0; i < 12; i++) increase.listeners.click();
  expect(editor.dataset.fontSize).toBe("28");
  expect(increase.disabled).toBe(true);
  for (let i = 0; i < 12; i++) decrease.listeners.click();
  expect(editor.dataset.fontSize).toBe("14");
  expect(decrease.disabled).toBe(true);
  reset.click();
  expect(editor.dataset.fontSize).toBe("18");
  expect(reset.disabled).toBe(true);
  await tick();
  expect(ui.calls.length).toBe(calls);
  expect(ui.draftCalls.length).toBe(drafts);
  ui.edit(text + "unsaved"); await ui.flushDraft();
  const stored = structuredClone(ui.storedDraft()), draftCalls = ui.draftCalls.length;
  decrease.click(); await ui.flushDraft();
  expect(ui.storedDraft()).toEqual(stored);
  expect(ui.draftCalls.length).toBe(draftCalls);
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
  await ui.click("save");
  expect(ui.calls.at(-1).method).toBe("to");
  expect(ui.calls.at(-1).params.target).toBe(1);
});

test("font size is available in browser preview but guarded during IME, work, loading and modals", async () => {
  const preview = await harness(false);
  preview.node("#font-increase").click();
  expect(preview.node("#editor").dataset.fontSize).toBe("20");
  expect(preview.calls).toEqual([]);
  const ui = await harness();
  const increase = ui.node("#font-increase"), editor = ui.node("#editor");
  editor.listeners.compositionstart();
  expect(increase.disabled).toBe(true);
  increase.listeners.click();
  expect(editor.dataset.fontSize).toBe("18");
  editor.listeners.compositionend();
  expect(increase.disabled).toBe(false);
  ui.edit("keep"); await ui.click("new");
  expect(increase.disabled).toBe(true);
  increase.listeners.click();
  expect(editor.dataset.fontSize).toBe("18");
  ui.node("#discard-dialog").close("cancel");
  let finish: (value: unknown) => void = () => {};
  ui.setWait(new Promise(resolve => { finish = resolve; }));
  await ui.click("save");
  expect(increase.disabled).toBe(true);
  increase.listeners.click();
  expect(editor.dataset.fontSize).toBe("18");
  finish({cancelled:true}); await tick();
  expect(increase.disabled).toBe(false);
  const loading = await harness(true, {load:new Promise(() => {})});
  expect(loading.node("#font-increase").disabled).toBe(true);
  const recovery = await harness(true, {draft:{name:"draft.txt",text:"draft"}});
  expect(recovery.node("#font-increase").disabled).toBe(true);
  recovery.node("#font-increase").listeners.click();
  expect(recovery.node("#editor").dataset.fontSize).toBe("18");
  recovery.node("#recovery-dialog").close("recover"); await tick();
  expect(recovery.node("#font-increase").disabled).toBe(false);
});

test("font size survives New/Open and replacement Undo only for the current window", async () => {
  const ui = await harness();
  ui.edit("one one");
  replaceFields(ui, "one", "two");
  ui.node("#replace-all").click();
  ui.node("#word-wrap").click();
  ui.node("#font-increase").click();
  ui.node("#replace-undo").click();
  expect(ui.node("#editor").value).toBe("one one");
  expect(ui.node("#editor").dataset.fontSize).toBe("20");
  expect(ui.node("#editor").wrap).toBe("off");
  await ui.click("save");
  ui.setOpen({cancelled:false,name:"next.txt",text:"next\nline"});
  await ui.click("open");
  expect(ui.node("#editor").dataset.fontSize).toBe("20");
  await ui.click("new");
  expect(ui.node("#editor").dataset.fontSize).toBe("20");
  const fresh = await harness();
  expect(fresh.node("#editor").dataset.fontSize).toBe("18");
});
