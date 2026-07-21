import { describe, expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./web/model.js", import.meta.url), "utf8");
const context: Record<string, unknown> = {};
runInNewContext(source, context);
const model = context.FileNotesModel as {
  createState(): any;
  restoreDraft(candidate: unknown): any;
  replaceText(state: any, text: string, updatedAt: string): any;
  openDocument(state: any, name: string, text: string, handle: unknown, updatedAt: string): any;
  markSaved(state: any, name: string, handle: unknown, updatedAt: string): any;
  newDocument(): any;
  isDirty(state: any): boolean;
  stats(state: any): { lines: number; characters: number };
};

describe("FileNotesModel", () => {
  test("derives dirty state from document and saved text", () => {
    const initial = model.createState();
    const edited = model.replaceText(initial, "# Note", "2026-07-21T01:00:00.000Z");
    expect(model.isDirty(initial)).toBe(false);
    expect(model.isDirty(edited)).toBe(true);
    expect(model.markSaved(edited, "note.md", null, "2026-07-21T01:01:00.000Z").savedText).toBe("# Note");
  });

  test("opens a selected document as the saved baseline", () => {
    const handle = { name: "note.md" };
    const opened = model.openDocument(model.createState(), "note.md", "hello", handle, "2026-07-21T01:00:00.000Z");
    expect(opened).toMatchObject({ name: "note.md", text: "hello", savedText: "hello", handle });
    expect(model.isDirty(opened)).toBe(false);
  });

  test("normalizes malformed draft metadata while retaining valid text", () => {
    const restored = model.restoreDraft({ schemaVersion: 1, name: "", text: "draft", savedText: 3, updatedAt: "bad" });
    expect(restored).toMatchObject({ name: "Untitled.md", text: "draft", savedText: "", updatedAt: null });
  });

  test("counts Unicode characters and mixed newlines", () => {
    expect(model.stats(model.replaceText(model.createState(), "A😀\r\nB\nC", "2026-07-21T01:00:00.000Z"))).toEqual({ lines: 3, characters: 6 });
  });
});
