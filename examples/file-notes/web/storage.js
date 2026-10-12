(function defineFileNotesStorage(root) {
  "use strict";

  const databaseName = "dev.velox.filenotes";
  const storeName = "drafts";
  const currentKey = "current";
  const maxBytes = 2 * 1024 * 1024;

  function boundedText(value) {
    return typeof value === "string" && value.length <= maxBytes &&
      new TextEncoder().encode(value).byteLength <= maxBytes;
  }

  function assertReadable(draft) {
    if (draft === undefined) return;
    if (!draft || draft.schemaVersion !== 1 || !boundedText(draft.text) ||
        (typeof draft.savedText === "string" && !boundedText(draft.savedText))) {
      throw Object.assign(new Error("Stored draft format or text is not supported."), { code: "DRAFT_UNREADABLE" });
    }
  }

  function openDatabase() {
    return new Promise((resolve, reject) => {
      let settled = false;
      const request = indexedDB.open(databaseName, 1);
      const fail = () => {
        settled = true;
        reject(new Error("Draft database could not be opened."));
      };
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

  async function run(mode, operation, afterRead) {
    const database = await openDatabase();
    try {
      return await new Promise((resolve, reject) => {
        const transaction = database.transaction(storeName, mode);
        const request = operation(transaction.objectStore(storeName));
        let result;
        let failure;
        request.onerror = () => reject(request.error || new Error("Draft operation failed."));
        request.onsuccess = () => {
          result = request.result;
          try { afterRead?.(transaction.objectStore(storeName), result); }
          catch (error) { failure = error; transaction.abort(); }
        };
        transaction.oncomplete = () => resolve(result);
        transaction.onerror = () => reject(transaction.error || new Error("Draft transaction failed."));
        transaction.onabort = () => reject(failure || transaction.error || new Error("Draft transaction was aborted."));
      });
    } finally {
      database.close();
    }
  }

  function load() {
    return run("readonly", (store) => store.get(currentKey), (_store, draft) => assertReadable(draft));
  }

  async function save(state) {
    const { schemaVersion, name, text, savedText, updatedAt } = state;
    if (schemaVersion !== 1) throw new Error("Draft schema version is not supported.");
    for (const value of [text, savedText]) {
      if (!boundedText(value)) {
        throw new Error("Draft exceeds the 2 MiB UTF-8 limit.");
      }
    }
    // Recheck in the same transaction: another app version may have stored a draft.
    await run("readwrite", (store) => store.get(currentKey), (store, current) => {
      assertReadable(current);
      store.put({ schemaVersion, name, text, savedText, updatedAt }, currentKey);
    });
  }

  function clear() {
    return run("readwrite", (store) => store.delete(currentKey));
  }

  root.FileNotesStorage = Object.freeze({ load, save, clear });
})(globalThis);
