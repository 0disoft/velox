"use strict";

const button = document.querySelector("#open");
const status = document.querySelector("#status");
button.disabled = typeof window.velox?.invoke !== "function";
if (button.disabled) status.textContent = "Native bridge unavailable.";
button.addEventListener("click", async () => {
  button.disabled = true;
  try {
    const file = await window.velox.invoke("file.openText");
    if (file.cancelled) {
      status.textContent = "Cancelled.";
      return;
    }
    document.querySelector("#filename").textContent = file.name;
    document.querySelector("#size").textContent = `${file.bytes.toLocaleString()} bytes`;
    document.querySelector("#text").value = file.text;
    status.textContent = "Opened.";
  } catch (error) {
    status.textContent = `${error.code || "ERROR"}: ${error.message}`;
  } finally {
    button.disabled = false;
  }
});
