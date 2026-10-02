import { test, expect } from "bun:test";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";

const source = readFileSync(new URL("../internal/ipc/bridge.js", import.meta.url), "utf8");

function bridge(failAppend = false) {
  const requests: any[] = [];
  let offset = 0;
  const parts: string[] = [];
  const window: any = { __veloxInvoke: async (request: any) => {
    requests.push(request);
    // Include the WebView binding envelope and worst-case escaping, not just inner params.
    expect(new TextEncoder().encode(JSON.stringify({ id: 1, method: "__veloxInvoke", params: [request], session: "document" })).length).toBeLessThanOrEqual(65536);
    let result: any;
    switch (request.method) {
      case "file.beginSave": offset = 0; result = { token: 1 }; break;
      case "file.appendSave":
        if (failAppend) return { v: 1, id: request.id, ok: false, error: { code: "INVALID_PARAMS", message: "bad chunk" } };
        expect(request.params.offset).toBe(offset);
        expect(request.params.text.isWellFormed()).toBe(true);
        parts.push(request.params.text);
        offset += new TextEncoder().encode(request.params.text).length;
        result = { bytes: offset }; break;
      case "file.commitSave": result = { cancelled: false, name: "notes.txt", bytes: offset }; break;
      case "file.commitSaveAs": result = { cancelled: false, name: "notes.txt", bytes: offset, target: 7 }; break;
      case "file.commitSaveTo":
        expect(request.params.target).toBe(7);
        result = { cancelled: false, name: "notes.txt", bytes: offset, target: 7 }; break;
      case "file.cancelSave": result = null; break;
      default: throw new Error(request.method);
    }
    return { v: 1, id: request.id, ok: true, result };
  }};
  window.top = window;
  runInNewContext(source, { window, TextEncoder });
  return { api: window.velox, requests, parts };
}

test("save helper sends 2 MiB without enlarging transport limits", async () => {
  const { api, requests, parts } = bridge();
  const text = "a".repeat(2 * 1024 * 1024);
  const result = await api.saveText(text, "notes.txt");
  expect(parts.join("")).toBe(text);
  expect(result.bytes).toBe(text.length);
  expect(requests.at(-2).method).toBe("file.commitSave");
  expect(Object.isFrozen(api)).toBe(true);
});

test("connected helpers reuse only a native target token", async () => {
  const { api, requests } = bridge();
  const connected = await api.saveTextAs("first", "notes.txt");
  expect(connected.target).toBe(7);
  const saved = await api.saveTextTo("second", connected.target);
  expect(saved.bytes).toBe(6);
  const commits = requests.filter((r) => r.method.startsWith("file.commit"));
  expect(commits.map((r) => r.method)).toEqual(["file.commitSaveAs", "file.commitSaveTo"]);
  expect(commits[1].params).toEqual({ token: 1, target: 7 });
  for (const target of [0, -1, 1.5, 4294967296, "C:/private.txt", null]) {
    const denied = bridge();
    await expect(denied.api.saveTextTo("text", target)).rejects.toThrow();
    expect(denied.requests).toHaveLength(0);
  }
});

test("save chunks preserve Korean, emoji boundaries and escaped controls", async () => {
  for (const text of ["a".repeat(4095) + "\u{1f600}" + "\ud55c\uae00", "\u0001\"\\\n".repeat(5000), ""]) {
    const { api, parts } = bridge();
    await api.saveText(text);
    expect(parts.join("")).toBe(text);
  }
});

test("invalid text is rejected before admission; failed upload is discarded", async () => {
  for (const text of ["\ud800", "a\0b", "a".repeat(2097153), "\ud55c".repeat(699051)]) {
    const { api, requests } = bridge();
    await expect(api.saveText(text)).rejects.toThrow();
    expect(requests).toHaveLength(0);
  }
  const { api, requests } = bridge(true);
  await expect(api.saveText("hello")).rejects.toThrow("bad chunk");
  expect(requests.at(-1).method).toBe("file.cancelSave");
  expect(requests.some((r) => r.method === "file.commitSave")).toBe(false);
});
