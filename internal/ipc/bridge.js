(() => {
  "use strict";

  const nativeInvoke = window.__veloxInvoke;
  if (window.top !== window || typeof nativeInvoke !== "function") {
    return;
  }

  Object.defineProperty(window, "__veloxInvoke", {
    value: nativeInvoke,
    configurable: false,
    enumerable: false,
    writable: false,
  });

  const pending = new Set();
  let nextRequestID = 1;

  function allocateRequestID() {
    for (let attempts = 0; attempts < 0xffffffff; attempts += 1) {
      const candidate = nextRequestID;
      nextRequestID = nextRequestID === 0xffffffff ? 1 : nextRequestID + 1;
      if (!pending.has(candidate)) {
        return candidate;
      }
    }
    throw createError("TOO_MANY_REQUESTS", "No native request identifier is available.");
  }

  function createError(code, message) {
    const error = new Error(message);
    Object.defineProperty(error, "code", {
      value: code,
      configurable: false,
      enumerable: true,
      writable: false,
    });
    return error;
  }

  async function invoke(method, params = {}) {
    if (pending.size >= 64) {
      throw createError("TOO_MANY_REQUESTS", "The native request limit has been reached.");
    }

    const id = allocateRequestID();
    pending.add(id);
    try {
      const response = await nativeInvoke({ v: 1, id, method, params });
      if (!response || response.v !== 1 || response.id !== id || typeof response.ok !== "boolean") {
        throw createError("INVALID_RESPONSE", "The native response is malformed.");
      }
      if (!response.ok) {
        const code = response.error?.code || "INTERNAL";
        const message = response.error?.message || "The native operation failed.";
        throw createError(code, message);
      }
      return response.result;
    } finally {
      pending.delete(id);
    }
  }

  async function uploadText(text, name, commit, extra = {}) {
    if (typeof text !== "string" || !text.isWellFormed() || text.includes("\0")) {
      throw createError("INVALID_PARAMS", "Save text must be valid UTF-8 without NUL characters.");
    }
    if (text.length > 2 * 1024 * 1024) {
      throw createError("PAYLOAD_TOO_LARGE", "The text exceeds 2 MiB.");
    }
    const encoder = new TextEncoder();
    const bytes = encoder.encode(text).length;
    if (bytes > 2 * 1024 * 1024) {
      throw createError("PAYLOAD_TOO_LARGE", "The text exceeds 2 MiB.");
    }
    const { token } = await invoke("file.beginSave", { name, bytes });
    try {
      let offset = 0;
      for (let start = 0; start < text.length;) {
        let end = Math.min(start + 4096, text.length);
        const last = text.charCodeAt(end - 1);
        if (end < text.length && last >= 0xd800 && last <= 0xdbff) end -= 1;
        const chunk = text.slice(start, end);
        await invoke("file.appendSave", { token, offset, text: chunk });
        offset += encoder.encode(chunk).length;
        start = end;
      }
      // Commit consumes the upload even when selection is cancelled or writing fails.
      return await invoke(commit, { token, ...extra });
    } finally {
      // A consumed token no longer exists; cleanup errors must not mask the save result.
      await invoke("file.cancelSave", { token }).catch(() => {});
    }
  }

  function saveText(text, name = "Untitled.txt") {
    return uploadText(text, name, "file.commitSave");
  }

  function saveTextAs(text, name = "Untitled.txt") {
    return uploadText(text, name, "file.commitSaveAs");
  }

  async function saveTextTo(text, target) {
    if (!Number.isInteger(target) || target <= 0 || target > 0xffffffff) {
      throw createError("INVALID_PARAMS", "A connected save target is required.");
    }
    return uploadText(text, "Untitled.txt", "file.commitSaveTo", { target });
  }

  Object.defineProperty(window, "velox", {
    value: Object.freeze({ invoke: Object.freeze(invoke), saveText: Object.freeze(saveText), saveTextAs: Object.freeze(saveTextAs), saveTextTo: Object.freeze(saveTextTo) }),
    configurable: false,
    enumerable: true,
    writable: false,
  });
})();
