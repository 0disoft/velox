(function startDeskboard() {
  "use strict";

  const storageKey = "dev.velox.deskboard.tasks.v1";
  const model = window.DeskboardModel;
  const elements = {
    form: document.querySelector("#task-form"),
    title: document.querySelector("#task-title"),
    priority: document.querySelector("#task-priority"),
    formError: document.querySelector("#form-error"),
    search: document.querySelector("#task-search"),
    priorityFilter: document.querySelector("#priority-filter"),
    filters: Array.from(document.querySelectorAll("[data-filter]")),
    list: document.querySelector("#task-list"),
    empty: document.querySelector("#empty-state"),
    emptyTitle: document.querySelector("#empty-title"),
    emptyCopy: document.querySelector("#empty-copy"),
    resultCount: document.querySelector("#result-count"),
    clearCompleted: document.querySelector("#clear-completed"),
    progress: document.querySelector("#progress"),
    progressPercent: document.querySelector("#progress-percent"),
    progressCopy: document.querySelector("#progress-copy"),
    openCount: document.querySelector("#open-count"),
    doneCount: document.querySelector("#done-count"),
    totalCount: document.querySelector("#total-count"),
    date: document.querySelector("#current-date"),
    detailsButton: document.querySelector("#app-details-button"),
    dialog: document.querySelector("#app-dialog"),
    appName: document.querySelector("#app-name"),
    appVersion: document.querySelector("#app-version"),
    appPlatform: document.querySelector("#app-platform"),
    windowState: document.querySelector("#window-state"),
    windowActions: document.querySelector("#window-actions"),
    status: document.querySelector("#status-message"),
  };

  let state = loadState();
  let view = { filter: "all", priority: "all", query: "" };

  function loadState() {
    try {
      const raw = localStorage.getItem(storageKey);
      return raw ? model.normalizeState(JSON.parse(raw)) : model.createInitialState();
    } catch {
      return model.createInitialState();
    }
  }

  function saveState() {
    try {
      localStorage.setItem(storageKey, JSON.stringify(state));
    } catch {
      announce("Changes are available for this session but could not be saved.");
    }
  }

  function commit(nextState, message) {
    state = nextState;
    saveState();
    render();
    if (message) announce(message);
  }

  function render() {
    const tasks = model.selectTasks(state, view);
    const summary = model.summarize(state);
    elements.list.replaceChildren(...tasks.map(renderTask));
    elements.resultCount.textContent = `${tasks.length} ${tasks.length === 1 ? "item" : "items"}`;
    elements.openCount.textContent = String(summary.open);
    elements.doneCount.textContent = String(summary.done);
    elements.totalCount.textContent = String(summary.total);
    elements.progress.value = summary.percent;
    elements.progress.textContent = `${summary.percent}%`;
    elements.progressPercent.textContent = `${summary.percent}%`;
    elements.progressCopy.textContent = summary.total === 0
      ? "No tasks yet"
      : `${summary.done} of ${summary.total} completed`;
    elements.clearCompleted.disabled = summary.done === 0;
    elements.filters.forEach((button) => button.setAttribute("aria-pressed", String(button.dataset.filter === view.filter)));

    const isEmpty = tasks.length === 0;
    elements.empty.hidden = !isEmpty;
    if (isEmpty) {
      const filtering = view.filter !== "all" || view.priority !== "all" || view.query;
      elements.emptyTitle.textContent = filtering ? "No matching tasks." : "Nothing here yet.";
      elements.emptyCopy.textContent = filtering ? "Adjust the current filters." : "Add a task to start the board.";
    }
  }

  function renderTask(item) {
    const row = document.createElement("li");
    row.className = `task-row${item.done ? " is-done" : ""}`;
    row.dataset.taskId = item.id;

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.checked = item.done;
    checkbox.setAttribute("aria-label", `${item.done ? "Reopen" : "Complete"} ${item.title}`);

    const content = document.createElement("div");
    content.className = "task-content";
    const title = document.createElement("span");
    title.className = "task-title";
    title.textContent = item.title;
    const meta = document.createElement("span");
    meta.className = `priority priority-${item.priority}`;
    meta.textContent = `${capitalize(item.priority)} priority`;
    content.append(title, meta);

    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "icon-button remove-button";
    remove.dataset.removeTask = item.id;
    remove.setAttribute("aria-label", `Remove ${item.title}`);
    remove.textContent = "×";

    row.append(checkbox, content, remove);
    return row;
  }

  function capitalize(value) {
    return value.charAt(0).toUpperCase() + value.slice(1);
  }

  function announce(message) {
    elements.status.textContent = "";
    requestAnimationFrame(() => { elements.status.textContent = message; });
  }

  function createID() {
    if (typeof crypto.randomUUID === "function") return crypto.randomUUID();
    return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
  }

  async function invoke(method) {
    if (!window.velox?.invoke) throw new Error("Native controls are available inside Velox.");
    return window.velox.invoke(method, {});
  }

  async function refreshAppInfo() {
    if (!window.velox?.invoke) return;
    try {
      const [info, windowState] = await Promise.all([invoke("app.getInfo"), invoke("window.getState")]);
      elements.appName.textContent = info.name;
      elements.appVersion.textContent = info.version;
      elements.appPlatform.textContent = info.platform;
      elements.windowState.textContent = windowState;
      elements.windowActions.hidden = false;
    } catch (error) {
      elements.windowState.textContent = error.code || "Unavailable";
    }
  }

  function reportReady() {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
      });
    });
  }

  elements.form.addEventListener("submit", (event) => {
    event.preventDefault();
    elements.formError.hidden = true;
    try {
      const next = model.addTask(state, {
        title: elements.title.value,
        priority: elements.priority.value,
      }, new Date().toISOString(), createID());
      commit(next, "Task added.");
      elements.form.reset();
      elements.title.focus();
    } catch (error) {
      elements.formError.textContent = error.message;
      elements.formError.hidden = false;
      elements.title.focus();
    }
  });

  elements.list.addEventListener("change", (event) => {
    const row = event.target.closest("[data-task-id]");
    if (!row || event.target.type !== "checkbox") return;
    commit(model.toggleTask(state, row.dataset.taskId, new Date().toISOString()), event.target.checked ? "Task completed." : "Task reopened.");
  });

  elements.list.addEventListener("click", (event) => {
    const button = event.target.closest("[data-remove-task]");
    if (!button) return;
    commit(model.removeTask(state, button.dataset.removeTask), "Task removed.");
  });

  elements.filters.forEach((button) => button.addEventListener("click", () => {
    view = { ...view, filter: button.dataset.filter };
    render();
  }));
  elements.search.addEventListener("input", () => {
    view = { ...view, query: elements.search.value };
    render();
  });
  elements.priorityFilter.addEventListener("change", () => {
    view = { ...view, priority: elements.priorityFilter.value };
    render();
  });
  elements.clearCompleted.addEventListener("click", () => commit(model.clearCompleted(state), "Completed tasks cleared."));
  elements.detailsButton.addEventListener("click", async () => {
    await refreshAppInfo();
    elements.dialog.showModal();
  });
  elements.windowActions.addEventListener("click", async (event) => {
    const button = event.target.closest("[data-window-action]");
    if (!button) return;
    try {
      await invoke(button.dataset.windowAction);
      if (button.dataset.windowAction !== "window.minimize") await refreshAppInfo();
    } catch (error) {
      announce(error.message);
    }
  });

  elements.date.dateTime = new Date().toISOString().slice(0, 10);
  elements.date.textContent = new Intl.DateTimeFormat(undefined, { weekday: "long", month: "short", day: "numeric" }).format(new Date());
  render();
  refreshAppInfo().finally(reportReady);
})();
