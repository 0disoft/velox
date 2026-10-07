/// <reference path="../velox.d.ts" />
(function startEditor() {
  "use strict";
  const editor = document.querySelector("#editor");
  const nameLabel = document.querySelector("#document-name");
  const saveState = document.querySelector("#save-state");
  const status = document.querySelector("#status");
  const draftState = document.querySelector("#draft-state");
  const recoveryDialog = document.querySelector("#recovery-dialog");
  const draftName = document.querySelector("#draft-name");
  const dialog = document.querySelector("#discard-dialog");
  const wrapButton = document.querySelector("#word-wrap");
  const fontButtons = ["decrease", "increase", "reset"].map(id => document.querySelector(`#font-${id}`));
  const fontLabel = document.querySelector("#font-size");
  const drafts = window.EditorDrafts;
  const buttons = ["new", "open", "save", "save-as"].map((id) => document.querySelector(`#${id}-document`));
  const appName = document.title;
  let name = "Untitled.txt";
  let savedText = "";
  let target = null;
  let busy = false;
  let composing = false;
  let pendingAction = null;
  let checkingDraft = true;
  let recoveryCandidate = null;
  let recovered = false;
  let draftTimer = null;
  let draftRevision = 0;
  let draftPending = false;
  let draftWrites = Promise.resolve();
  let wordWrap = true;
  let fontSize = 18;
  const positions = window.EditorPosition.attach(document, editor, () => composing);
  const native = typeof window.velox?.invoke === "function" &&
    typeof window.velox?.saveTextAs === "function" && typeof window.velox?.saveTextTo === "function";
  const finder = window.EditorFind.attach(document, editor,
    () => busy || composing || checkingDraft || recoveryCandidate !== null || dialog.open || recoveryDialog.open,
    (message, changed = true) => { status.textContent = message; render(); if (changed) scheduleDraft(); });

  function isDirty() { return recovered || editor.value !== savedText; }

  function viewBlocked() {
    return busy || composing || checkingDraft || recoveryCandidate !== null || dialog.open || recoveryDialog.open;
  }

  function refreshViewControls() {
    const blocked = viewBlocked();
    wrapButton.disabled = blocked;
    fontButtons[0].disabled = blocked || fontSize === 14;
    fontButtons[1].disabled = blocked || fontSize === 28;
    fontButtons[2].disabled = blocked || fontSize === 18;
  }

  function setFontSize(size) {
    if (viewBlocked()) return;
    size = Math.min(28, Math.max(14, size));
    if (size === fontSize) return;
    const top = editor.scrollTop, left = editor.scrollLeft;
    fontSize = size;
    editor.dataset.fontSize = String(size);
    fontLabel.textContent = `${size} px`;
    refreshViewControls();
    editor.focus({ preventScroll: true });
    editor.scrollTop = top;
    editor.scrollLeft = left;
  }

  function render() {
    const dirty = isDirty();
    const blocked = busy || checkingDraft || recoveryCandidate !== null;
    nameLabel.textContent = name;
    saveState.textContent = dirty ? "Unsaved changes" : target ? "Saved to file" : "No save target";
    saveState.dataset.dirty = String(dirty);
    document.title = `${dirty ? "* " : ""}${name} - ${appName}`;
    buttons.forEach((button, index) => { button.disabled = blocked || (index > 0 && !native); });
    editor.readOnly = blocked;
    refreshViewControls();
    finder.refresh();
    positions.update();
  }

  function writeDraft(snapshot, revision) {
    const job = draftWrites.then(() => snapshot === null ? drafts.clear() : drafts.save(snapshot));
    // Keep one serial chain even after failures; never let an old write follow a clear.
    draftWrites = job.catch(() => {});
    job.then(() => {
      if (revision !== draftRevision) return;
      draftPending = false;
      draftState.textContent = snapshot === null ? "No draft" : "Draft stored";
    }, () => {
      if (revision !== draftRevision) return;
      draftPending = false;
      draftState.textContent = "Draft recovery unavailable";
    });
    return job;
  }

  function scheduleDraft() {
    clearTimeout(draftTimer);
    const revision = ++draftRevision;
    draftPending = true;
    draftState.textContent = "Saving draft...";
    draftTimer = setTimeout(() => {
      draftTimer = null;
      const snapshot = isDirty() ? { name, text: editor.value, updatedAt: Date.now() } : null;
      void writeDraft(snapshot, revision).catch(() => {});
    }, 300);
  }

  function clearDraft() {
    clearTimeout(draftTimer);
    draftTimer = null;
    draftPending = true;
    draftState.textContent = "Saving draft...";
    return writeDraft(null, ++draftRevision);
  }

  async function initializeDraft() {
    try {
      recoveryCandidate = await drafts.load();
      if (recoveryCandidate !== null) {
        draftName.textContent = recoveryCandidate.name;
        draftState.textContent = "Draft available";
        recoveryDialog.returnValue = "";
        recoveryDialog.showModal();
      } else draftState.textContent = "No draft";
    } catch {
      recoveryCandidate = null;
      draftState.textContent = "Draft recovery unavailable";
    } finally {
      checkingDraft = false;
      render();
    }
  }

  async function perform(action) {
    if (busy || checkingDraft || recoveryCandidate !== null) return;
    busy = true;
    render();
    try {
      await action();
    } catch (error) {
      if (error.code === "SAVE_TARGET_INVALID") target = null;
      status.textContent = `${error.code || "Operation failed"}: ${error.message}`;
    } finally {
      busy = false;
      render();
      editor.focus();
    }
  }

  async function releaseTarget() {
    if (target !== null) {
      await window.velox.invoke("file.releaseSaveTarget", { target });
      target = null;
    }
  }

  async function newDocument() {
    await releaseTarget();
    let cleared = true;
    try { await clearDraft(); } catch { cleared = false; }
    name = "Untitled.txt";
    editor.value = savedText = "";
    finder.reset();
    recovered = false;
    status.textContent = cleared ? "New document." : "New document. Draft cleanup unavailable.";
  }

  async function openDocument() {
    const result = await window.velox.invoke("file.openText", {});
    if (result.cancelled) { status.textContent = "Open canceled."; return; }
    await releaseTarget();
    let cleared = true;
    try { await clearDraft(); } catch { cleared = false; }
    name = result.name;
    editor.value = savedText = result.text;
    finder.reset();
    recovered = false;
    status.textContent = cleared ? "File opened." : "File opened. Draft cleanup unavailable.";
  }

  async function saveDocument(saveAs) {
    const snapshot = editor.value;
    const result = saveAs || target === null
      ? await window.velox.saveTextAs(snapshot, name)
      : await window.velox.saveTextTo(snapshot, target);
    if (result.cancelled) { status.textContent = "Save canceled."; return; }
    target = result.target;
    name = result.name;
    savedText = snapshot;
    recovered = false;
    status.textContent = "File saved.";
    try { await clearDraft(); }
    catch { status.textContent = "File saved. Draft cleanup unavailable."; }
  }

  function requestReplacement(action) {
    if (busy || checkingDraft || recoveryCandidate !== null || dialog.open) return;
    if (!isDirty()) { void perform(action); return; }
    pendingAction = action;
    dialog.returnValue = "cancel";
    dialog.showModal();
    refreshViewControls();
  }

  buttons[0].addEventListener("click", () => requestReplacement(newDocument));
  buttons[1].addEventListener("click", () => requestReplacement(openDocument));
  buttons[2].addEventListener("click", () => { if (!dialog.open) void perform(() => saveDocument(false)); });
  buttons[3].addEventListener("click", () => { if (!dialog.open) void perform(() => saveDocument(true)); });
  wrapButton.addEventListener("click", () => {
    if (viewBlocked()) return;
    const top = editor.scrollTop, left = editor.scrollLeft;
    wordWrap = !wordWrap;
    editor.wrap = wordWrap ? "soft" : "off";
    wrapButton.setAttribute("aria-pressed", String(wordWrap));
    wrapButton.title = `Word wrap (${wordWrap ? "on" : "off"})`;
    editor.focus({ preventScroll: true });
    editor.scrollTop = top;
    editor.scrollLeft = left;
  });
  fontButtons[0].addEventListener("click", () => setFontSize(fontSize - 2));
  fontButtons[1].addEventListener("click", () => setFontSize(fontSize + 2));
  fontButtons[2].addEventListener("click", () => setFontSize(18));
  editor.addEventListener("input", () => { finder.edited(); render(); if (!composing && !checkingDraft && recoveryCandidate === null) scheduleDraft(); });
  editor.addEventListener("compositionstart", () => {
    composing = true;
    refreshViewControls();
    clearTimeout(draftTimer);
    draftTimer = null;
    draftRevision++;
    draftPending = true;
    draftState.textContent = "Saving draft...";
    finder.refresh();
    positions.update();
  });
  editor.addEventListener("compositionend", () => { composing = false; refreshViewControls(); scheduleDraft(); finder.refresh(); positions.update(); });
  recoveryDialog.addEventListener("cancel", (event) => { event.preventDefault(); });
  recoveryDialog.addEventListener("close", async () => {
    if (recoveryCandidate === null) return;
    if (recoveryDialog.returnValue === "recover") {
      name = recoveryCandidate.name;
      editor.value = recoveryCandidate.text;
      finder.reset();
      savedText = "";
      target = null;
      recovered = true;
      recoveryCandidate = null;
      status.textContent = "Draft restored.";
      scheduleDraft();
    } else if (recoveryDialog.returnValue === "discard") {
      busy = true;
      render();
      try {
        await clearDraft();
        recoveryCandidate = null;
        status.textContent = "Draft discarded.";
      } catch {
        status.textContent = "Draft could not be discarded.";
        recoveryDialog.returnValue = "";
        recoveryDialog.showModal();
      } finally { busy = false; }
    } else recoveryDialog.showModal();
    render();
    if (!recoveryDialog.open) editor.focus();
  });
  dialog.addEventListener("close", () => {
    const action = pendingAction;
    pendingAction = null;
    refreshViewControls();
    if (dialog.returnValue === "discard" && action) void perform(action);
    else editor.focus();
  });
  window.addEventListener("beforeunload", (event) => {
    if (!busy && !isDirty() && !draftPending) return;
    event.preventDefault();
    event.returnValue = "";
  });
  window.addEventListener("keydown", (event) => {
    if (event.defaultPrevented || composing || event.isComposing) return;
    if (finder.handleKey(event)) return;
    if (event.keyCode === 229) return;
    if (!event.ctrlKey || event.altKey || event.metaKey) return;
    if ((event.code === "KeyF" || event.code === "KeyH") && !event.shiftKey) {
      event.preventDefault();
      if (!event.repeat) finder.show(event.code === "KeyH");
      return;
    }
    const index = event.code === "KeyN" && !event.shiftKey ? 0 :
      event.code === "KeyO" && !event.shiftKey ? 1 : event.code === "KeyS" ? event.shiftKey ? 3 : 2 : -1;
    if (index < 0) return;
    event.preventDefault();
    if (!event.repeat && !busy && !dialog.open) buttons[index].click();
  });
  if (!native) status.textContent = "Native file access unavailable.";
  render();
  void initializeDraft();
})();
