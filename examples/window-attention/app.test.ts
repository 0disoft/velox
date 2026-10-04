import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./web/app.js", import.meta.url), "utf8");

function harness(invoke?: (method: string, params: any) => Promise<unknown>) {
  const nodes = new Map<string, any>();
  const timers = new Map<number, Function>();
  const events = new Map<string, Function>();
  let next = 0;
  function node(id: string) {
    if (!nodes.has(id)) nodes.set(id, {
      value: "3", textContent: "", disabled: false,
      addEventListener(event: string, fn: Function) { this[event] = fn; },
    });
    return nodes.get(id);
  }
  const context: any = {
    document: { querySelector: node },
    setTimeout(fn: Function) { timers.set(++next, fn); return next; },
    clearTimeout(id: number) { timers.delete(id); },
    addEventListener(event: string, fn: Function) { events.set(event, fn); },
    requestAnimationFrame(fn: Function) { fn(); },
  };
  context.window = context;
  if (invoke) context.velox = { invoke };
  runInNewContext(source, context);
  return {
    node, timers,
    click(id: string) { return node("#" + id).click(); },
    async fire() { for (const [id, fn] of [...timers]) { timers.delete(id); await fn(); } },
    hide() { events.get("pagehide")?.(); },
  };
}

test("attention is explicit, delayed, snapshot-bound and deduplicated while scheduled", async () => {
  const calls: any[] = [];
  const ui = harness(async (method, params) => { calls.push({ method, params }); });
  expect(calls).toEqual([]);
  ui.node("#flash-count").value = "5";
  ui.click("request-attention");
  ui.click("request-attention");
  expect(ui.timers.size).toBe(1);
  expect(calls).toEqual([]);
  expect(ui.node("#status").textContent).toBe("Scheduled.");
  ui.node("#flash-count").value = "1";
  await ui.fire();
  expect(calls).toEqual([{ method: "window.requestAttention", params: { count: 5 } }]);
  expect(ui.node("#status").textContent).toBe("Requested.");
  expect(ui.node("#request-attention").disabled).toBe(false);
});

test("cancel clears a delayed request and explicitly stops native attention", async () => {
  const calls: any[] = [];
  const ui = harness(async (method, params) => { calls.push({ method, params }); });
  ui.click("request-attention");
  await ui.click("cancel-attention");
  await ui.fire();
  expect(ui.timers.size).toBe(0);
  expect(calls).toEqual([{ method: "window.cancelAttention", params: {} }]);
  expect(ui.node("#status").textContent).toBe("Canceled.");
});

test("late request completion cannot override cancel state or unlock pending cancel", async () => {
  let finishRequest!: () => void;
  let finishCancel!: () => void;
  const calls: string[] = [];
  const ui = harness((method) => {
    calls.push(method);
    return new Promise<void>((done) => {
      if (method === "window.requestAttention") finishRequest = done;
      else finishCancel = done;
    });
  });
  ui.node("#delay-seconds").value = "0";
  const requesting = ui.click("request-attention");
  const canceling = ui.click("cancel-attention");
  ui.click("request-attention");
  ui.click("cancel-attention");
  finishRequest();
  await requesting;
  expect(calls).toEqual(["window.requestAttention", "window.cancelAttention"]);
  expect(ui.node("#request-attention").disabled).toBe(true);
  expect(ui.node("#cancel-attention").disabled).toBe(true);
  finishCancel();
  await canceling;
  expect(ui.node("#status").textContent).toBe("Canceled.");
  expect(ui.node("#request-attention").disabled).toBe(false);
});

test("invalid count/delay never reaches native IPC", async () => {
  let calls = 0;
  for (const [count, delay] of [["0", "3"], ["6", "3"], ["", "3"], ["1.5", "3"], ["3", "-1"], ["3", "11"], ["3", ""], ["3", "1.5"]]) {
    const ui = harness(async () => { calls++; });
    ui.node("#flash-count").value = count!;
    ui.node("#delay-seconds").value = delay!;
    await ui.click("request-attention");
    expect(ui.node("#status").textContent).toBe("Invalid count or delay.");
    expect(ui.timers.size).toBe(0);
  }
  expect(calls).toBe(0);
});

test("native failures stay redacted and controls recover; no browser fallback", async () => {
  const ui = harness(async () => { throw Object.assign(new Error("private native details"), { code: "PERMISSION_DENIED" }); });
  ui.node("#delay-seconds").value = "0";
  await ui.click("request-attention");
  expect(ui.node("#status").textContent).toBe("Attention failed: PERMISSION_DENIED");
  expect(ui.node("#request-attention").disabled).toBe(false);
  await ui.click("cancel-attention");
  expect(ui.node("#cancel-attention").disabled).toBe(false);
  const missing = harness();
  await missing.click("request-attention");
  expect(missing.node("#status").textContent).toBe("Native attention is unavailable.");
  expect(missing.timers.size).toBe(0);
});

test("pagehide cancels local scheduling without an automatic native request", async () => {
  let calls = 0;
  const ui = harness(async () => { calls++; });
  ui.click("request-attention");
  ui.hide();
  await ui.fire();
  expect(calls).toBe(0);
  expect(ui.node("#request-attention").disabled).toBe(false);
});
