import { expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./text-editor/positions.js", import.meta.url), "utf8");
function module(intl: any = Intl) {
  const context: any = { Intl: intl };
  runInNewContext(source, context);
  return context.EditorPosition;
}

test("logical line and column are one-based, including empty and trailing lines", () => {
  const create = module().createIndex;
  expect(create("").at(0)).toEqual({ line: 1, column: 1, selected: 0 });
  const index = create("ab\n\n\txy\n");
  expect(index.at(2)).toEqual({ line: 1, column: 3, selected: 0 });
  expect(index.at(3)).toEqual({ line: 2, column: 1, selected: 0 });
  expect(index.at(5)).toEqual({ line: 3, column: 2, selected: 0 });
  expect(index.at(8)).toEqual({ line: 4, column: 1, selected: 0 });
  expect(index.at(-1, Infinity)).toEqual({ line: 1, column: 1, selected: 0 });
});

test("combining marks, decomposed Korean, flags and ZWJ emoji count as graphemes", () => {
  const samples = ["e\u0301", "\ud55c", "\u1112\u1161\u11ab", "\ud83c\uddf0\ud83c\uddf7", "\ud83d\udc69\u200d\ud83d\udcbb"];
  for (const text of samples) {
    const index = module().createIndex(text);
    expect(index.at(text.length)).toEqual({ line: 1, column: 2, selected: 0 });
    expect(index.at(0, text.length)).toEqual({ line: 1, column: 2, selected: 1 });
    if (text.length > 1) {
      expect(index.at(1)).toEqual({ line: 1, column: 1, selected: 0 });
      expect(index.at(0, 1).selected).toBe(1);
    }
  }
});

test("selection uses active endpoint and counts intersected clusters and newlines", () => {
  const index = module().createIndex("\ud55c\n\ud83d\udc69\u200d\ud83d\udcbbx");
  expect(index.at(0, 8)).toEqual({ line: 2, column: 3, selected: 4 });
  expect(index.at(0, 8, "backward")).toEqual({ line: 1, column: 1, selected: 4 });
  expect(index.at(3, 6)).toEqual({ line: 2, column: 1, selected: 1 });
  expect(module().createIndex("a\r\nb").at(0, 3)).toEqual({ line: 2, column: 1, selected: 2 });
});

test("cursor and selection queries reuse segmentation instead of rebuilding it", () => {
  let calls = 0;
  class Segmenter extends Intl.Segmenter {
    segment(text: string) { calls++; return super.segment(text); }
  }
  const index = module({ Segmenter }).createIndex("\ud55c\n\ud83d\ude42 text");
  for (let i = 0; i < 1000; i++) index.at(i % 9, 9, i % 2 ? "backward" : "forward");
  expect(calls).toBe(1);
});

test("2 MiB ASCII and dense newline positions use the no-segmentation fast path", () => {
  const create = module({}).createIndex;
  const size = 2 << 20;
  expect(create("x".repeat(size)).at(size)).toEqual({ line: 1, column: size + 1, selected: 0 });
  expect(create("\n".repeat(size)).at(size)).toEqual({ line: size + 1, column: 1, selected: 0 });
  expect(() => create("x".repeat(size + 1))).toThrow("document limit");
});
