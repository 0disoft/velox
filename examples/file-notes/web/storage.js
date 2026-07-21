(function defineFileNotesStorage(root) {
  "use strict";

  const databaseName = "dev.velox.filenotes";
  const storeName = "drafts";
  const currentKey = "current";

  function openDatabase() {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(databaseName, 1);
      request.onupgradeneeded = () => request.result.createObjectStore(storeName);
      request.onerror = () => reject(request.error || new Error("Draft database could not be opened."));
      request.onsuccess = () => resolve(request.result);
    });
  }

  async function run(mode, operation) {
    const database = await openDatabase();
    try {
      return await new Promise((resolve, reject) => {
        const transaction = database.transaction(storeName, mode);
        const request = operation(transaction.objectStore(storeName));
        let result;
        request.onerror = () => reject(request.error || new Error("Draft operation failed."));
        request.onsuccess = () => { result = request.result; };
        transaction.oncomplete = () => resolve(result);
        transaction.onerror = () => reject(transaction.error || new Error("Draft transaction failed."));
        transaction.onabort = () => reject(transaction.error || new Error("Draft transaction was aborted."));
      });
    } finally {
      database.close();
    }
  }

  function load() {
    return run("readonly", (store) => store.get(currentKey));
  }

  async function save(state) {
    try {
      await run("readwrite", (store) => store.put(state, currentKey));
      return true;
    } catch (error) {
      if (state.handle && error.name === "DataCloneError") {
        await run("readwrite", (store) => store.put({ ...state, handle: null }, currentKey));
        return false;
      }
      throw error;
    }
  }

  function clear() {
    return run("readwrite", (store) => store.delete(currentKey));
  }

  root.FileNotesStorage = Object.freeze({ load, save, clear });
})(globalThis);
