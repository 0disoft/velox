(() => {
  "use strict";
  const select = document.querySelector("#select");
  const refresh = document.querySelector("#refresh");
  const release = document.querySelector("#release");
  const folder = document.querySelector("#folder-name");
  const count = document.querySelector("#count");
  const entries = document.querySelector("#entries");
  const status = document.querySelector("#status");
  const available = typeof window.velox?.invoke === "function";
  let target = 0;
  let busy = false;

  function controls() {
    select.disabled = busy || !available;
    refresh.disabled = release.disabled = busy || !target;
  }
  function clear() {
    target = 0;
    folder.textContent = "No folder selected";
    entries.replaceChildren();
    count.textContent = "0 items";
  }
  async function list() {
    const result = await window.velox.invoke("folder.list", { target });
    entries.replaceChildren(...result.entries.map((item) => {
      const row = document.createElement("tr");
      for (const value of [item.name, item.kind]) {
        const cell = document.createElement("td");
        cell.textContent = value;
        row.appendChild(cell);
      }
      return row;
    }));
    count.textContent = `${result.entries.length} items`;
    status.textContent = result.truncated ? `Partial list. ${result.skipped} entries excluded.` : `List loaded. ${result.skipped} entries excluded.`;
  }
  async function run(action) {
    if (busy || !available) return;
    busy = true;
    controls();
    try {
      await action();
    } catch (error) {
      if (error.code === "FOLDER_TARGET_INVALID") clear();
      status.textContent = `${error.code || "ERROR"}: ${error.message}`;
    } finally {
      busy = false;
      controls();
    }
  }
  select.addEventListener("click", () => run(async () => {
    const result = await window.velox.invoke("folder.select");
    if (result.cancelled) { status.textContent = "Selection canceled."; return; }
    clear();
    target = result.target;
    folder.textContent = result.name;
    await list();
  }));
  refresh.addEventListener("click", () => target && run(list));
  release.addEventListener("click", () => target && run(async () => {
    await window.velox.invoke("folder.release", { target });
    clear();
    status.textContent = "Folder released.";
  }));
  controls();
  if (!available) status.textContent = "Native folder bridge unavailable.";
  requestAnimationFrame(() => requestAnimationFrame(() => {
    if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
  }));
})();
