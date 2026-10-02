import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./web/app.js", import.meta.url), "utf8");
function harness(invoke: (method: string, params?: any) => Promise<any>) {
  const nodes = new Map<string, any>();
  const events = new Map<string, Function>();
  function element(id = "") {
    const handlers = new Map<string, Function>();
    return {
      textContent: "", value: "", disabled: false, children: [] as any[],
      addEventListener(event: string, fn: Function) { handlers.set(event, fn); events.set(`${id}:${event}`, fn); },
      click() { return handlers.get("click")?.(); },
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
  expect(ui.node("#entries").children[0].children[0].children[0].textContent).toBe("<img src=x>");
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

test("files open only on click, text is literal and directories are not navigable", async () => {
  const calls: any[] = [];
  const ui = harness(async (method, params) => {
    calls.push({ method, params });
    if (method === "folder.select") return { name: "Selected", target: 7 };
    if (method === "folder.list") return { entries: [{ name: "notes.txt", kind: "file" }, { name: "child", kind: "directory" }], skipped: 0 };
    if (method === "folder.openText") return { name: "notes.txt", text: "<script>private</script>", bytes: 24 };
    return null;
  });
  await ui.click("select");
  expect(calls.map((c) => c.method)).toEqual(["folder.select", "folder.list"]);
  expect(ui.node("#entries").children[1].children[0].children.length).toBe(0);
  await ui.node("#entries").children[0].children[0].children[0].click();
  expect(calls[2]).toEqual({ method: "folder.openText", params: { target: 7, name: "notes.txt" } });
  expect(ui.node("#preview").value).toBe("<script>private</script>");
  expect(ui.node("#file-name").textContent).toBe("notes.txt");
  expect(ui.node("#file-bytes").textContent).toBe("24 bytes");
  await ui.click("release");
  expect(ui.node("#preview").value).toBe("");
});

test("cancel preserves preview; refresh or replacement clears it and old rows cannot read", async () => {
  let selections = 0, reads = 0;
  const ui = harness(async (method) => {
    if (method === "folder.select") {
      selections++;
      return selections === 2 ? { cancelled: true } : { name: "Selected", target: selections };
    }
    if (method === "folder.list") return { entries: [{ name: "notes.txt", kind: "file" }], skipped: 0 };
    if (method === "folder.openText") { reads++; return { name: "notes.txt", text: "hello", bytes: 5 }; }
  });
  await ui.click("select");
  const old = ui.node("#entries").children[0].children[0].children[0];
  await old.click();
  await ui.click("select");
  expect(ui.node("#preview").value).toBe("hello");
  await ui.click("refresh");
  expect(ui.node("#preview").value).toBe("");
  await old.click();
  await ui.click("select");
  expect(ui.node("#preview").value).toBe("");
  await old.click();
  expect(reads).toBe(2);
});

test("pending read disables operations and expired read clears both panes without retry", async () => {
  let complete!: (result: any) => void;
  let reads = 0;
  const ui = harness(async (method) => {
    if (method === "folder.select") return { name: "Selected", target: 1 };
    if (method === "folder.list") return { entries: [{ name: "notes.txt", kind: "file" }], skipped: 0 };
    reads++;
    return new Promise((resolve) => { complete = resolve; });
  });
  await ui.click("select");
  const file = ui.node("#entries").children[0].children[0].children[0];
  const pending = file.click();
  expect(file.disabled).toBe(true);
  expect(ui.node("#release").disabled).toBe(true);
  await file.click();
  await ui.click("select");
  expect(reads).toBe(1);
  complete({ name: "notes.txt", text: "hello", bytes: 5 });
  await pending;
  expect(file.disabled).toBe(false);
  expect(ui.node("#preview").value).toBe("hello");

  let calls = 0;
  const expired = harness(async (method) => {
    calls++;
    if (method === "folder.select") return { name: "Selected", target: 1 };
    if (method === "folder.list") return { entries: [{ name: "notes.txt", kind: "file" }], skipped: 0 };
    throw Object.assign(new Error("expired"), { code: "FOLDER_TARGET_INVALID" });
  });
  await expired.click("select");
  await expired.node("#entries").children[0].children[0].children[0].click();
  expect(calls).toBe(3);
  expect(expired.node("#preview").value).toBe("");
  expect(expired.node("#entries").children).toEqual([]);
  expect(expired.node("#refresh").disabled).toBe(true);
});

test("unsupported read clears stale preview but preserves folder for another file", async () => {
  const ui = harness(async (method) => {
    if (method === "folder.select") return { name: "Selected", target: 1 };
    if (method === "folder.list") return { entries: [{ name: "binary.bin", kind: "file" }], skipped: 0 };
    throw Object.assign(new Error("unsupported"), { code: "UNSUPPORTED_FILE" });
  });
  await ui.click("select");
  ui.node("#preview").value = "old text";
  await ui.node("#entries").children[0].children[0].children[0].click();
  expect(ui.node("#preview").value).toBe("");
  expect(ui.node("#status").textContent).toBe("UNSUPPORTED_FILE: unsupported");
  expect(ui.node("#refresh").disabled).toBe(false);
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
