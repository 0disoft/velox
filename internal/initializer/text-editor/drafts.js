(function defineEditorDrafts(root) {
  "use strict";
  const databaseName = "velox.text-editor";
  const storeName = "drafts";
  const currentKey = "current";
  const maxBytes = 2 * 1024 * 1024;

  function valid(draft) {
    return draft && draft.schemaVersion === 1 && typeof draft.name === "string" &&
      draft.name.length > 0 && draft.name.length <= 255 && !/[\u0000-\u001f\u007f]/.test(draft.name) &&
      typeof draft.text === "string" && draft.text.length <= maxBytes &&
      new TextEncoder().encode(draft.text).byteLength <= maxBytes &&
      Number.isSafeInteger(draft.updatedAt) && draft.updatedAt >= 0 && draft.updatedAt <= 8640000000000000 &&
      Object.keys(draft).length === 4;
  }

  function openDatabase() {
    return new Promise((resolve, reject) => {
      let settled = false;
      const request = root.indexedDB.open(databaseName, 1);
      const fail = () => { settled = true; reject(new Error("Draft storage unavailable.")); };
      request.onupgradeneeded = () => request.result.createObjectStore(storeName);
      request.onblocked = fail;
      request.onerror = fail;
      request.onsuccess = () => {
        if (settled) { request.result.close(); return; }
        settled = true;
        request.result.onversionchange = () => request.result.close();
        resolve(request.result);
      };
    });
  }

  async function run(mode, operation) {
    const database = await openDatabase();
    try {
      return await new Promise((resolve, reject) => {
        const transaction = database.transaction(storeName, mode);
        const request = operation(transaction.objectStore(storeName));
        let result;
        const fail = () => reject(new Error("Draft storage unavailable."));
        request.onsuccess = () => { result = request.result; };
        request.onerror = fail;
        transaction.oncomplete = () => resolve(result);
        transaction.onerror = fail;
        transaction.onabort = fail;
      });
    } catch {
      throw new Error("Draft storage unavailable.");
    } finally {
      database.close();
    }
  }

  async function load() {
    const draft = await run("readonly", store => store.get(currentKey));
    if (draft === undefined) return null;
    if (!valid(draft)) throw new Error("Stored draft is invalid.");
    return draft;
  }

  async function save(state) {
    const draft = { schemaVersion: 1, name: state.name, text: state.text, updatedAt: state.updatedAt };
    if (!valid(draft)) throw new Error("Draft is outside the allowed size or format.");
    await run("readwrite", store => store.put(draft, currentKey));
  }

  function clear() {
    return run("readwrite", store => store.delete(currentKey));
  }

  root.EditorDrafts = Object.freeze({ load, save, clear });
})(globalThis);
