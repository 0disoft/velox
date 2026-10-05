/// <reference path="../velox.d.ts" />
(function startEditor() {
  "use strict";
  const editor = document.querySelector("#editor");
  const nameLabel = document.querySelector("#document-name");
  const saveState = document.querySelector("#save-state");
  const status = document.querySelector("#status");
  const dialog = document.querySelector("#discard-dialog");
  const buttons = ["new", "open", "save", "save-as"].map((id) => document.querySelector(`#${id}-document`));
  const appName = document.title;
  let name = "Untitled.txt";
  let savedText = "";
  let target = null;
  let busy = false;
  let composing = false;
  let pendingAction = null;
  const native = typeof window.velox?.invoke === "function" &&
    typeof window.velox?.saveTextAs === "function" && typeof window.velox?.saveTextTo === "function";

  function render() {
    const dirty = editor.value !== savedText;
    nameLabel.textContent = name;
    saveState.textContent = dirty ? "Unsaved changes" : target ? "Saved to file" : "No save target";
    saveState.dataset.dirty = String(dirty);
    document.title = `${dirty ? "* " : ""}${name} - ${appName}`;
    buttons.forEach((button, index) => { button.disabled = busy || (index > 0 && !native); });
    editor.readOnly = busy;
  }

  async function perform(action) {
    if (busy) return;
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
    name = "Untitled.txt";
    editor.value = savedText = "";
    status.textContent = "New document.";
  }

  async function openDocument() {
    const result = await window.velox.invoke("file.openText", {});
    if (result.cancelled) { status.textContent = "Open canceled."; return; }
    await releaseTarget();
    name = result.name;
    editor.value = savedText = result.text;
    status.textContent = "File opened.";
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
    status.textContent = "File saved.";
  }

  function requestReplacement(action) {
    if (busy || dialog.open) return;
    if (editor.value === savedText) { void perform(action); return; }
    pendingAction = action;
    dialog.returnValue = "cancel";
    dialog.showModal();
  }

  buttons[0].addEventListener("click", () => requestReplacement(newDocument));
  buttons[1].addEventListener("click", () => requestReplacement(openDocument));
  buttons[2].addEventListener("click", () => { if (!dialog.open) void perform(() => saveDocument(false)); });
  buttons[3].addEventListener("click", () => { if (!dialog.open) void perform(() => saveDocument(true)); });
  editor.addEventListener("input", render);
  editor.addEventListener("compositionstart", () => { composing = true; });
  editor.addEventListener("compositionend", () => { composing = false; });
  dialog.addEventListener("close", () => {
    const action = pendingAction;
    pendingAction = null;
    if (dialog.returnValue === "discard" && action) void perform(action);
    else editor.focus();
  });
  window.addEventListener("beforeunload", (event) => {
    if (!busy && editor.value === savedText) return;
    event.preventDefault();
    event.returnValue = "";
  });
  window.addEventListener("keydown", (event) => {
    if (event.defaultPrevented || composing || event.isComposing || event.keyCode === 229 ||
        !event.ctrlKey || event.altKey || event.metaKey) return;
    const index = event.code === "KeyN" && !event.shiftKey ? 0 :
      event.code === "KeyO" && !event.shiftKey ? 1 : event.code === "KeyS" ? event.shiftKey ? 3 : 2 : -1;
    if (index < 0) return;
    event.preventDefault();
    if (!event.repeat && !busy && !dialog.open) buttons[index].click();
  });
  if (!native) status.textContent = "Native file access unavailable.";
  render();
})();
