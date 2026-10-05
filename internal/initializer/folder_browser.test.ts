import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./folder-browser/app.js", import.meta.url), "utf8");

function harness(invoke?: (method: string, params?: any) => Promise<any>) {
  const nodes = new Map<string, any>();
  function element() {
    const handlers = new Map<string, Function>();
    return {
      textContent: "", value: "", disabled: false, children: [] as any[],
      addEventListener(event: string, fn: Function) { handlers.set(event, fn); },
      click() { return handlers.get("click")?.(); },
      appendChild(child: any) { this.children.push(child); },
      replaceChildren(...children: any[]) { this.children = children; },
    };
  }
  const node = (id: string) => { if (!nodes.has(id)) nodes.set(id, element()); return nodes.get(id); };
  const context: any = { document: { querySelector: node, createElement: element },
    requestAnimationFrame: (fn: Function) => fn() };
  if (invoke) context.velox = { invoke };
  context.window = context;
  runInNewContext(source, context);
  return { node, click: (id: string) => node(`#${id}`).click(),
    file: () => node("#entries").children[0].children[0].children[0] };
}

test("folder starter accesses only folder methods on clicks and renders names literally", async () => {
  const calls: any[] = [];
  const ui = harness(async (method, params) => {
    calls.push({ method, params });
    if (method === "folder.select") return { name: "Selected", target: 7 };
    if (method === "folder.list") return { entries: [{ name: "<img src=x>", kind: "file" }, { name: "child", kind: "directory" }], truncated: true, skipped: 2 };
    if (method === "folder.openText") return { name: "<img src=x>", text: "\ud55c\uae00\n<script>literal</script>", bytes: 30 };
    return null;
  });
  expect(calls).toEqual([]);
  await ui.click("select");
  expect(calls[1]).toEqual({ method: "folder.list", params: { target: 7 } });
  expect(ui.file().textContent).toBe("<img src=x>");
  expect(ui.node("#entries").children[1].children[0].children).toEqual([]);
  expect(ui.node("#status").textContent).toBe("Partial list. 2 entries excluded.");
  await ui.file().click();
  expect(calls[2]).toEqual({ method: "folder.openText", params: { target: 7, name: "<img src=x>" } });
  expect(ui.node("#preview").value).toBe("\ud55c\uae00\n<script>literal</script>");
  expect(ui.node("#file-bytes").textContent).toBe("30 bytes");
  await ui.click("release");
  expect(calls[3]).toEqual({ method: "folder.release", params: { target: 7 } });
  expect(ui.node("#entries").children).toEqual([]);
  expect(ui.node("#preview").value).toBe("");
  expect(ui.node("#refresh").disabled).toBe(true);
});

test("selection cancellation preserves folder, preview and refresh target", async () => {
  let selections = 0;
  const targets: number[] = [];
  const ui = harness(async (method, params) => {
    if (method === "folder.select") return ++selections === 1 ? { name: "Old", target: 3 } : { cancelled: true };
    if (method === "folder.openText") return { name: "notes.txt", text: "keep", bytes: 4 };
    targets.push(params.target);
    return { entries: [{ name: "notes.txt", kind: "file" }], skipped: 0 };
  });
  await ui.click("select");
  await ui.file().click();
  await ui.click("select");
  expect(ui.node("#folder-name").textContent).toBe("Old");
  expect(ui.node("#preview").value).toBe("keep");
  expect(ui.node("#status").textContent).toBe("Selection canceled.");
  await ui.click("refresh");
  expect(targets).toEqual([3, 3]);
  expect(ui.node("#preview").value).toBe("");
});

test("expired target clears both panes without retries; unsupported file clears stale preview only", async () => {
  for (const code of ["FOLDER_TARGET_INVALID", "UNSUPPORTED_FILE", "PERMISSION_DENIED"]) {
    let calls = 0;
    const ui = harness(async (method) => {
      calls++;
      if (method === "folder.select") return { name: "Selected", target: 1 };
      if (method === "folder.list") return { entries: [{ name: "binary.bin", kind: "file" }], skipped: 0 };
      throw Object.assign(new Error("blocked"), { code });
    });
    await ui.click("select");
    ui.node("#preview").value = "stale";
    await ui.file().click();
    expect(calls).toBe(3);
    expect(ui.node("#preview").value).toBe("");
    expect(ui.node("#status").textContent).toBe(`${code}: blocked`);
    expect(ui.node("#refresh").disabled).toBe(code === "FOLDER_TARGET_INVALID");
    expect(ui.node("#select").disabled).toBe(false);
  }
});

test("pending selection and read cannot dispatch duplicate requests", async () => {
  let complete!: (result: any) => void;
  let reads = 0;
  const ui = harness(async (method) => {
    if (method === "folder.select") return { name: "Selected", target: 1 };
    if (method === "folder.list") return { entries: [{ name: "notes.txt", kind: "file" }], skipped: 0 };
    reads++;
    return new Promise(resolve => { complete = resolve; });
  });
  await ui.click("select");
  const file = ui.file();
  const pending = file.click();
  expect(file.disabled).toBe(true);
  await file.click();
  await ui.click("release");
  await ui.click("select");
  expect(reads).toBe(1);
  complete({ name: "notes.txt", text: "", bytes: 0 });
  await pending;
  expect(file.disabled).toBe(false);
  expect(ui.node("#file-bytes").textContent).toBe("0 bytes");

  let calls = 0;
  const selecting = harness(async () => { calls++; return new Promise(resolve => { complete = resolve; }); });
  const selection = selecting.click("select");
  await selecting.click("select");
  expect(calls).toBe(1);
  complete({ cancelled: true });
  await selection;
  expect(selecting.node("#select").disabled).toBe(false);
});

test("replacement invalidates detached rows; failed listing permits explicit refresh", async () => {
  let target = 0, reads = 0, fail = false;
  const ui = harness(async (method) => {
    if (method === "folder.select") return { name: "Selected", target: ++target };
    if (method === "folder.list") {
      if (fail) throw Object.assign(new Error("list failed"), { code: "NATIVE_OPERATION_FAILED" });
      return { entries: [{ name: "notes.txt", kind: "file" }], skipped: 0 };
    }
    if (method === "folder.openText") { reads++; return { name: "notes.txt", text: "hello", bytes: 5 }; }
  });
  await ui.click("select");
  const old = ui.file();
  await old.click();
  fail = true;
  await ui.click("select");
  expect(ui.node("#entries").children).toEqual([]);
  expect(ui.node("#refresh").disabled).toBe(false);
  await old.click();
  expect(reads).toBe(1);
  fail = false;
  await ui.click("refresh");
  expect(ui.node("#entries").children).toHaveLength(1);
});

test("absent native bridge disables selection and never uses a browser picker", async () => {
  const ui = harness();
  await ui.click("select");
  expect(ui.node("#select").disabled).toBe(true);
  expect(ui.node("#status").textContent).toBe("Native folder bridge unavailable.");
  expect(source).not.toContain("clipboard");
  expect(source).not.toContain("showDirectoryPicker");
});
