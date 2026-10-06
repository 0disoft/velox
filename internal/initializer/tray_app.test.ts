import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./tray-app/app.js", import.meta.url), "utf8");

function harness(invoke?: (method: string, params: any) => Promise<unknown>) {
  const nodes = new Map<string, any>();
  const events = new Map<string, Function>();
  let focused = "";
  function node(selector: string) {
    if (!nodes.has(selector)) nodes.set(selector, {
      value: selector === "#kind" ? "info" : "", textContent: "", disabled: false,
      addEventListener(name: string, handler: Function) { events.set(`${selector}:${name}`, handler); },
      focus() { focused = selector; },
    });
    return nodes.get(selector);
  }
  const document = { querySelector: node, documentElement: { dataset: {} } };
  runInNewContext(source, { document, window: { velox: invoke ? { invoke } : undefined },
    Notification: class { constructor() { throw Error("Browser fallback must not run"); } } });
  return { node, document, get focused() { return focused; },
    input(text: string) { node("#message").value = text; events.get("#message:input")!(); },
    kind(value: string) { node("#kind").value = value; events.get("#kind:change")!(); },
    submit() { return events.get("#notification-form:submit")!({ preventDefault() {} }); },
  };
}

test("tray starter only sends explicitly submitted text and preserves it", async () => {
  const calls: any[] = [];
  const ui = harness(async (method, params) => { calls.push([method, params]); return null; });
  expect(calls).toEqual([]);
  expect(ui.node("#send-notification").disabled).toBe(true);
  expect(ui.document.documentElement.dataset).toEqual({ velox: "ready" });
  const text = "\ud55c\uae00\n<tag>";
  ui.input(text);
  for (const kind of ["info", "warning", "error"]) {
    const previous = calls.length;
    ui.kind(kind);
    expect(calls.length).toBe(previous);
    await ui.submit();
    expect(calls.at(-1)).toEqual(["notification.show", { kind, message: text }]);
    expect(ui.node("#message").value).toBe(text);
    expect(ui.node("#status").textContent).toBe("Request accepted.");
    expect(ui.focused).toBe("#send-notification");
  }
});

test("empty, oversized, control-containing and invalid-kind input never reaches IPC", async () => {
  const calls: any[] = [];
  const ui = harness(async (...call) => { calls.push(call); });
  for (const text of ["", " \n\t", "a\x00b", "a\rb", "a\x7fb", "a\u0085b", "a".repeat(256), "\ud83d\ude80".repeat(128)]) {
    ui.input(text);
    expect(ui.node("#send-notification").disabled).toBe(true);
    await ui.submit();
  }
  ui.input("valid");
  ui.kind("other");
  await ui.submit();
  expect(calls).toEqual([]);
  ui.kind("info");
  ui.input("\ud83d\ude80".repeat(127) + "a");
  expect(ui.node("#message-count").textContent).toBe("255 / 255");
  await ui.submit();
  expect(calls).toHaveLength(1);
});

test("pending requests block duplicates and failure restores controls without raw errors", async () => {
  let reject!: (error: unknown) => void;
  let calls = 0;
  const ui = harness(() => { calls++; return new Promise((_, fail) => { reject = fail; }); });
  ui.input("keep");
  const pending = ui.submit();
  for (const selector of ["#kind", "#message", "#send-notification"]) expect(ui.node(selector).disabled).toBe(true);
  await ui.submit();
  expect(calls).toBe(1);
  reject(Object.assign(new Error("private details"), { code: "NATIVE_OPERATION_FAILED" }));
  await pending;
  expect(ui.node("#status").textContent).toBe("Notification failed: NATIVE_OPERATION_FAILED");
  expect(ui.node("#message").value).toBe("keep");
  for (const selector of ["#kind", "#message", "#send-notification"]) expect(ui.node(selector).disabled).toBe(false);
});

test("missing bridge is explicit with no browser notification fallback", async () => {
  const ui = harness();
  ui.input("message");
  await ui.submit();
  expect(ui.node("#status").textContent).toBe("Native notifications are unavailable.");
  expect(source).not.toContain("new Notification");
  expect(source).not.toContain("setInterval");
});
