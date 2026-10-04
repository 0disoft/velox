(function startWindowAttention() {
  "use strict";
  const count = document.querySelector("#flash-count");
  const delay = document.querySelector("#delay-seconds");
  const request = document.querySelector("#request-attention");
  const cancel = document.querySelector("#cancel-attention");
  const status = document.querySelector("#status");
  let timer = null;
  let revision = 0;

  function lock(locked, cancelPending = false) {
    request.disabled = count.disabled = delay.disabled = locked;
    cancel.disabled = cancelPending;
  }

  function available() {
    if (typeof window.velox?.invoke === "function") return true;
    status.textContent = "Native attention is unavailable.";
    return false;
  }

  async function invoke(method, params, current) {
    try {
      await window.velox.invoke(method, params);
      if (current === revision) status.textContent = method === "window.cancelAttention" ? "Canceled." : "Requested.";
    } catch (error) {
      if (current === revision) status.textContent = `Attention failed: ${error?.code || "NATIVE_OPERATION_FAILED"}`;
    } finally {
      if (current === revision) lock(false);
    }
  }

  request.addEventListener("click", () => {
    if (request.disabled) return;
    const flashes = Number(count.value);
    const seconds = Number(delay.value);
    if (!count.value.trim() || !delay.value.trim() || !Number.isInteger(flashes) || flashes < 1 || flashes > 5 ||
        !Number.isInteger(seconds) || seconds < 0 || seconds > 10) {
      status.textContent = "Invalid count or delay.";
      return;
    }
    if (!available()) return;
    const current = ++revision;
    lock(true);
    if (seconds === 0) return invoke("window.requestAttention", { count: flashes }, current);
    status.textContent = "Scheduled.";
    timer = setTimeout(() => {
      timer = null;
      if (current === revision) return invoke("window.requestAttention", { count: flashes }, current);
    }, seconds * 1000);
  });

  cancel.addEventListener("click", () => {
    if (cancel.disabled) return;
    const current = ++revision;
    clearTimeout(timer);
    timer = null;
    if (!available()) { lock(false); return; }
    lock(true, true);
    return invoke("window.cancelAttention", {}, current);
  });

  window.addEventListener("pagehide", () => { ++revision; clearTimeout(timer); timer = null; lock(false); });
  requestAnimationFrame(() => requestAnimationFrame(() => {
    if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
  }));
})();
