"use strict";

const bridge = window.velox;
const status = document.querySelector("#bridge-status");
const lastResponse = document.querySelector("#last-response");
status.textContent = typeof bridge?.invoke === "function" ? "Native bridge ready" : "Native bridge unavailable";

for (const button of document.querySelectorAll("button[data-field]")) {
  button.disabled = typeof bridge?.invoke !== "function";
  button.addEventListener("click", async () => {
    const field = button.dataset.field;
    const output = document.getElementById(`${field}-result`);
    button.disabled = true;
    output.textContent = "Requesting";
    delete output.dataset.state;
    try {
      const result = await bridge.invoke("external.open", { url: document.getElementById(field).value });
      output.textContent = result?.queued === true ? "Queued for confirmation" : "Unexpected response";
      output.dataset.state = "queued";
      lastResponse.textContent = JSON.stringify(result, null, 2);
    } catch (error) {
      output.textContent = error.code || "ERROR";
      output.dataset.state = "error";
      lastResponse.textContent = `${error.code || "ERROR"}: ${error.message}`;
    } finally {
      button.disabled = false;
    }
  });
}
