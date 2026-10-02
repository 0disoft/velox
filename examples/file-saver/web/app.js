"use strict";

const button = document.querySelector("#save");
const editor = document.querySelector("#text");
const status = document.querySelector("#status");
const filename = document.querySelector("#filename");
button.disabled = typeof window.velox?.saveText !== "function";
if (button.disabled) status.textContent = "Native bridge unavailable.";
editor.addEventListener("input", () => {
  document.querySelector("#size").textContent = `${new TextEncoder().encode(editor.value).length.toLocaleString()} bytes`;
  status.textContent = "Unsaved changes.";
});
button.addEventListener("click", async () => {
  button.disabled = true;
  editor.readOnly = true;
  try {
    const result = await window.velox.saveText(editor.value, filename.textContent);
    if (result.cancelled) { status.textContent = "Cancelled. Text kept."; return; }
    filename.textContent = result.name;
    status.textContent = `${result.name} saved (${result.bytes.toLocaleString()} bytes).`;
  } catch (error) {
    status.textContent = `${error.code || "ERROR"}: ${error.message}`;
  } finally {
    button.disabled = false;
    editor.readOnly = false;
  }
});
