(function defineDeskboardModel(root) {
  "use strict";

  const schemaVersion = 1;
  const priorities = new Set(["high", "normal", "low"]);

  function createInitialState() {
    return {
      version: schemaVersion,
      tasks: [
        task("welcome-1", "Package Deskboard with Velox", "high", false, "2026-07-20T09:00:00.000Z"),
        task("welcome-2", "Review the portable output", "normal", false, "2026-07-20T08:00:00.000Z"),
        task("welcome-3", "Confirm local data survives a restart", "low", true, "2026-07-20T07:00:00.000Z", "2026-07-20T07:30:00.000Z"),
      ],
    };
  }

  function task(id, title, priority, done, createdAt, completedAt = null) {
    return { id, title, priority, done, createdAt, completedAt };
  }

  function normalizeState(candidate) {
    if (!candidate || candidate.version !== schemaVersion || !Array.isArray(candidate.tasks)) {
      return createInitialState();
    }

    const seen = new Set();
    const tasks = [];
    for (const candidateTask of candidate.tasks) {
      if (!candidateTask || typeof candidateTask !== "object") continue;
      const id = typeof candidateTask.id === "string" ? candidateTask.id.trim() : "";
      const title = normalizeTitle(candidateTask.title);
      const priority = priorities.has(candidateTask.priority) ? candidateTask.priority : "normal";
      const createdAt = validDate(candidateTask.createdAt) ? candidateTask.createdAt : new Date(0).toISOString();
      if (!id || !title || seen.has(id)) continue;
      seen.add(id);
      const done = candidateTask.done === true;
      tasks.push(task(
        id,
        title,
        priority,
        done,
        createdAt,
        done && validDate(candidateTask.completedAt) ? candidateTask.completedAt : null,
      ));
    }
    return { version: schemaVersion, tasks };
  }

  function addTask(state, input, now, id) {
    const title = normalizeTitle(input?.title);
    if (!title) throw new Error("Enter a task title.");
    if (title.length > 120) throw new Error("Task titles can contain at most 120 characters.");
    if (!id || state.tasks.some((item) => item.id === id)) throw new Error("The task identifier is invalid.");
    const priority = priorities.has(input?.priority) ? input.priority : "normal";
    return {
      version: schemaVersion,
      tasks: [task(id, title, priority, false, now, null), ...state.tasks],
    };
  }

  function toggleTask(state, id, now) {
    return {
      version: schemaVersion,
      tasks: state.tasks.map((item) => item.id === id
        ? { ...item, done: !item.done, completedAt: item.done ? null : now }
        : item),
    };
  }

  function removeTask(state, id) {
    return { version: schemaVersion, tasks: state.tasks.filter((item) => item.id !== id) };
  }

  function clearCompleted(state) {
    return { version: schemaVersion, tasks: state.tasks.filter((item) => !item.done) };
  }

  function selectTasks(state, view) {
    const query = String(view?.query || "").trim().toLocaleLowerCase();
    const filter = ["all", "open", "done"].includes(view?.filter) ? view.filter : "all";
    const priority = priorities.has(view?.priority) ? view.priority : "all";
    return state.tasks.filter((item) => {
      if (filter === "open" && item.done) return false;
      if (filter === "done" && !item.done) return false;
      if (priority !== "all" && item.priority !== priority) return false;
      return !query || item.title.toLocaleLowerCase().includes(query);
    });
  }

  function summarize(state) {
    const done = state.tasks.reduce((count, item) => count + (item.done ? 1 : 0), 0);
    const total = state.tasks.length;
    return { total, done, open: total - done, percent: total === 0 ? 0 : Math.round((done / total) * 100) };
  }

  function normalizeTitle(value) {
    return typeof value === "string" ? value.trim().replace(/\s+/g, " ") : "";
  }

  function validDate(value) {
    return typeof value === "string" && !Number.isNaN(Date.parse(value));
  }

  root.DeskboardModel = Object.freeze({
    schemaVersion,
    createInitialState,
    normalizeState,
    addTask,
    toggleTask,
    removeTask,
    clearCompleted,
    selectTasks,
    summarize,
  });
})(globalThis);
