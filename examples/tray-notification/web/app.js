(function startTrayNotification() {
  "use strict";
  const form = document.querySelector("#notification-form");
  const kind = document.querySelector("#kind");
  const message = document.querySelector("#message");
  const count = document.querySelector("#message-count");
  const send = document.querySelector("#send-notification");
  const status = document.querySelector("#status");
  const KINDS = ["info", "warning", "error"];
  const DISALLOWED = /[\u0000-\u0008\u000B-\u001F\u007F-\u009F]/;
  let busy = false;

  function validMessage(text) {
    return text.length >= 1 && text.length <= 255 && text.trim().length > 0 && !DISALLOWED.test(text);
  }

  function valid() {
    return KINDS.includes(kind.value) && validMessage(message.value);
  }

  function render() {
    const label = `${message.value.length} / 255`;
    count.value = label;
    count.textContent = label;
    send.disabled = busy || !valid();
  }

  function lock(locked) {
    busy = locked;
    kind.disabled = locked;
    message.disabled = locked;
    render();
  }

  async function handleSubmit(event) {
    if (event && typeof event.preventDefault === "function") event.preventDefault();
    if (busy) return;
    if (!valid()) {
      status.textContent = "Enter a message of 1 to 255 UTF-16 units.";
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
      if (typeof send.focus === "function") send.focus();
    }
  }

  form.addEventListener("submit", handleSubmit);
  kind.addEventListener("change", render);
  message.addEventListener("input", render);
  render();
  requestAnimationFrame(() => requestAnimationFrame(() => {
    if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
  }));
})();
