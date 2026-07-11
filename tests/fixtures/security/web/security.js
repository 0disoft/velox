function exercisePolicies() {
  setTimeout(() => window.__veloxReady("security"), 2000);

  const frame = document.createElement("iframe");
  frame.src = "frame.html";
  document.body.append(frame);

  window.open("popup.html", "velox-security-popup");

  const download = document.createElement("a");
  download.href = "download.txt";
  download.download = "velox-security-download.txt";
  document.body.append(download);
  download.click();

  if (typeof Notification !== "undefined") {
    Notification.requestPermission().catch(() => {});
  }

  window.location.href = "https://example.invalid/blocked";
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", exercisePolicies, { once: true });
} else {
  exercisePolicies();
}
