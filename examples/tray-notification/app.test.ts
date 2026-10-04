import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const app = await readFile(new URL("./web/app.js", import.meta.url), "utf8");

function harness(invoke?: (method: string, params: any) => Promise<unknown>) {
  const nodes = new Map<string, any>();
  const events = new Map<string, Function>();
  let ready = false, prevented = 0, focused = "";
  function node(selector: string) {
    if (!nodes.has(selector)) nodes.set(selector, {
      value: selector === "#kind" ? "info" : selector === "#message" ? "Velox notification test." : "",
      textContent: "", disabled: false,
      addEventListener(name: string, handler: Function) { events.set(`${selector}:${name}`, handler); },
      focus() { focused = selector; },
    });
    return nodes.get(selector);
  }
  const window = { velox: invoke ? { invoke } : undefined, __veloxReady(phase: string) { expect(phase).toBe("dom-2raf"); ready = true; } };
  runInNewContext(app, { window, document: { querySelector: node }, requestAnimationFrame(fn: Function) { fn(); },
    Notification: class { constructor() { throw Error("Browser notification fallback must not run"); } } });
  return {
    node, get ready() { return ready; }, get focused() { return focused; }, get prevented() { return prevented; },
    change(kind: string) { node("#kind").value = kind; events.get("#kind:change")!(); },
    input(message: string) { node("#message").value = message; events.get("#message:input")!(); },
    submit() { return events.get("#notification-form:submit")!({ preventDefault() { prevented++; } }); },
  };
}

test("only explicit submission invokes native notification with a preserved snapshot", async () => {
  const calls: any[] = [];
  const ui = harness(async (method, params) => { calls.push([method, params]); return null; });
  expect(ui.ready).toBe(true);
  expect(calls).toEqual([]);
  const message = "  \uD55C\uAE00 \uD83D\uDE80\n<tag>  ";
  ui.input(message);
  expect(ui.node("#message-count").value || ui.node("#message-count").textContent).toBe(`${message.length} / 255`);
  for (const kind of ["info", "warning", "error"]) {
    const previous = calls.length;
    ui.change(kind);
    expect(calls.length).toBe(previous);
    await ui.submit();
    expect(calls.at(-1)).toEqual(["notification.show", { kind, message }]);
    expect(ui.node("#status").textContent).toBe("Request accepted.");
    expect(ui.node("#message").value).toBe(message);
    expect(ui.node("#kind").value).toBe(kind);
    expect(ui.focused).toBe("#send-notification");
  }
  expect(ui.prevented).toBe(3);
});

test("invalid input never reaches native IPC; length counts UTF-16 units", async () => {
  const calls: any[] = [];
  const ui = harness(async (method, params) => { calls.push([method, params]); });
  for (const message of ["", " \n\t\u3000", "a\x00b", "a\x7fb", "a\u0085b", "a\rb", "a".repeat(256), "\uD83D\uDE80".repeat(128)]) {
    ui.input(message);
    expect(ui.node("#send-notification").disabled).toBe(true);
    await ui.submit();
    expect(ui.node("#status").textContent).toBe("Enter a message of 1 to 255 UTF-16 units.");
  }
  ui.input("Ready");
  ui.change("other");
  await ui.submit();
  expect(calls).toEqual([]);
  ui.change("info");
  for (const message of ["a".repeat(255), "\uD83D\uDE80".repeat(127) + "a", "\uD55C".repeat(255)]) {
    ui.input(message);
    expect(ui.node("#send-notification").disabled).toBe(false);
    await ui.submit();
    expect(calls.at(-1)).toEqual(["notification.show", { kind: "info", message }]);
  }
});

test("pending submission blocks duplicates and failures unlock controls without raw details", async () => {
  let reject!: (error: unknown) => void;
  let calls = 0;
  const ui = harness(() => { calls++; return new Promise((_, fail) => { reject = fail; }); });
  ui.input("Pending message");
  ui.change("warning");
  const pending = ui.submit();
  for (const selector of ["#kind", "#message", "#send-notification"]) expect(ui.node(selector).disabled).toBe(true);
  await ui.submit();
  expect(calls).toBe(1);
  reject(Object.assign(new Error("private HRESULT details"), { code: "NATIVE_OPERATION_FAILED" }));
  await pending;
  expect(ui.node("#status").textContent).toBe("Notification failed: NATIVE_OPERATION_FAILED");
  for (const selector of ["#kind", "#message", "#send-notification"]) expect(ui.node(selector).disabled).toBe(false);
  expect(ui.node("#message").value).toBe("Pending message");
  expect(ui.node("#kind").value).toBe("warning");
  expect(ui.focused).toBe("#send-notification");
});

test("missing bridge is explicit and has no browser fallback", async () => {
  const ui = harness();
  await ui.submit();
  expect(ui.node("#status").textContent).toBe("Native notifications are unavailable.");
  expect(ui.node("#send-notification").disabled).toBe(false);
  expect(ui.node("#message").value).toBe("Velox notification test.");
});

test("manifest requires tray and grants only notification permission", async () => {
  const manifest = JSON.parse(await readFile(new URL("./velox.json", import.meta.url), "utf8"));
  expect(manifest.app.id).toBe("dev.velox.notification");
  expect(manifest.window.tray).toBe(true);
  expect(manifest.security.permissions).toEqual(["notification.show"]);
});
