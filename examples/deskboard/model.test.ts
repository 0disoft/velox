import { describe, expect, test } from "bun:test";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";

const source = await readFile(new URL("./web/model.js", import.meta.url), "utf8");
const context: Record<string, unknown> = {};
runInNewContext(source, context);
const model = context.DeskboardModel as {
  createInitialState(): any;
  normalizeState(value: unknown): any;
  addTask(state: any, input: unknown, now: string, id: string): any;
  toggleTask(state: any, id: string, now: string): any;
  removeTask(state: any, id: string): any;
  clearCompleted(state: any): any;
  selectTasks(state: any, view: unknown): any[];
  summarize(state: any): { total: number; open: number; done: number; percent: number };
};

describe("DeskboardModel", () => {
  test("normalizes corrupt persisted state without keeping invalid tasks", () => {
    const normalized = model.normalizeState({
      version: 1,
      tasks: [
        { id: "one", title: "  Keep   this  ", priority: "urgent", done: false, createdAt: "bad" },
        { id: "one", title: "duplicate", priority: "high", done: false, createdAt: "2026-01-01T00:00:00.000Z" },
        { id: "", title: "missing id" },
      ],
    });
    expect(normalized.tasks).toHaveLength(1);
    expect(normalized.tasks[0]).toMatchObject({ id: "one", title: "Keep this", priority: "normal", done: false });
  });

  test("adds, completes, filters, and clears tasks immutably", () => {
    const empty = { version: 1, tasks: [] };
    const added = model.addTask(empty, { title: "  Ship   the app ", priority: "high" }, "2026-07-20T10:00:00.000Z", "task-1");
    expect(empty.tasks).toHaveLength(0);
    expect(added.tasks[0].title).toBe("Ship the app");

    const completed = model.toggleTask(added, "task-1", "2026-07-20T10:05:00.000Z");
    expect(model.summarize(completed)).toEqual({ total: 1, open: 0, done: 1, percent: 100 });
    expect(model.selectTasks(completed, { filter: "open", priority: "all", query: "" })).toHaveLength(0);
    expect(model.selectTasks(completed, { filter: "done", priority: "high", query: "ship" })).toHaveLength(1);
    expect(model.clearCompleted(completed).tasks).toHaveLength(0);
  });

  test("rejects blank and duplicate task identities", () => {
    const state = model.createInitialState();
    expect(() => model.addTask(state, { title: "   " }, "2026-07-20T10:00:00.000Z", "new")).toThrow("Enter a task title");
    expect(() => model.addTask(state, { title: "Duplicate" }, "2026-07-20T10:00:00.000Z", "welcome-1")).toThrow("identifier");
  });

  test("removes only the selected task", () => {
    const state = model.createInitialState();
    const next = model.removeTask(state, "welcome-2");
    expect(next.tasks).toHaveLength(state.tasks.length - 1);
    expect(next.tasks.some((task: any) => task.id === "welcome-2")).toBe(false);
  });
});
