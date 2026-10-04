(function startActivationShortcut() {
  "use strict";
  const note = document.querySelector("#note");
  const count = document.querySelector("#note-count");
  const status = document.querySelector("#status");
  const LIMIT = 2048;

  function render() {
    const label = `${note.value.length} / ${LIMIT}`;
    count.value = label;
    count.textContent = label;
  }

  note.addEventListener("input", () => {
    render();
    status.textContent = "Note changed.";
  });

  window.addEventListener("focus", () => {
    status.textContent = "Window focused.";
  });

  render();
  requestAnimationFrame(() => requestAnimationFrame(() => {
    if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
  }));
})();
