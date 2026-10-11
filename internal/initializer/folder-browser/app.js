/// <reference path="../velox.d.ts" />
(() => {
  "use strict";
  const select = document.querySelector("#select");
  const refresh = document.querySelector("#refresh");
  const release = document.querySelector("#release");
  const folder = document.querySelector("#folder-name");
  const count = document.querySelector("#count");
  const entries = document.querySelector("#entries");
  const status = document.querySelector("#status");
  const preview = document.querySelector("#preview");
  const fileName = document.querySelector("#file-name");
  const fileBytes = document.querySelector("#file-bytes");
  const available = typeof window.velox?.invoke === "function";
  let target = 0;
  let busy = false;
  let fileButtons = [];
  let listingVersion = 0;

  function controls() {
    select.disabled = busy || !available;
    refresh.disabled = release.disabled = busy || !target;
    for (const button of fileButtons) button.disabled = busy || !target;
  }
  function clearPreview() {
    preview.value = "";
    fileName.textContent = "No file selected";
    fileBytes.textContent = "";
  }
  function clearListing() {
    listingVersion++;
    entries.replaceChildren();
    fileButtons = [];
    count.textContent = "";
    clearPreview();
  }
  function clear() {
    target = 0;
    folder.textContent = "No folder selected";
    clearListing();
    count.textContent = "0 items";
  }
  async function openText(name) {
    clearPreview();
    status.textContent = "Loading text...";
    const result = await window.velox.invoke("folder.openText", { target, name });
    preview.value = result.text;
    fileName.textContent = result.name;
    fileBytes.textContent = `${result.bytes.toLocaleString()} bytes`;
    status.textContent = "Text loaded.";
  }
  async function list() {
    clearListing();
    status.textContent = "Loading list...";
    const result = await window.velox.invoke("folder.list", { target });
    const listedTarget = target;
    const listedVersion = listingVersion;
    entries.replaceChildren(...result.entries.map((item) => {
      const row = document.createElement("tr");
      const nameCell = document.createElement("td");
      if (item.kind === "file") {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "file-name";
        button.textContent = item.name;
        button.title = `Open ${item.name} as text`;
        button.addEventListener("click", () => target === listedTarget && listingVersion === listedVersion && run(() => openText(item.name)));
        fileButtons.push(button);
        nameCell.appendChild(button);
      } else {
        nameCell.textContent = item.name;
      }
      const kindCell = document.createElement("td");
      kindCell.textContent = item.kind;
      row.appendChild(nameCell);
      row.appendChild(kindCell);
      return row;
    }));
    count.textContent = `${result.entries.length} items`;
    status.textContent = result.truncated ? `Partial list. ${result.skipped} entries excluded.` : `List loaded. ${result.skipped} entries excluded.`;
  }
  async function run(action) {
    if (busy || !available) return;
    const previousFocus = document.activeElement;
    busy = true;
    controls();
    try {
      await action();
    } catch (error) {
      const code = typeof error?.code === "string" ? error.code : "ERROR";
      const message = typeof error?.message === "string" ? error.message : "The folder operation failed.";
      if (code === "FOLDER_TARGET_INVALID") clear();
      status.textContent = `${code}: ${message}`;
    } finally {
      busy = false;
      controls();
      if (document.activeElement === document.body || document.activeElement === previousFocus) {
        if (previousFocus?.isConnected && !previousFocus.disabled) previousFocus.focus();
        else select.focus();
      }
    }
  }
  select.addEventListener("click", () => run(async () => {
    status.textContent = "Selecting folder...";
    const result = await window.velox.invoke("folder.select");
    if (result.cancelled) { status.textContent = "Selection canceled."; return; }
    clear();
    target = result.target;
    folder.textContent = result.name;
    await list();
  }));
  refresh.addEventListener("click", () => target && run(list));
  release.addEventListener("click", () => target && run(async () => {
    status.textContent = "Releasing folder...";
    await window.velox.invoke("folder.release", { target });
    clear();
    status.textContent = "Folder released.";
  }));
  controls();
  if (!available) status.textContent = "Native folder bridge unavailable.";
  requestAnimationFrame(() => requestAnimationFrame(() => {
    if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
  }));
})();
