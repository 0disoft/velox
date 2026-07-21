(function startFileNotes() {
  "use strict";

  const maximumFileBytes = 2 * 1024 * 1024;
  const model = window.FileNotesModel;
  const storage = window.FileNotesStorage;
  const elements = {
    name: document.querySelector("#document-name"),
    saveState: document.querySelector("#save-state"),
    editor: document.querySelector("#editor"),
    lines: document.querySelector("#line-count"),
    characters: document.querySelector("#character-count"),
    status: document.querySelector("#status"),
    draftState: document.querySelector("#draft-state"),
    create: document.querySelector("#new-document"),
    open: document.querySelector("#open-document"),
    save: document.querySelector("#save-document"),
    saveAs: document.querySelector("#save-as-document"),
    discardDialog: document.querySelector("#discard-dialog"),
  };

  let state = model.createState();
  let pendingAction = null;
  let draftTimer = null;
  let draftWrites = Promise.resolve();

  function render() {
    const dirty = model.isDirty(state);
    const stats = model.stats(state);
    elements.name.textContent = state.name;
    elements.saveState.textContent = dirty ? "Unsaved changes" : state.handle ? "Saved to file" : "Saved locally";
    elements.lines.textContent = `${stats.lines} ${stats.lines === 1 ? "line" : "lines"}`;
    elements.characters.textContent = `${stats.characters} ${stats.characters === 1 ? "character" : "characters"}`;
    document.title = `${dirty ? "• " : ""}${state.name} · Velox File Notes`;
  }

  function announce(message) {
    elements.status.textContent = message;
  }

  function queueDraftSave() {
    clearTimeout(draftTimer);
    draftTimer = setTimeout(() => {
      const snapshot = { ...state };
      draftWrites = draftWrites.then(async () => {
        const handleStored = await storage.save(snapshot);
        elements.draftState.textContent = handleStored ? "Draft and file handle saved" : "Draft saved without file handle";
      }).catch((error) => {
        elements.draftState.textContent = "Draft recovery unavailable";
        announce(`Draft save failed: ${error.message}`);
      });
    }, 300);
  }

  async function readSelectedFile(handle) {
    const file = await handle.getFile();
    if (file.size > maximumFileBytes) throw new Error("The selected file exceeds 2 MiB.");
    return { file, text: await file.text() };
  }

  async function openDocument() {
    if (typeof window.showOpenFilePicker !== "function") {
      announce("This WebView2 runtime does not expose the file picker.");
      return;
    }
    try {
      const [handle] = await window.showOpenFilePicker({
        multiple: false,
        types: [{ description: "Markdown or text", accept: { "text/plain": [".md", ".markdown", ".txt"] } }],
      });
      const selected = await readSelectedFile(handle);
      state = model.openDocument(state, selected.file.name, selected.text, handle, new Date().toISOString());
      elements.editor.value = state.text;
      render();
      queueDraftSave();
      announce(`${selected.file.name} opened.`);
    } catch (error) {
      announce(error.name === "AbortError" ? "Open canceled." : `Open failed: ${error.message}`);
    }
  }

  async function writeDocument(handle) {
    const permission = await handle.queryPermission({ mode: "readwrite" });
    if (permission !== "granted" && await handle.requestPermission({ mode: "readwrite" }) !== "granted") {
      throw new Error("Write permission was not granted.");
    }
    const writable = await handle.createWritable();
    try {
      await writable.write(state.text);
      await writable.close();
    } catch (error) {
      await writable.abort().catch(() => {});
      throw error;
    }
    state = model.markSaved(state, handle.name, handle, new Date().toISOString());
    render();
    queueDraftSave();
    announce(`${state.name} saved.`);
  }

  async function saveAsDocument() {
    if (typeof window.showSaveFilePicker !== "function") {
      announce("This WebView2 runtime does not expose the save picker.");
      return;
    }
    try {
      const handle = await window.showSaveFilePicker({
        suggestedName: state.name,
        types: [{ description: "Markdown", accept: { "text/markdown": [".md"], "text/plain": [".txt"] } }],
      });
      await writeDocument(handle);
    } catch (error) {
      announce(error.name === "AbortError" ? "Save canceled." : `Save failed: ${error.message}`);
    }
  }

  async function saveDocument() {
    if (!state.handle) {
      await saveAsDocument();
      return;
    }
    try {
      await writeDocument(state.handle);
    } catch (error) {
      announce(`Save failed: ${error.message}`);
    }
  }

  function createDocument() {
    state = model.newDocument();
    elements.editor.value = "";
    render();
    queueDraftSave();
    elements.editor.focus();
    announce("New document created.");
  }

  function requestDestructiveAction(action) {
    if (!model.isDirty(state)) {
      action();
      return;
    }
    pendingAction = action;
    elements.discardDialog.showModal();
  }

  elements.editor.addEventListener("input", () => {
    state = model.replaceText(state, elements.editor.value, new Date().toISOString());
    render();
    queueDraftSave();
  });
  elements.create.addEventListener("click", () => requestDestructiveAction(createDocument));
  elements.open.addEventListener("click", () => requestDestructiveAction(openDocument));
  elements.save.addEventListener("click", saveDocument);
  elements.saveAs.addEventListener("click", saveAsDocument);
  elements.discardDialog.addEventListener("close", () => {
    const action = pendingAction;
    pendingAction = null;
    if (elements.discardDialog.returnValue === "discard" && action) action();
  });
  window.addEventListener("beforeunload", (event) => {
    if (!model.isDirty(state)) return;
    event.preventDefault();
    event.returnValue = "";
  });

  async function restore() {
    if (!("indexedDB" in window)) {
      elements.draftState.textContent = "Draft recovery unavailable";
      render();
      return;
    }
    try {
      state = model.restoreDraft(await storage.load());
      elements.editor.value = state.text;
      if (state.updatedAt) announce(`Draft restored from ${new Date(state.updatedAt).toLocaleString()}.`);
    } catch (error) {
      elements.draftState.textContent = "Draft recovery unavailable";
      announce(`Draft restore failed: ${error.message}`);
    }
    render();
  }

  function reportReady() {
    requestAnimationFrame(() => requestAnimationFrame(() => {
      if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
    }));
  }

  restore().finally(reportReady);
})();
