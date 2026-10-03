import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./web/app.js", import.meta.url), "utf8");

function harness(invoke?: (method: string, params?: unknown) => Promise<unknown>) {
  const nodes = new Map<string, any>();
  function node(id: string) {
    if (!nodes.has(id)) nodes.set(id, { value: "", textContent: "", disabled: false,
      addEventListener(event: string, fn: Function) { this[event] = fn; } });
    return nodes.get(id);
  }
  const context: any = { document: { querySelector: node }, requestAnimationFrame: (fn: Function) => fn() };
  context.window = context;
  if (invoke) context.velox = { invoke };
  runInNewContext(source, context);
  return { node, click: (id: string) => node(`#${id}`).click() };
}

test("clipboard example accesses native methods only on explicit clicks", async () => {
  const calls: any[] = [];
  const text = "\ud55c\uae00\n\ud83d\ude42 <script>not HTML</script>";
  const ui = harness(async (method, params) => { calls.push({ method, params }); return { cancelled: false, text }; });
  expect(calls).toEqual([]);
  ui.node("#source-text").value = text;
  await ui.click("copy-text");
  expect(calls).toEqual([{ method: "clipboard.writeText", params: { text } }]);
  await ui.click("paste-text");
  expect(calls[1]).toEqual({ method: "clipboard.readText", params: undefined });
  expect(ui.node("#result-text").value).toBe(text);
  expect(ui.node("#status").textContent).toBe("Pasted.");
});

test("cancellation, invalid replies and errors retain prior pasted text without private error details", async () => {
  for (const outcome of ["cancel", "invalid", "error"]) {
    const ui = harness(async () => {
      if (outcome === "error") throw Object.assign(new Error("private contents"), { code: "CLIPBOARD_BUSY" });
      return outcome === "cancel" ? { cancelled: true, text: "unexpected" } : { text: "unexpected" };
    });
    ui.node("#result-text").value = "previous";
    await ui.click("paste-text");
    expect(ui.node("#result-text").value).toBe("previous");
    expect(ui.node("#status").textContent).not.toContain("private");
    expect(ui.node("#paste-text").disabled).toBe(false);
  }
});

test("pending confirmation blocks duplicate reads and writes; empty text is valid", async () => {
  let finish: (result: unknown) => void = () => {};
  let calls = 0;
  const ui = harness(async () => { calls++; return await new Promise((resolve) => { finish = resolve; }); });
  const pending = ui.click("paste-text");
  expect(ui.node("#copy-text").disabled).toBe(true);
  await ui.click("paste-text");
  await ui.click("copy-text");
  expect(calls).toBe(1);
  finish({ cancelled: false, text: "" });
  await pending;
  expect(ui.node("#result-text").value).toBe("");
  expect(ui.node("#copy-text").disabled).toBe(false);
});

test("missing native bridge never uses a browser clipboard fallback", async () => {
  const ui = harness();
  ui.node("#result-text").value = "previous";
  await ui.click("paste-text");
  expect(ui.node("#result-text").value).toBe("previous");
  expect(ui.node("#status").textContent).toBe("Native clipboard access is unavailable.");
});
