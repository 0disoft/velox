/// <reference path="../velox.d.ts" />
(function startTrayApp() {
  "use strict";
  const form = document.querySelector("#notification-form");
  const kind = document.querySelector("#kind");
  const message = document.querySelector("#message");
  const count = document.querySelector("#message-count");
  const send = document.querySelector("#send-notification");
  const status = document.querySelector("#status");
  const kinds = ["info", "warning", "error"];
  const disallowed = /[\u0000-\u0008\u000B-\u001F\u007F-\u009F]/;
  let busy = false;

  function valid() {
    return kinds.includes(kind.value) && message.value.length <= 255 &&
      message.value.trim().length > 0 && !disallowed.test(message.value);
  }

  function render() {
    count.textContent = `${message.value.length} / 255`;
    send.disabled = busy || !valid();
  }

  function lock(locked) {
    busy = locked;
    kind.disabled = locked;
    message.disabled = locked;
    render();
  }

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (busy) return;
    if (!valid()) {
      status.textContent = "Enter a message of 1 to 255 characters.";
      render();
      return;
    }
    if (typeof window.velox?.invoke !== "function") {
      status.textContent = "Native notifications are unavailable.";
      return;
    }
    lock(true);
    try {
      await window.velox.invoke("notification.show", { kind: kind.value, message: message.value });
      status.textContent = "Request accepted.";
    } catch (error) {
      status.textContent = `Notification failed: ${error?.code || "NATIVE_OPERATION_FAILED"}`;
    } finally {
      lock(false);
      send.focus();
    }
  });
  kind.addEventListener("change", render);
  message.addEventListener("input", render);
  render();
  document.documentElement.dataset.velox = "ready";
})();
