import { expect, test } from "bun:test";
import { harness } from "./text_editor_harness";

const key = "velox.text-editor.view.v1";
function storage(raw: string | null = null) {
  const values = new Map<string, string>(raw === null ? [] : [[key, raw]]);
  const writes: string[] = [];
  let fail = false;
  return {
    values, writes,
    getItem: (name: string) => values.get(name) ?? null,
    setItem(name: string, value: string) {
      if (fail) throw new Error("quota exceeded");
      values.set(name, value); writes.push(value);
    },
    failWrites: (value: boolean) => { fail = value; },
  };
}

test("view defaults do not write storage or draft/native state", async () => {
  const viewStorage = storage();
  const ui = await harness(true, { viewStorage });
  expect(ui.node("#editor").dataset.fontSize).toBe("18");
  expect(ui.node("#editor").wrap).toBe("soft");
  expect(ui.node("#word-wrap")["aria-pressed"]).toBe("true");
  expect(ui.node("#font-reset").disabled).toBe(true);
  expect(viewStorage.writes).toEqual([]);
  expect(ui.calls).toEqual([]);
  expect(ui.draftCalls).toEqual([]);
});

test("view changes survive restart separately from document content and authority", async () => {
  const viewStorage = storage();
  const ui = await harness(true, { viewStorage });
  ui.edit("private text");
  ui.node("#font-increase").click();
  ui.node("#word-wrap").click();
  expect(JSON.parse(viewStorage.values.get(key)!)).toEqual({ schemaVersion: 1, fontSize: 20, wordWrap: false });
  const next = await harness(true, { viewStorage });
  expect(next.node("#editor").dataset.fontSize).toBe("20");
  expect(next.node("#font-size").textContent).toBe("20 px");
  expect(next.node("#editor").wrap).toBe("off");
  expect(next.node("#word-wrap")["aria-pressed"]).toBe("false");
  expect(next.node("#word-wrap").title).toBe("Word wrap (off)");
  expect(next.node("#editor").value).toBe("");
  expect(next.calls).toEqual([]);
  expect(next.draftCalls).toEqual([]);
  next.node("#font-reset").click();
  const reset = await harness(true, { viewStorage });
  expect(reset.node("#editor").dataset.fontSize).toBe("18");
  expect(reset.node("#editor").wrap).toBe("off");
});

for (const raw of ["{", "null", "[]", "x".repeat(257),
  ...[{ schemaVersion: 2, fontSize: 20, wordWrap: false },
    { schemaVersion: 1, fontSize: 15, wordWrap: true },
    { schemaVersion: 1, fontSize: 30, wordWrap: true },
    { schemaVersion: 1, fontSize: "20", wordWrap: false },
    { schemaVersion: 1, fontSize: 20, wordWrap: "false" },
    { schemaVersion: 1, fontSize: 20, wordWrap: true, text: "private" },
  ].map(value => JSON.stringify(value))]) {
  test(`unreadable view settings remain intact: ${raw.slice(0, 65)}`, async () => {
    const viewStorage = storage(raw);
    const ui = await harness(true, { viewStorage });
    expect(ui.node("#editor").dataset.fontSize).toBe("18");
    expect(ui.node("#editor").wrap).toBe("soft");
    expect(ui.node("#status").textContent).toContain("View settings unavailable");
    ui.node("#font-increase").click();
    ui.node("#word-wrap").click();
    expect(ui.node("#editor").dataset.fontSize).toBe("20");
    expect(ui.node("#editor").wrap).toBe("off");
    expect(viewStorage.values.get(key)).toBe(raw);
    expect(viewStorage.writes).toEqual([]);
    expect(ui.node("#status").textContent).toContain("this window");
    ui.edit("still editable");
    await ui.click("save");
    expect(ui.calls[0].params.text).toBe("still editable");
  });
}

test("denied storage getter still allows editing and browser preview", async () => {
  const ui = await harness(false, { viewStorageUnavailable: true });
  expect(ui.node("#status").textContent).toContain("Native file access unavailable");
  expect(ui.node("#status").textContent).toContain("View settings unavailable");
  ui.node("#word-wrap").click();
  ui.node("#font-increase").click();
  ui.edit("typing");
  expect(ui.node("#editor").value).toBe("typing");
  expect(ui.node("#editor").wrap).toBe("off");
  expect(ui.node("#editor").dataset.fontSize).toBe("20");
});

test("failed settings write permits a later retry without losing document state", async () => {
  const viewStorage = storage();
  const ui = await harness(true, { viewStorage });
  viewStorage.failWrites(true);
  ui.edit("keep");
  ui.node("#font-increase").click();
  expect(ui.node("#editor").dataset.fontSize).toBe("20");
  expect(ui.node("#status").textContent).toContain("this window");
  expect(viewStorage.writes).toEqual([]);
  viewStorage.failWrites(false);
  ui.node("#font-increase").click();
  expect(JSON.parse(viewStorage.values.get(key)!).fontSize).toBe(22);
  expect(ui.node("#status").textContent).toBe("View settings saved.");
  expect(ui.node("#editor").value).toBe("keep");
  expect(ui.node("#save-state").dataset.dirty).toBe("true");
});

test("another window changing a different field is not overwritten by stale view state", async () => {
  const viewStorage = storage();
  const first = await harness(true, { viewStorage });
  const second = await harness(true, { viewStorage });
  first.node("#font-increase").click();
  second.node("#word-wrap").click();
  expect(JSON.parse(viewStorage.values.get(key)!)).toEqual({ schemaVersion: 1, fontSize: 20, wordWrap: false });
  const next = await harness(true, { viewStorage });
  expect(next.node("#editor").dataset.fontSize).toBe("20");
  expect(next.node("#editor").wrap).toBe("off");
});

test("a late future-format record is protected before any setting write", async () => {
  const viewStorage = storage();
  const ui = await harness(true, { viewStorage });
  const future = JSON.stringify({ schemaVersion: 2, fontSize: 26, wordWrap: false });
  viewStorage.values.set(key, future);
  ui.node("#font-increase").click();
  expect(viewStorage.values.get(key)).toBe(future);
  expect(viewStorage.writes).toEqual([]);
  expect(ui.node("#editor").dataset.fontSize).toBe("20");
});
