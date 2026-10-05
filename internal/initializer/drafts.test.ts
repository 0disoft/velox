import { test, expect } from "bun:test";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";

const source = readFileSync(new URL("./text-editor/drafts.js", import.meta.url), "utf8");
const tick = async () => { for (let i = 0; i < 12; i++) await Promise.resolve(); };
const draft = { schemaVersion: 1, name: "note.txt", text: "\ud55c\uae00", updatedAt: 123 };
function harness(initial: unknown = undefined, abort = false, blocked = false) {
  let record = initial, closes = 0, complete: Function = () => {};
  const database: any = {
    close() { closes++; },
    transaction() {
      const transaction: any = { objectStore() { return {
        get() { return request("get"); },
        put(value: unknown) { return request("put", value); },
        delete() { return request("delete"); },
      }; } };
      function request(method: string, value?: unknown) {
        const req: any = {};
        queueMicrotask(() => {
          req.result = method === "get" ? record : undefined;
          req.onsuccess();
          complete = () => {
            if (abort) { transaction.onabort(); return; }
            if (method === "put") record = structuredClone(value);
            if (method === "delete") record = undefined;
            transaction.oncomplete();
          };
        });
        return req;
      }
      return transaction;
    },
  };
  const indexedDB = { open() {
    const request: any = { result: database };
    queueMicrotask(() => { if (blocked) request.onblocked(); request.onsuccess(); });
    return request;
  } };
  const context: any = { indexedDB, TextEncoder };
  runInNewContext(source, context);
  return { storage: context.EditorDrafts, finish: () => complete(), record: () => record, closes: () => closes };
}

test("save resolves only after commit and excludes native targets and saved baselines", async () => {
  const h = harness();
  let settled = false;
  const saving = h.storage.save({ ...draft, target: 42, savedText: "private baseline" }).then(() => { settled = true; });
  await tick();
  expect(settled).toBe(false);
  expect(h.record()).toBeUndefined();
  h.finish(); await saving;
  expect(h.record()).toEqual(draft);
  expect(h.closes()).toBe(1);
});

test("load and clear wait for completed transactions", async () => {
  const h = harness(draft);
  const loading = h.storage.load(); await tick(); h.finish();
  expect(await loading).toEqual(draft);
  const clearing = h.storage.clear(); await tick(); h.finish(); await clearing;
  expect(h.record()).toBeUndefined();
  expect(h.closes()).toBe(2);
});

test("aborted writes preserve old draft and never report success", async () => {
  const h = harness(draft, true);
  const saving = h.storage.save({ ...draft, text: "replacement" });
  const rejected = saving.catch((error: Error) => error.message);
  await tick(); h.finish();
  expect(await rejected).toBe("Draft storage unavailable.");
  expect(h.record()).toEqual(draft);
  expect(h.closes()).toBe(1);
});

test("blocked open rejects and closes a late successful connection", async () => {
  const h = harness(undefined, false, true);
  await expect(h.storage.load()).rejects.toThrow("Draft storage unavailable.");
  expect(h.closes()).toBe(1);
});

test("invalid, oversized UTF-8 and authority-bearing records are rejected", async () => {
  const h = harness();
  for (const invalid of [
    { ...draft, name: "" }, { ...draft, name: "bad\nname" },
    { ...draft, text: "\ud55c".repeat(700000) }, { ...draft, updatedAt: -1 },
  ]) await expect(h.storage.save(invalid)).rejects.toThrow();
  for (const invalid of [null, { ...draft, schemaVersion: 2 }, { ...draft, target: 42 }]) {
    const corrupt = harness(invalid);
    const loading = corrupt.storage.load();
    const rejected = loading.catch((error: Error) => error.message);
    await tick(); corrupt.finish();
    expect(await rejected).toBe("Stored draft is invalid.");
    expect(corrupt.record()).toEqual(invalid);
  }
});
