import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./text-editor/app.js", import.meta.url), "utf8");
const tick = async () => { for (let i = 0; i < 12; i++) await Promise.resolve(); };

function harness(native = true) {
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
  const window: any = { addEventListener: (event: string, fn: Function) => { listeners[event] = fn; } };
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
  runInNewContext(source, { document, window });
  return {
    node, calls, listeners, document,
    edit(text: string) { node("#editor").value = text; node("#editor").listeners.input(); },
    async click(id: string) { node(`#${id}-document`).click(); await tick(); },
    setOpen: (value: unknown) => { openResult = value; },
    setSave: (value: unknown) => { saveResult = value; },
    setFailure: (value: unknown) => { failure = value; },
    setWait: (value: Promise<unknown> | null) => { wait = value; },
  };
}

test("no startup native access; first save connects and next save reuses target", async () => {
  const ui = harness();
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
  const ui = harness();
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
  const ui = harness();
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
  const ui = harness();
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
  const ui = harness();
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
  const ui = harness(false);
  await ui.click("save");
  await ui.click("open");
  expect(ui.calls).toEqual([]);
  expect(ui.node("#open-document").disabled).toBe(true);
  expect(ui.node("#status").textContent).toBe("Native file access unavailable.");
});
