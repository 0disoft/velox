import { expect, test } from "bun:test";
import { harness, tick } from "./text_editor_harness";

function deferred() {
  let resolve!: () => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<void>((ok, fail) => { resolve = ok; reject = fail; });
  return { promise, resolve, reject };
}

test("slow draft storage keeps only the in-flight and latest waiting snapshot", async () => {
  const ui = await harness();
  const storage = deferred();
  ui.setDraftWait(storage.promise);
  for (let revision = 0; revision < 6; revision++) {
    ui.edit(`revision ${revision}`);
    await ui.flushDraft();
  }
  expect(ui.draftCalls.map(call => call.snapshot?.text)).toEqual(["revision 0"]);
  expect(ui.node("#draft-state").textContent).toBe("Saving draft...");
  ui.setDraftWait(null);
  storage.resolve();
  await tick();
  expect(ui.draftCalls.map(call => call.snapshot?.text)).toEqual(["revision 0", "revision 5"]);
  expect(ui.storedDraft().text).toBe("revision 5");
  expect(ui.node("#draft-state").textContent).toBe("Draft stored");
});

test("a failed in-flight draft does not prevent the latest waiting draft", async () => {
  const ui = await harness();
  const storage = deferred();
  ui.setDraftWait(storage.promise);
  for (const text of ["failed", "obsolete", "latest"]) {
    ui.edit(text); await ui.flushDraft();
  }
  ui.setDraftWait(null);
  storage.reject(new Error("storage failed"));
  await tick();
  expect(ui.draftCalls.map(call => call.snapshot?.text)).toEqual(["failed", "latest"]);
  expect(ui.storedDraft().text).toBe("latest");
  expect(ui.node("#draft-state").textContent).toBe("Draft stored");
});

test("save drops obsolete waiting drafts and awaits the clear boundary", async () => {
  const ui = await harness();
  const storage = deferred(), clear = deferred();
  ui.setDraftWait(storage.promise);
  ui.setDraftClearWait(clear.promise);
  for (const text of ["first", "obsolete", "saved"]) {
    ui.edit(text); await ui.flushDraft();
  }
  await ui.click("save");
  expect(ui.calls.at(-1).params.text).toBe("saved");
  expect(ui.node("#editor").readOnly).toBe(true);
  ui.setDraftWait(null);
  storage.resolve();
  await tick();
  expect(ui.draftCalls.map(call => call.method)).toEqual(["save", "clear"]);
  expect(ui.storedDraft().text).toBe("first");
  expect(ui.node("#editor").readOnly).toBe(true);
  expect(ui.node("#draft-state").textContent).toBe("Saving draft...");
  let blocked = false;
  ui.listeners.beforeunload({ preventDefault() { blocked = true; } });
  expect(blocked).toBe(true);
  clear.resolve();
  await tick();
  expect(ui.storedDraft()).toBeNull();
  expect(ui.node("#draft-state").textContent).toBe("No draft");
  expect(ui.node("#editor").readOnly).toBe(false);
  await ui.flushDraft();
  expect(ui.storedDraft()).toBeNull();
  blocked = false;
  ui.listeners.beforeunload({ preventDefault() { blocked = true; } });
  expect(blocked).toBe(false);
});

test("a later snapshot cannot replace or run ahead of a pending clear", async () => {
  const ui = await harness();
  const storage = deferred(), clear = deferred();
  ui.setDraftWait(storage.promise);
  ui.setDraftClearWait(clear.promise);
  ui.edit("old"); await ui.flushDraft();
  await ui.click("save");
  // Programmatic events stress the queue even while native work freezes input.
  ui.edit("later obsolete"); await ui.flushDraft();
  ui.edit("later latest"); await ui.flushDraft();
  ui.setDraftWait(null);
  storage.resolve();
  await tick();
  expect(ui.draftCalls.map(call => call.method)).toEqual(["save", "clear"]);
  expect(ui.storedDraft().text).toBe("old");
  clear.resolve();
  await tick();
  expect(ui.draftCalls.map(call => call.method)).toEqual(["save", "clear", "save"]);
  expect(ui.draftCalls.at(-1).snapshot.text).toBe("later latest");
  expect(ui.storedDraft().text).toBe("later latest");
  expect(ui.node("#draft-state").textContent).toBe("Draft stored");
});

test("failed clear remains visible and does not poison subsequent draft writes", async () => {
  const ui = await harness();
  ui.edit("old"); await ui.flushDraft();
  ui.setDraftFailure(true);
  await ui.click("save");
  expect(ui.storedDraft().text).toBe("old");
  expect(ui.node("#status").textContent).toBe("File saved. Draft cleanup unavailable.");
  expect(ui.node("#draft-state").textContent).toBe("Draft recovery unavailable");
  ui.setDraftFailure(false);
  ui.edit("new"); await ui.flushDraft();
  expect(ui.storedDraft().text).toBe("new");
  expect(ui.node("#draft-state").textContent).toBe("Draft stored");
});

test("repeated pending clears share one awaited clear and discard intervening snapshots", async () => {
  const ui = await harness();
  const storage = deferred(), clear = deferred();
  ui.setDraftWait(storage.promise);
  ui.setDraftClearWait(clear.promise);
  ui.edit("saved"); await ui.flushDraft();
  await ui.click("save");
  for (let i = 0; i < 3; i++) {
    ui.edit(`obsolete ${i}`); await ui.flushDraft();
    ui.edit("saved"); await ui.flushDraft();
  }
  ui.setDraftWait(null);
  storage.resolve(); await tick();
  expect(ui.draftCalls.map(call => call.method)).toEqual(["save", "clear"]);
  expect(ui.node("#editor").readOnly).toBe(true);
  clear.resolve(); await tick();
  expect(ui.draftCalls.map(call => call.method)).toEqual(["save", "clear"]);
  expect(ui.storedDraft()).toBeNull();
  expect(ui.node("#draft-state").textContent).toBe("No draft");
  expect(ui.node("#editor").readOnly).toBe(false);
});
