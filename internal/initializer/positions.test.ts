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

function uiHarness(intl: any = Intl) {
  let text = "", reads = 0, composing = false, id = 0;
  const timers = new Map<number, Function>();
  const editor: any = { selectionStart: 0, selectionEnd: 0, selectionDirection: "none", listeners: {},
    get value() { reads++; return text; }, addEventListener(event: string, fn: Function) { this.listeners[event] = fn; } };
  const label = { textContent: "" };
  const document: any = { querySelector: () => label, activeElement: editor, listeners: {},
    addEventListener(event: string, fn: Function) { this.listeners[event] = fn; } };
  const window: any = { setTimeout(fn: Function) { timers.set(++id, fn); return id; }, clearTimeout(id: number) { timers.delete(id); } };
  runInNewContext(source, { window, Intl: intl });
  const position = window.EditorPosition.attach(document, editor, () => composing);
  return { label, editor, timers, position,
    setText(value: string) { text = value; position.update(); },
    setComposing(value: boolean) { composing = value; position.update(); },
    reads: () => reads,
    runOne() { const [id, fn] = timers.entries().next().value!; timers.delete(id); fn(); },
    runAll() { let limit = 1000; while (timers.size && --limit) this.runOne(); expect(limit).toBeGreaterThan(0); },
  };
}

test("selection-only refresh never reads document value or rebuilds index", () => {
  const ui = uiHarness();
  ui.setText("\ud55c\n\ud83d\ude42x");
  const reads = ui.reads();
  ui.editor.selectionStart = 0; ui.editor.selectionEnd = 5;
  ui.editor.listeners.select();
  expect(ui.label.textContent).toBe("Ln 2, Col 3 | Selected 4");
  ui.editor.selectionDirection = "backward";
  ui.editor.listeners.selectionchange();
  expect(ui.label.textContent).toBe("Ln 1, Col 1 | Selected 4");
  for (let i = 0; i < 1000; i++) ui.editor.listeners.keyup();
  expect(ui.reads()).toBe(reads);
});

test("large edits debounce and yield, stale timers cancel and old index survives cancellation", () => {
  const ui = uiHarness();
  ui.setText("\ud55c");
  ui.setText("\ud55c" + "x".repeat(64 << 10));
  expect(ui.label.textContent).toBe("Ln ..., Col ...");
  expect(ui.timers.size).toBe(1);
  ui.runOne();
  expect(ui.label.textContent).toBe("Ln ..., Col ...");
  expect(ui.timers.size).toBe(1);
  ui.setText("\ud55c");
  expect(ui.label.textContent).toBe("Ln 1, Col 1");
  expect(ui.timers.size).toBe(0);
  ui.setText("\ud55c" + "x".repeat(64 << 10));
  ui.runAll();
  ui.editor.selectionEnd = (64 << 10) + 1;
  ui.editor.selectionStart = ui.editor.selectionEnd;
  ui.editor.listeners.select();
  expect(ui.label.textContent).toBe("Ln 1, Col 65,538");
  expect(ui.timers.size).toBe(0);
});

test("IME defers indexing and unavailable index never changes editor text", () => {
  const ui = uiHarness();
  ui.setComposing(true); ui.setText("\u1112\u1161\u11ab");
  expect(ui.label.textContent).toBe("Ln ..., Col ...");
  expect(ui.timers.size).toBe(0);
  ui.setComposing(false);
  ui.editor.selectionStart = ui.editor.selectionEnd = 3;
  ui.editor.listeners.select();
  expect(ui.label.textContent).toBe("Ln 1, Col 2");
  ui.setText("x".repeat((2 << 20) + 1));
  expect(ui.label.textContent).toBe("Position unavailable");
  expect(ui.editor.value.length).toBe((2 << 20) + 1);
  ui.setText("x");
  expect(ui.label.textContent).toBe("Ln 1, Col 2");
  const missing = uiHarness({});
  missing.setText("\ud55c");
  expect(missing.label.textContent).toBe("Position unavailable");
  expect(missing.editor.value).toBe("\ud55c");
});
