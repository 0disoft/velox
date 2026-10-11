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
    find: document.querySelector("#find-document"),
  };

  let state = model.createState();
  let pendingAction = null;
  let draftTimer = null;
  let draftWriteRunning = false;
  let pendingDraftWrite = null;
  let draftRevision = 0;
  let fileActionPending = true;
  let composing = false;
  let requestedTitle = null;
  let sourceText = "";
  let sourceValue = "";
  let lineEnding = "\n";
  const finder = window.FileNotesFind.attach(document, elements.editor,
    () => fileActionPending || elements.discardDialog.open);

  function setDocumentText(text) {
    elements.editor.value = text;
    sourceText = text;
    sourceValue = elements.editor.value;
    lineEnding = text.match(/\r\n|\r|\n/)?.[0] ?? "\n";
  }

  function serializeText(value) {
    if (value === sourceValue) return sourceText;
    return lineEnding === "\n" ? value : value.replace(/\n/g, lineEnding);
  }

  function saveSnapshot() {
    const snapshot = { ...state };
    if (snapshot.text.length > 2 * 1024 * 1024 ||
        new TextEncoder().encode(snapshot.text).byteLength > 2 * 1024 * 1024) {
      throw new Error("Text exceeds the 2 MiB UTF-8 limit.");
    }
    return snapshot;
  }

  function setBusy(busy, freezeEditor = false) {
    fileActionPending = busy;
    for (const button of [elements.create, elements.open, elements.save, elements.saveAs]) button.disabled = busy;
    elements.editor.readOnly = busy && freezeEditor;
    elements.find.disabled = busy;
    finder.refresh();
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
    updateNativeTitle(document.title);
    finder.refresh();
  }

  async function updateNativeTitle(title) {
    if (title === requestedTitle || typeof window.velox?.invoke !== "function") return;
    requestedTitle = title;
    try {
      await window.velox.invoke("window.setTitle", { title });
    } catch {
      // A cosmetic caption failure must not interrupt editing or file operations.
    }
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

  async function drainDraftWrites() {
    if (draftWriteRunning) return;
    draftWriteRunning = true;
    try {
      while (pendingDraftWrite) {
        const { snapshot, revision } = pendingDraftWrite;
        pendingDraftWrite = null;
        try {
          await storage.save(snapshot);
          if (revision === draftRevision) elements.draftState.textContent = "Draft saved";
        } catch (error) {
          if (revision !== draftRevision) continue;
          elements.draftState.textContent = "Draft recovery unavailable";
          announce(`Draft save failed: ${error.message}`);
        }
      }
    } finally { draftWriteRunning = false; }
  }

  function queueDraftSave() {
    clearTimeout(draftTimer);
    const revision = ++draftRevision;
    if (composing) return;
    draftTimer = setTimeout(() => {
      draftTimer = null;
      pendingDraftWrite = { snapshot: { ...state }, revision };
      void drainDraftWrites();
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
      setDocumentText(state.text);
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
    sourceText = snapshot.text;
    sourceValue = snapshot.text.replace(/\r\n?/g, "\n");
    render();
    queueDraftSave();
    announce(`${state.name} saved.`);
  }

  async function saveAsDocument() {
    if (typeof window.velox?.saveTextAs !== "function") {
      announce("Native file saving is unavailable.");
      return;
    }
    try {
      const snapshot = saveSnapshot();
      const suggestedName = new TextEncoder().encode(snapshot.name).byteLength > 240 ? "Untitled.md" : snapshot.name;
      finishSave(await window.velox.saveTextAs(snapshot.text, suggestedName), snapshot);
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
      const snapshot = saveSnapshot();
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
    setDocumentText("");
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
    state = model.replaceText(state, serializeText(elements.editor.value), new Date().toISOString());
    render();
    queueDraftSave();
  });
  elements.create.addEventListener("click", () => requestDestructiveAction(() => performFileAction(createDocument, true)));
  elements.open.addEventListener("click", () => requestDestructiveAction(() => performFileAction(openDocument, true)));
  elements.save.addEventListener("click", () => performFileAction(saveDocument));
  elements.saveAs.addEventListener("click", () => performFileAction(saveAsDocument));
  elements.editor.addEventListener("compositionstart", () => {
    composing = true;
    clearTimeout(draftTimer);
    draftTimer = null;
    pendingDraftWrite = null;
    draftRevision++;
  });
  elements.editor.addEventListener("compositionend", () => { composing = false; queueDraftSave(); });
  window.addEventListener("keydown", (event) => {
    if (event.defaultPrevented ||
        composing || event.isComposing || event.keyCode === 229) return;
    if (finder.handleKey(event)) return;
    if (!event.ctrlKey || event.altKey || event.metaKey) return;
    if (event.shiftKey && event.code !== "KeyS") return;
    let button;
    switch (event.code) {
      case "KeyN": button = elements.create; break;
      case "KeyO": button = elements.open; break;
      case "KeyS": button = event.shiftKey ? elements.saveAs : elements.save; break;
      case "KeyF": button = elements.find; break;
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
    if (!fileActionPending && !model.isDirty(state)) return;
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
      setDocumentText(state.text);
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
