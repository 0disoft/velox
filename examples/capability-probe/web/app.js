(function startCapabilityProbe() {
  "use strict";

  const maximumTextBytes = 1024 * 1024;
  const elements = {
    list: document.querySelector("#capability-list"),
    count: document.querySelector("#result-count"),
    summary: document.querySelector("#summary-copy"),
    rerun: document.querySelector("#rerun"),
    runtimeName: document.querySelector("#runtime-name"),
    runtimeVersion: document.querySelector("#runtime-version"),
    copy: document.querySelector("#copy-sample"),
    open: document.querySelector("#open-file"),
    save: document.querySelector("#save-report"),
    drop: document.querySelector("#drop-zone"),
    output: document.querySelector("#interaction-output"),
  };

  let results = [];

  function result(id, name, state, detail) {
    return { id, name, state, detail };
  }

  function availability(value) {
    return value ? "available" : "unavailable";
  }

  async function testLocalStorage() {
    const key = "dev.velox.capability-probe.local-storage";
    try {
      localStorage.setItem(key, "ok");
      const passed = localStorage.getItem(key) === "ok";
      localStorage.removeItem(key);
      return result("local-storage", "Local storage", passed ? "passed" : "failed", passed ? "Write, read, and cleanup succeeded." : "Read-back did not match.");
    } catch (error) {
      return result("local-storage", "Local storage", "blocked", error.message);
    }
  }

  async function testIndexedDB() {
    if (!("indexedDB" in window)) return result("indexed-db", "IndexedDB", "unavailable", "The API is absent.");
    const databaseName = "dev.velox.capability-probe";
    try {
      const value = await new Promise((resolve, reject) => {
        const request = indexedDB.open(databaseName, 1);
        request.onupgradeneeded = () => request.result.createObjectStore("probe");
        request.onerror = () => reject(request.error || new Error("Open failed."));
        request.onsuccess = () => {
          const database = request.result;
          const transaction = database.transaction("probe", "readwrite");
          const store = transaction.objectStore("probe");
          store.put("ok", "status");
          const read = store.get("status");
          read.onerror = () => reject(read.error || new Error("Read failed."));
          read.onsuccess = () => resolve(read.result);
          transaction.oncomplete = () => database.close();
        };
      });
      indexedDB.deleteDatabase(databaseName);
      return result("indexed-db", "IndexedDB", value === "ok" ? "passed" : "failed", value === "ok" ? "Open, write, and read succeeded." : "Read-back did not match.");
    } catch (error) {
      indexedDB.deleteDatabase(databaseName);
      return result("indexed-db", "IndexedDB", "blocked", error.message);
    }
  }

  async function testAppInfo() {
    if (!window.velox?.invoke) return result("app-info", "Velox app identity", "unavailable", "The native bridge is absent.");
    try {
      const info = await window.velox.invoke("app.getInfo", {});
      elements.runtimeName.textContent = info.name;
      elements.runtimeVersion.textContent = `${info.version} · ${info.platform}`;
      return result("app-info", "Velox app identity", "passed", `${info.id} on ${info.platform}`);
    } catch (error) {
      return result("app-info", "Velox app identity", "blocked", error.code || error.message);
    }
  }

  async function runChecks() {
    elements.rerun.disabled = true;
    elements.summary.textContent = "Running non-interactive checks.";
    const staticResults = [
      result("secure-context", "Secure context", isSecureContext ? "passed" : "failed", isSecureContext ? location.origin : "Secure-context APIs may be unavailable."),
      result("open-picker", "Open file picker", availability(typeof window.showOpenFilePicker === "function"), "A user gesture is required to verify operation."),
      result("save-picker", "Save file picker", availability(typeof window.showSaveFilePicker === "function"), "A user gesture is required to verify operation."),
      result("clipboard", "Clipboard write", availability(Boolean(navigator.clipboard?.writeText)), "A user gesture is required to verify operation."),
      result("drag-drop", "File drag and drop", availability("DataTransfer" in window && "FileReader" in window), "Drop operation remains manual."),
      result("notifications", "Notifications", availability("Notification" in window), "Notification permission: " + ("Notification" in window ? Notification.permission : "not exposed")),
    ];
    results = [...staticResults, await testLocalStorage(), await testIndexedDB(), await testAppInfo()];
    renderResults();
    const passed = results.filter((item) => item.state === "passed" || item.state === "available").length;
    elements.summary.textContent = `${passed} of ${results.length} capabilities are available or passed.`;
    elements.rerun.disabled = false;
  }

  function renderResults() {
    elements.list.replaceChildren(...results.map((item) => {
      const row = document.createElement("li");
      row.className = "result-row";
      const copy = document.createElement("div");
      const title = document.createElement("strong");
      title.textContent = item.name;
      const detail = document.createElement("span");
      detail.textContent = item.detail;
      copy.append(title, detail);
      const state = document.createElement("span");
      state.className = `state state-${item.state}`;
      state.textContent = item.state;
      row.append(copy, state);
      return row;
    }));
    elements.count.textContent = `${results.length} checked`;
    elements.open.disabled = typeof window.showOpenFilePicker !== "function";
    elements.save.disabled = typeof window.showSaveFilePicker !== "function";
    elements.copy.disabled = !navigator.clipboard?.writeText;
  }

  async function readTextFile(file) {
    if (file.size > maximumTextBytes) throw new Error("The selected file exceeds 1 MiB.");
    const text = await file.text();
    return `${file.name} · ${file.size} bytes\n${text.slice(0, 500)}`;
  }

  elements.rerun.addEventListener("click", runChecks);
  elements.copy.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText("Velox capability probe");
      elements.output.textContent = "Sample text copied.";
    } catch (error) {
      elements.output.textContent = `Clipboard write failed: ${error.message}`;
    }
  });
  elements.open.addEventListener("click", async () => {
    try {
      const [handle] = await window.showOpenFilePicker({ multiple: false, types: [{ description: "Text", accept: { "text/plain": [".txt", ".md", ".json"] } }] });
      elements.output.textContent = await readTextFile(await handle.getFile());
    } catch (error) {
      elements.output.textContent = error.name === "AbortError" ? "Open canceled." : `Open failed: ${error.message}`;
    }
  });
  elements.save.addEventListener("click", async () => {
    try {
      const handle = await window.showSaveFilePicker({ suggestedName: "velox-capabilities.json", types: [{ description: "JSON", accept: { "application/json": [".json"] } }] });
      const writable = await handle.createWritable();
      await writable.write(`${JSON.stringify({ capturedAt: new Date().toISOString(), results }, null, 2)}\n`);
      await writable.close();
      elements.output.textContent = "Capability report saved.";
    } catch (error) {
      elements.output.textContent = error.name === "AbortError" ? "Save canceled." : `Save failed: ${error.message}`;
    }
  });
  elements.drop.addEventListener("dragover", (event) => {
    event.preventDefault();
    elements.drop.classList.add("is-dragging");
  });
  elements.drop.addEventListener("dragleave", () => elements.drop.classList.remove("is-dragging"));
  elements.drop.addEventListener("drop", async (event) => {
    event.preventDefault();
    elements.drop.classList.remove("is-dragging");
    const file = event.dataTransfer?.files?.[0];
    if (!file) return;
    try {
      elements.output.textContent = await readTextFile(file);
    } catch (error) {
      elements.output.textContent = `Drop failed: ${error.message}`;
    }
  });
  elements.drop.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      elements.open.click();
    }
  });

  requestAnimationFrame(() => requestAnimationFrame(() => {
    if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
  }));
  runChecks();
})();
