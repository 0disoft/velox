import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";

const source = readFileSync(new URL("./web/storage.js", import.meta.url), "utf8");
const tick = () => new Promise(resolve => setImmediate(resolve));

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
