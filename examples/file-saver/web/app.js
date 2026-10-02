"use strict";

const saveButton = document.querySelector("#save");
const saveAsButton = document.querySelector("#save-as");
const newButton = document.querySelector("#new");
const buttons = [saveButton, saveAsButton, newButton];
const editor = document.querySelector("#text");
const status = document.querySelector("#status");
const filename = document.querySelector("#filename");
let target = 0;
let baseline = "";
const unavailable = typeof window.velox?.saveTextAs !== "function";
for (const button of buttons) button.disabled = unavailable;
if (unavailable) status.textContent = "Native bridge unavailable.";
editor.addEventListener("input", () => {
  document.querySelector("#size").textContent = `${new TextEncoder().encode(editor.value).length.toLocaleString()} bytes`;
  status.textContent = "Unsaved changes.";
});
async function save(forceSelection) {
  for (const button of buttons) button.disabled = true;
  editor.readOnly = true;
  try {
    const result = target && !forceSelection
      ? await window.velox.saveTextTo(editor.value, target)
      : await window.velox.saveTextAs(editor.value, filename.textContent);
    if (result.cancelled) { status.textContent = "Cancelled. Text kept."; return; }
    filename.textContent = result.name;
    target = result.target;
    baseline = editor.value;
    status.textContent = `${result.name} saved (${result.bytes.toLocaleString()} bytes).`;
  } catch (error) {
    status.textContent = `${error.code || "ERROR"}: ${error.message}`;
  } finally {
    for (const button of buttons) button.disabled = false;
    editor.readOnly = false;
  }
}
saveButton.addEventListener("click", () => save(false));
saveAsButton.addEventListener("click", () => save(true));
newButton.addEventListener("click", async () => {
  if (editor.value !== baseline && !window.confirm("Discard unsaved changes?")) return;
  for (const button of buttons) button.disabled = true;
  editor.readOnly = true;
  try {
    if (target) await window.velox.invoke("file.releaseSaveTarget", { target });
    target = 0;
    baseline = "";
    editor.value = "";
    filename.textContent = "Untitled.txt";
    document.querySelector("#size").textContent = "";
    status.textContent = "New document.";
  } catch (error) {
    status.textContent = `${error.code || "ERROR"}: ${error.message}`;
  } finally {
    for (const button of buttons) button.disabled = false;
    editor.readOnly = false;
  }
});
window.addEventListener("beforeunload", (event) => {
  if (editor.value !== baseline) { event.preventDefault(); event.returnValue = ""; }
});
