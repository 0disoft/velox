import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./text-editor/app.js", import.meta.url), "utf8");
export const findSource = await readFile(new URL("./text-editor/find.js", import.meta.url), "utf8");
const positionSource = await readFile(new URL("./text-editor/positions.js", import.meta.url), "utf8");
export const tick = async () => { for (let i = 0; i < 12; i++) await Promise.resolve(); };

export async function harness(native = true, options: { draft?: unknown; load?: Promise<unknown> } = {}) {
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
