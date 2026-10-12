import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";

const source = readFileSync(new URL("./web/storage.js", import.meta.url), "utf8");
const tick = () => new Promise(resolve => setImmediate(resolve));

test("oversized draft text or saved baseline fails before touching the stored draft", async () => {
  let opens = 0;
  const context: any = { TextEncoder, indexedDB: { open() { opens++; throw new Error("unexpected open"); } } };
  runInNewContext(source, context);
  const draft = { schemaVersion: 1, name: "note.md", text: "new", savedText: "old", updatedAt: null };
  for (const field of ["text", "savedText"]) {
    for (const oversized of ["x".repeat(2 * 1024 * 1024 + 1), "\uD55C".repeat(699051)]) {
      await expect(context.FileNotesStorage.save({ ...draft, [field]: oversized })).rejects.toThrow("2 MiB UTF-8 limit");
    }
  }
  expect(opens).toBe(0);
});

function harness() {
  let closes = 0;
  let transaction: any;
  const database: any = {
    close() { closes++; },
    transaction() {
      transaction = { objectStore: () => ({ get: () => ({}) }) };
      return transaction;
    },
  };
  const request: any = { result: database };
  const context: any = { indexedDB: { open: () => request } };
  runInNewContext(source, context);
  return { storage: context.FileNotesStorage, request, database, closes: () => closes,
    finish: () => transaction.oncomplete() };
}

test("blocked draft open rejects without waiting for the blocking connection", async () => {
  const h = harness();
  let outcome = "pending";
  const loading = h.storage.load().then(() => { outcome = "loaded"; }, () => { outcome = "unavailable"; });
  h.request.onblocked?.();
  await tick();
  expect(outcome).toBe("unavailable");
  h.request.onsuccess();
  await loading;
  expect(h.closes()).toBe(1);
});

test("an opened draft database releases its connection when another version is requested", async () => {
  const h = harness();
  const loading = h.storage.load();
  h.request.onsuccess();
  await tick();
  h.database.onversionchange?.();
  expect(h.closes()).toBe(1);
  h.finish();
  await loading;
});

function storedDraft(initial: any) {
  let value = initial, writes = 0, closes = 0;
  const modes: string[] = [];
  const database = {
    close() { closes++; },
    transaction(_store: string, mode: string) {
      modes.push(mode);
      let pending = 0, aborted = false;
      const transaction: any = {
        abort() { aborted = true; queueMicrotask(() => transaction.onabort?.()); },
        objectStore: () => ({
          get() { return request(() => value); },
          put(next: any) { writes++; return request(() => { value = next; }); },
          delete() { writes++; return request(() => { value = undefined; }); },
        }),
      };
      function request(operation: () => any) {
        pending++;
        const result: any = {};
        queueMicrotask(() => {
          if (aborted) return;
          result.result = operation();
          result.onsuccess?.();
          pending--;
          if (!pending && !aborted) queueMicrotask(() => transaction.oncomplete?.());
        });
        return result;
      }
      return transaction;
    },
  };
  const context: any = { TextEncoder, indexedDB: { open() {
    const request: any = { result: database };
    queueMicrotask(() => request.onsuccess());
    return request;
  } } };
  runInNewContext(source, context);
  return { storage: context.FileNotesStorage, value: () => value, replace: (next: any) => { value = next; },
    writes: () => writes, closes: () => closes, modes };
}

const replacement = { schemaVersion: 1, name: "new.md", text: "new text", savedText: "", updatedAt: null };

test("unsupported or unreadable drafts are reported without automatic replacement", async () => {
  for (const current of [null, { schemaVersion: 2, text: "future text", extra: { preserve: true } },
    { schemaVersion: 1, text: { corrupt: true } },
    { schemaVersion: 1, text: "x".repeat(2 * 1024 * 1024 + 1) },
    { schemaVersion: 1, text: "small", savedText: "\uD55C".repeat(699051) }]) {
    const h = storedDraft(current);
    await expect(h.storage.load()).rejects.toThrow("Stored draft");
    await expect(h.storage.save(replacement)).rejects.toThrow("Stored draft");
    expect(h.value()).toBe(current);
    expect(h.writes()).toBe(0);
    expect(h.closes()).toBe(2);
  }
});

test("a newer draft appearing after load is rechecked in the write transaction", async () => {
  const h = storedDraft(undefined);
  await h.storage.load();
  const future = { schemaVersion: 2, text: "another writer's text" };
  h.replace(future);
  await expect(h.storage.save(replacement)).rejects.toThrow("Stored draft");
  expect(h.value()).toBe(future);
  expect(h.writes()).toBe(0);
  expect(h.modes).toEqual(["readonly", "readwrite"]);
});

test("empty storage and legacy schema-v1 metadata remain writable", async () => {
  for (const current of [undefined, { schemaVersion: 1, text: "legacy", savedText: 3, handle: {}, target: 42 }]) {
    const h = storedDraft(current);
    await h.storage.save(replacement);
    expect(h.value()).toEqual(replacement);
    expect(h.writes()).toBe(1);
    expect(h.modes).toEqual(["readwrite"]);
  }
});
