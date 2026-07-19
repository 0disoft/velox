async function verifyIPC() {
  if (!window.velox || typeof window.velox.invoke !== "function") {
    throw Object.assign(new Error("public bridge is unavailable"), { code: "BRIDGE_MISSING" });
  }
  if (!Object.isFrozen(window.velox) || !Object.isFrozen(window.velox.invoke)) {
    throw Object.assign(new Error("public bridge is not frozen"), { code: "BRIDGE_MUTABLE" });
  }

  const info = await window.velox.invoke("app.getInfo");
  if (info.id !== "dev.velox.security-fixture" || info.platform !== "windows") {
    throw new Error("application identity mismatch");
  }

  try {
    await window.velox.invoke("window.getState");
    throw new Error("window.getState bypassed its permission");
  } catch (error) {
    if (error.code !== "PERMISSION_DENIED") {
      throw error;
    }
  }

  try {
    await window.velox.invoke("shell.execute");
    throw new Error("unknown native method was accepted");
  } catch (error) {
    if (error.code !== "METHOD_NOT_FOUND") {
      throw error;
    }
  }

  await window.__veloxReady("ipc-ok");
}

async function exercisePolicies() {
  await new Promise((resolve) => {
    requestAnimationFrame(() => requestAnimationFrame(resolve));
  });
  try {
    await verifyIPC();
  } catch (error) {
    const code = typeof error.code === "string" ? error.code : "UNKNOWN_FAILURE";
    await window.__veloxReady(`ipc-${code}`);
    return;
  }

  window.open("popup.html", "velox-security-popup");

  const frame = document.createElement("iframe");
  frame.src = "frame.html";
  document.body.append(frame);

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
  void exercisePolicies();
}
