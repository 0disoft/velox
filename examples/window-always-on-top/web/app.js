"use strict";

requestAnimationFrame(() => {
  requestAnimationFrame(() => {
    if (typeof window.__veloxReady === "function") window.__veloxReady("dom-2raf");
  });
});
