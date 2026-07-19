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

  Object.defineProperty(window, "velox", {
    value: Object.freeze({ invoke: Object.freeze(invoke) }),
    configurable: false,
    enumerable: true,
    writable: false,
  });
})();
