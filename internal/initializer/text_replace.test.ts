import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./text-editor/find.js", import.meta.url), "utf8");
const context: any = { TextEncoder };
runInNewContext(source, context);
const replace = context.EditorFind.replace;

test("replace all uses literal queries and literal replacement tokens", () => {
  expect(replace("[x] .* [x]", "[x]", "$&$1").text).toBe("$&$1 .* $&$1");
  expect(replace("aaaa", "aa", "b")).toEqual({ text: "bb", count: 2, start: 0, length: 1 });
  expect(replace("aa", "a", "aa")).toEqual({ text: "aaaa", count: 2, start: 0, length: 2 });
  expect(replace("aa", "a", "").text).toBe("");
});

test("single replacement validates original UTF-16 match offsets and case mode", () => {
  const text = "\ud83d\ude42 Text text TEXT";
  expect(replace(text, "text", "\ud55c\uae00", false, 3).text).toBe("\ud83d\ude42 \ud55c\uae00 text TEXT");
  expect(replace(text, "text", "x", true, 3).count).toBe(0);
  expect(replace(text, "text", "x", true, 8).text).toBe("\ud83d\ude42 Text x TEXT");
  expect(replace(text, "text", "x", false, 4).count).toBe(0);
  expect(replace(text, "text", "x", false, -1).count).toBe(0);
  expect(replace("\u212a K k", "k", "x").text).toBe("x x x");
});

test("empty query and absent matches never mutate text", () => {
  for (const query of ["", "missing"]) {
    expect(replace("keep", query, "x")).toEqual({ text: "keep", count: 0, start: -1, length: 0 });
  }
  expect(replace("", "x", "y").count).toBe(0);
});

test("dense replacement stays within UTF-8 limits and refuses oversized expansion", () => {
  const limit = 2 << 20;
  const text = "x".repeat(limit);
  expect(replace(text, "x", "y").count).toBe(limit);
  expect(() => replace(text, "x", "yy")).toThrow("2 MiB");
  expect(() => replace("x".repeat(limit / 2), "x", "\ud55c")).toThrow("2 MiB");
  expect(() => replace("\ud55c".repeat(limit / 2), "\ud55c", "x")).toThrow("2 MiB");
  expect(new TextEncoder().encode(replace("\ud83d\ude42", "\ud83d\ude42", "\ud800").text).byteLength).toBe(3);
});
