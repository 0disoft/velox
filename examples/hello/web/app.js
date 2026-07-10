function reportReady() {
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      if (typeof window.__veloxM0Ready === "function") {
        window.__veloxM0Ready("dom-2raf");
      } else if (window.chrome?.webview?.postMessage) {
        window.chrome.webview.postMessage("ready dom-2raf");
      }
    });
  });
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", reportReady, { once: true });
} else {
  reportReady();
}
