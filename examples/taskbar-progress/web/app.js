(function startTaskbarProgress() {
  "use strict";
  const states = document.querySelector("#states");
  const slider = document.querySelector("#value");
  const percentage = document.querySelector("#percentage");
  const clear = document.querySelector("#clear-progress");
  const indicator = document.querySelector("#indicator");
  const bar = document.querySelector("#progress-bar");
  const fill = document.querySelector("#progress-fill");
  const label = document.querySelector("#progress-label");
  const status = document.querySelector("#status");
  let busy = false;

  function selection() {
    const state = document.querySelector('input[name="state"]:checked').value;
    const params = { state };
    if (["normal", "error", "paused"].includes(state)) params.value = Number(slider.value);
    return params;
  }

  function render() {
    const params = selection();
    const title = params.state[0].toUpperCase() + params.state.slice(1);
    states.disabled = clear.disabled = busy;
    slider.disabled = busy || params.value === undefined;
    percentage.value = `${slider.value}%`;
    indicator.dataset.state = params.state;
    label.textContent = title + (params.value === undefined ? "" : ` - ${params.value}%`);
    fill.style.width = `${Number(slider.value)}%`;
    bar.setAttribute("aria-hidden", String(params.state === "none"));
    bar.setAttribute("aria-valuetext", label.textContent);
    if (params.value === undefined) bar.removeAttribute("aria-valuenow");
    else bar.setAttribute("aria-valuenow", String(params.value));
  }

  async function apply() {
    if (busy) return;
    const params = selection();
    render();
    if (typeof window.velox?.invoke !== "function") {
      status.textContent = "Native progress is unavailable.";
      return;
    }
    busy = true;
    render();
    try {
      await window.velox.invoke("window.setProgress", params);
      status.textContent = params.state === "none" ? "Progress cleared." : `Requested: ${label.textContent}`;
    } catch (error) {
      status.textContent = `Progress failed: ${error?.code || "NATIVE_OPERATION_FAILED"}`;
    } finally {
      busy = false;
      render();
    }
  }

  states.addEventListener("change", apply);
  slider.addEventListener("input", render);
  slider.addEventListener("change", apply);
  clear.addEventListener("click", () => {
    if (busy) return;
    document.querySelector('input[name="state"][value="none"]').checked = true;
    apply();
  });
  render();
  requestAnimationFrame(() => requestAnimationFrame(() => {
    if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
  }));
})();
