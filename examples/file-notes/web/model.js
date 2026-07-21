(function defineFileNotesModel(root) {
  "use strict";

  const schemaVersion = 1;
  const defaultName = "Untitled.md";

  function createState() {
    return { schemaVersion, name: defaultName, text: "", savedText: "", handle: null, updatedAt: null };
  }

  function restoreDraft(candidate) {
    if (!candidate || candidate.schemaVersion !== schemaVersion || typeof candidate.text !== "string") return createState();
    return {
      schemaVersion,
      name: validName(candidate.name),
      text: candidate.text,
      savedText: typeof candidate.savedText === "string" ? candidate.savedText : "",
      handle: candidate.handle || null,
      updatedAt: validDate(candidate.updatedAt) ? candidate.updatedAt : null,
    };
  }

  function replaceText(state, text, updatedAt) {
    return { ...state, text: String(text), updatedAt };
  }

  function openDocument(state, name, text, handle, updatedAt) {
    return { ...state, name: validName(name), text, savedText: text, handle: handle || null, updatedAt };
  }

  function markSaved(state, name, handle, updatedAt) {
    return { ...state, name: validName(name), savedText: state.text, handle: handle || state.handle, updatedAt };
  }

  function newDocument() {
    return createState();
  }

  function isDirty(state) {
    return state.text !== state.savedText;
  }

  function stats(state) {
    return {
      lines: state.text.length === 0 ? 1 : state.text.split(/\r\n|\r|\n/).length,
      characters: Array.from(state.text.replace(/\r\n?/g, "\n")).length,
    };
  }

  function validName(value) {
    return typeof value === "string" && value.trim() ? value.trim().slice(0, 255) : defaultName;
  }

  function validDate(value) {
    return typeof value === "string" && !Number.isNaN(Date.parse(value));
  }

  root.FileNotesModel = Object.freeze({
    schemaVersion,
    createState,
    restoreDraft,
    replaceText,
    openDocument,
    markSaved,
    newDocument,
    isDirty,
    stats,
  });
})(globalThis);
