import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./web/find.js", import.meta.url), "utf8");
const context: any = {};
runInNewContext(source, context);
const find = context.FileNotesFind.find;

test("literal case-sensitive non-overlapping matches and wraparound", () => {
  expect(find("", "x")).toEqual({ count: 0, index: 0, start: -1 });
  expect(find("text", "")).toEqual({ count: 0, index: 0, start: -1 });
  expect(find("[x] [x]", "[x]")).toEqual({ count: 2, index: 1, start: 0 });
  expect(find("aa aa aa", "aa", 3)).toEqual({ count: 3, index: 3, start: 6 });
  expect(find("aa aa aa", "aa", 6)).toEqual({ count: 3, index: 1, start: 0 });
  expect(find("aa aa aa", "aa", 3, true)).toEqual({ count: 3, index: 1, start: 0 });
  expect(find("aa aa aa", "aa", 0, true)).toEqual({ count: 3, index: 3, start: 6 });
  expect(find("aaaa", "aa")).toEqual({ count: 2, index: 1, start: 0 });
  expect(find("Text text", "text")).toEqual({ count: 1, index: 1, start: 5 });
});

test("UTF-16 positions align with textarea selection for Korean, emoji and newlines", () => {
  const text = "\ud83d\ude42\n\ud55c\uae00\n\ud83d\ude42";
  expect(find(text, "\ud55c\uae00").start).toBe(3);
  expect(find(text, "\ud83d\ude42", 0)).toEqual({ count: 2, index: 2, start: 6 });
  expect(find(text, "\n").count).toBe(2);
});

test("2 MiB dense matches retain only count and one position", () => {
  const text = "x".repeat(2 << 20);
  expect(find(text, "x", text.length - 2)).toEqual({ count: text.length, index: text.length, start: text.length - 1 });
  expect(find(text, "missing")).toEqual({ count: 0, index: 0, start: -1 });
});
