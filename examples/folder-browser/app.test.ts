import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./web/app.js", import.meta.url), "utf8");
function harness(invoke: (method: string, params?: any) => Promise<any>) {
  const nodes = new Map<string, any>();
  const events = new Map<string, Function>();
  function element(id = "") {
    return {
      textContent: "", disabled: false, children: [] as any[],
      addEventListener(event: string, fn: Function) { events.set(`${id}:${event}`, fn); },
      appendChild(child: any) { this.children.push(child); },
      replaceChildren(...children: any[]) { this.children = children; },
    };
  }
  const node = (id: string) => { if (!nodes.has(id)) nodes.set(id, element(id)); return nodes.get(id); };
  const context: any = { document: { querySelector: node, createElement: () => element() },
    velox: { invoke }, requestAnimationFrame: (fn: Function) => fn() };
  context.window = context;
  runInNewContext(source, context);
  return { node, click: (id: string) => events.get(`#${id}:click`)?.() };
}

test("selection lists by token, renders names as text and releases explicitly", async () => {
  const calls: any[] = [];
  const ui = harness(async (method, params) => {
    calls.push({ method, params });
    if (method === "folder.select") return { name: "Selected", target: 7 };
    if (method === "folder.list") return { entries: [{ name: "<img src=x>", kind: "file" }], truncated: true, skipped: 2 };
    return null;
  });
  await ui.click("select");
  expect(calls[1]).toEqual({ method: "folder.list", params: { target: 7 } });
  expect(ui.node("#entries").children[0].children[0].textContent).toBe("<img src=x>");
  expect(ui.node("#status").textContent).toBe("Partial list. 2 entries excluded.");
  await ui.click("release");
  expect(calls[2]).toEqual({ method: "folder.release", params: { target: 7 } });
  expect(ui.node("#refresh").disabled).toBe(true);
  expect(ui.node("#entries").children).toEqual([]);
});

test("selection cancellation preserves the previous folder and refresh target", async () => {
  let selections = 0;
  const targets: number[] = [];
  const ui = harness(async (method, params) => {
    if (method === "folder.select") return ++selections === 1 ? { name: "Old", target: 3 } : { cancelled: true };
    targets.push(params.target);
    return { entries: [], truncated: false, skipped: 0 };
  });
  await ui.click("select");
  await ui.click("select");
  expect(ui.node("#folder-name").textContent).toBe("Old");
  expect(ui.node("#status").textContent).toBe("Selection canceled.");
  await ui.click("refresh");
  expect(targets).toEqual([3, 3]);
});

test("invalid target clears the view without retrying or reopening selection", async () => {
  let calls = 0;
  const ui = harness(async (method) => {
    calls++;
    if (method === "folder.select") return { name: "Old", target: 1 };
    throw Object.assign(new Error("expired"), { code: "FOLDER_TARGET_INVALID" });
  });
  await ui.click("select");
  expect(calls).toBe(2);
  expect(ui.node("#status").textContent).toBe("FOLDER_TARGET_INVALID: expired");
  expect(ui.node("#refresh").disabled).toBe(true);
  expect(ui.node("#select").disabled).toBe(false);
});

test("pending native selection cannot dispatch another operation", async () => {
  let complete!: (result: any) => void;
  let calls = 0;
  const ui = harness(async () => { calls++; return new Promise((resolve) => { complete = resolve; }); });
  const selecting = ui.click("select");
  expect(ui.node("#select").disabled).toBe(true);
  await ui.click("select");
  expect(calls).toBe(1);
  complete({ cancelled: true });
  await selecting;
  expect(ui.node("#select").disabled).toBe(false);
});
