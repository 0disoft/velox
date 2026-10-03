(function startFileNotes() {
  "use strict";

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
  let fileActionPending = true;
  let composing = false;

  function setBusy(busy, freezeEditor = false) {
    fileActionPending = busy;
    for (const button of [elements.create, elements.open, elements.save, elements.saveAs]) button.disabled = busy;
    elements.editor.readOnly = busy && freezeEditor;
  }

  async function performFileAction(action, freezeEditor = false) {
    if (fileActionPending) return;
    setBusy(true, freezeEditor);
    try {
      await action();
    } finally {
      setBusy(false);
    }
  }

  function render() {
    const dirty = model.isDirty(state);
    const stats = model.stats(state);
    elements.name.textContent = state.name;
    elements.saveState.textContent = dirty ? "Unsaved changes" : state.target ? "Saved to file" : "No save target";
    elements.lines.textContent = `${stats.lines} ${stats.lines === 1 ? "line" : "lines"}`;
    elements.characters.textContent = `${stats.characters} ${stats.characters === 1 ? "character" : "characters"}`;
    document.title = `${dirty ? "• " : ""}${state.name} · Velox File Notes`;
  }

  function announce(message) {
    elements.status.textContent = message;
  }

  function reportFileError(operation, error) {
    if (error.code === "PERMISSION_DENIED") {
      announce(`${operation} blocked: File access was denied. Your text is still in the editor.`);
    } else if (error.code === "FILE_CHANGED") {
      announce(`${operation} blocked: The file changed outside this app. Your text is still in the editor.`);
    } else if (error.code === "SAVE_TARGET_INVALID") {
      state = { ...state, target: null };
      render();
      announce(`${operation} blocked: The save connection expired. Your text is still in the editor.`);
    } else if (error.code === "SAVE_RECOVERY_REQUIRED") {
      announce(`${operation} needs recovery: ${error.message} Your text is still in the editor.`);
    } else {
      announce(`${operation} failed: ${error.message}`);
    }
  }

  function queueDraftSave() {
    clearTimeout(draftTimer);
    draftTimer = setTimeout(() => {
      const snapshot = { ...state };
      draftWrites = draftWrites.then(async () => {
        await storage.save(snapshot);
        elements.draftState.textContent = "Draft saved";
      }).catch((error) => {
        elements.draftState.textContent = "Draft recovery unavailable";
        announce(`Draft save failed: ${error.message}`);
      });
    }, 300);
  }

  async function releaseSaveTarget() {
    if (!state.target) return;
    await window.velox.invoke("file.releaseSaveTarget", { target: state.target });
    state = { ...state, target: null };
  }

  async function openDocument() {
    if (typeof window.velox?.invoke !== "function") {
      announce("Native file opening is unavailable.");
      return;
    }
    try {
      const selected = await window.velox.invoke("file.openText");
      if (selected.cancelled) { announce("Open canceled."); return; }
      await releaseSaveTarget();
      state = model.openDocument(state, selected.name, selected.text, new Date().toISOString());
      elements.editor.value = state.text;
      render();
      queueDraftSave();
      announce(`${selected.name} opened.`);
    } catch (error) {
      reportFileError("Open", error);
    }
  }

  function finishSave(result, snapshot) {
    if (result.cancelled) { announce("Save canceled."); return; }
    // Editing may continue during a write; only the submitted text reached disk.
    state = { ...model.markSaved(snapshot, result.name, result.target, new Date().toISOString()), text: state.text };
    render();
    queueDraftSave();
    announce(`${state.name} saved.`);
  }

  async function saveAsDocument() {
    const snapshot = { ...state };
    if (typeof window.velox?.saveTextAs !== "function") {
      announce("Native file saving is unavailable.");
      return;
    }
    try {
      finishSave(await window.velox.saveTextAs(snapshot.text, snapshot.name), snapshot);
    } catch (error) {
      reportFileError("Save", error);
    }
  }

  async function saveDocument() {
    if (!state.target) {
      await saveAsDocument();
      return;
    }
    try {
      const snapshot = { ...state };
      finishSave(await window.velox.saveTextTo(snapshot.text, snapshot.target), snapshot);
    } catch (error) {
      reportFileError("Save", error);
    }
  }

  async function createDocument() {
    try {
      await releaseSaveTarget();
    } catch (error) {
      reportFileError("New", error);
      return;
    }
    state = model.newDocument();
    elements.editor.value = "";
    render();
    queueDraftSave();
    elements.editor.focus();
    announce("New document created.");
  }

  function requestDestructiveAction(action) {
    if (fileActionPending || elements.discardDialog.open) return;
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
  elements.create.addEventListener("click", () => requestDestructiveAction(() => performFileAction(createDocument, true)));
  elements.open.addEventListener("click", () => requestDestructiveAction(() => performFileAction(openDocument, true)));
  elements.save.addEventListener("click", () => performFileAction(saveDocument));
  elements.saveAs.addEventListener("click", () => performFileAction(saveAsDocument));
  elements.editor.addEventListener("compositionstart", () => { composing = true; });
  elements.editor.addEventListener("compositionend", () => { composing = false; });
  window.addEventListener("keydown", (event) => {
    if (event.defaultPrevented || !event.ctrlKey || event.altKey || event.metaKey ||
        composing || event.isComposing || event.keyCode === 229) return;
    if (event.shiftKey && event.code !== "KeyS") return;
    let button;
    switch (event.code) {
      case "KeyN": button = elements.create; break;
      case "KeyO": button = elements.open; break;
      case "KeyS": button = event.shiftKey ? elements.saveAs : elements.save; break;
      default: return;
    }
    event.preventDefault();
    if (event.repeat || fileActionPending || elements.discardDialog.open) return;
    button.click();
  });
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
    setBusy(false);
    requestAnimationFrame(() => requestAnimationFrame(() => {
      if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
    }));
  }

  setBusy(true, true);
  restore().finally(reportReady);
})();
