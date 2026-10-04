import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const app = await readFile(new URL("./web/app.js", import.meta.url), "utf8");

function harness(invoke?: (method: string, params: any) => Promise<any>) {
  const nodes = new Map<string, any>();
  const events = new Map<string, Function>();
  const selected = { value: "none" };
  function node(selector: string) {
    if (selector.endsWith(":checked")) return selected;
    if (selector.includes('[value="none"]')) return { set checked(value: boolean) { if (value) selected.value = "none"; } };
    if (!nodes.has(selector)) nodes.set(selector, {
      value: selector === "#value" ? "50" : "", disabled: false, textContent: "", dataset: {}, style: {}, attributes: {},
      addEventListener(name: string, handler: Function) { events.set(`${selector}:${name}`, handler); },
      setAttribute(name: string, value: string) { this.attributes[name] = value; },
      removeAttribute(name: string) { delete this.attributes[name]; },
    });
    return nodes.get(selector);
  }
  let ready = false;
  const window = { velox: invoke ? { invoke } : undefined, __veloxReady() { ready = true; } };
  runInNewContext(app, { window, document: { querySelector: node }, requestAnimationFrame(fn: Function) { fn(); } });
  return { node, ready, select(state: string) { selected.value = state; return events.get("#states:change")!(); },
    input(value: number) { node("#value").value = String(value); events.get("#value:input")!(); },
    change() { return events.get("#value:change")!(); }, clear() { return events.get("#clear-progress:click")!(); } };
}

test("states use only progress permission; slider input is local until committed", async () => {
  const calls: any[] = [];
  const ui = harness(async (method, params) => { calls.push([method, params]); return null; });
  expect(ui.ready).toBe(true);
  expect(calls).toEqual([]);
  expect(ui.node("#value").disabled).toBe(true);
  for (const state of ["normal", "error", "paused"]) {
    await ui.select(state);
    expect(calls.at(-1)).toEqual(["window.setProgress", { state, value: 50 }]);
  }
  const previous = calls.length;
  ui.input(75);
  expect(calls.length).toBe(previous);
  expect(ui.node("#progress-bar").attributes["aria-valuenow"]).toBe("75");
  await ui.change();
  expect(calls.at(-1)).toEqual(["window.setProgress", { state: "paused", value: 75 }]);
  await ui.select("indeterminate");
  expect(calls.at(-1)).toEqual(["window.setProgress", { state: "indeterminate" }]);
  expect(ui.node("#progress-bar").attributes["aria-valuenow"]).toBeUndefined();
  ui.clear();
  await new Promise(resolve => setImmediate(resolve));
  expect(calls.at(-1)).toEqual(["window.setProgress", { state: "none" }]);
  expect(ui.node("#indicator").dataset.state).toBe("none");
});

test("pending request prevents duplicate actions and failures unlock controls", async () => {
  let reject!: (error: any) => void;
  let calls = 0;
  const ui = harness(() => { calls++; return new Promise((_, fail) => { reject = fail; }); });
  const pending = ui.select("normal");
  expect(ui.node("#states").disabled).toBe(true);
  expect(ui.node("#clear-progress").disabled).toBe(true);
  ui.clear();
  ui.change();
  expect(calls).toBe(1);
  reject({ code: "NATIVE_OPERATION_FAILED" });
  await pending;
  expect(ui.node("#status").textContent).toBe("Progress failed: NATIVE_OPERATION_FAILED");
  expect(ui.node("#states").disabled).toBe(false);
});

test("missing native bridge is reported without hidden fallback", async () => {
  const ui = harness();
  await ui.select("normal");
  expect(ui.node("#status").textContent).toBe("Native progress is unavailable.");
  expect(ui.node("#states").disabled).toBe(false);
});
