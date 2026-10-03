(function startClipboardExample() {
  "use strict";
  const source = document.querySelector("#source-text");
  const result = document.querySelector("#result-text");
  const copy = document.querySelector("#copy-text");
  const paste = document.querySelector("#paste-text");
  const status = document.querySelector("#status");
  let pending = false;

  async function perform(operation, action) {
    if (pending) return;
    if (typeof window.velox?.invoke !== "function") {
      status.textContent = "Native clipboard access is unavailable.";
      return;
    }
    const focused = document.activeElement;
    pending = true;
    copy.disabled = paste.disabled = true;
    status.textContent = operation === "Paste" ? "Waiting for clipboard approval." : "Copying.";
    try {
      await action();
    } catch (error) {
      status.textContent = `${operation} failed: ${error.code || "NATIVE_OPERATION_FAILED"}`;
    } finally {
      pending = false;
      copy.disabled = paste.disabled = false;
      if ((focused === copy || focused === paste) && document.activeElement === document.body) focused.focus();
    }
  }

  copy.addEventListener("click", () => perform("Copy", async () => {
    await window.velox.invoke("clipboard.writeText", { text: source.value });
    status.textContent = "Copied.";
  }));
  paste.addEventListener("click", () => perform("Paste", async () => {
    const selected = await window.velox.invoke("clipboard.readText");
    if (selected?.cancelled === true) { status.textContent = "Paste canceled."; return; }
    if (selected?.cancelled !== false || typeof selected.text !== "string") {
      throw Object.assign(new Error(), { code: "INVALID_RESPONSE" });
    }
    result.value = selected.text;
    status.textContent = "Pasted.";
  }));
  requestAnimationFrame(() => requestAnimationFrame(() => {
    if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
  }));
})();
