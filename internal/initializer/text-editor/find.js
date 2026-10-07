(function defineEditorFind(global) {
  "use strict";

  function patternFor(query, matchCase) {
    const literal = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    return new RegExp(literal, matchCase ? "gu" : "giu");
  }

  function find(text, query, anchor = -1, backwards = false, matchCase = false) {
    if (!query) return { count: 0, index: 0, start: -1, length: 0 };
    const pattern = patternFor(query, matchCase);
    let count = 0, first = null, last = null, selected = null, index = 0, match;
    // Search original UTF-16 positions; lowercasing can change string length.
    while ((match = pattern.exec(text)) !== null) {
      count++;
      const position = { start: match.index, length: match[0].length };
      if (first === null) first = position;
      last = position;
      if (backwards ? position.start < anchor : selected === null && position.start > anchor) {
        selected = position;
        index = count;
      }
    }
    if (selected === null && count) {
      selected = backwards ? last : first;
      index = backwards ? count : 1;
    }
    return { count, index, start: selected?.start ?? -1, length: selected?.length ?? 0 };
  }

  function replace(text, query, value, matchCase = false, start = null) {
    const unchanged = { text, count: 0, start: -1, length: 0 };
    if (!query) return unchanged;
    const maxBytes = 2 * 1024 * 1024;
    const encoder = new TextEncoder();
    if (text.length > maxBytes || encoder.encode(text).byteLength > maxBytes) {
      throw new RangeError("Replacement requires a document within 2 MiB.");
    }
    const pattern = patternFor(query, matchCase);
    let count = 0, removed = 0, first = -1, match;
    if (start !== null) {
      if (!Number.isInteger(start) || start < 0) return unchanged;
      pattern.lastIndex = start;
      match = pattern.exec(text);
      if (!match || match.index !== start) return unchanged;
      count = 1; first = start; removed = match[0].length;
    } else {
      while ((match = pattern.exec(text)) !== null) {
        if (first < 0) first = match.index;
        count++; removed += match[0].length;
      }
    }
    if (!count) return unchanged;
    // Bound expansion before allocating the result; callback keeps $ tokens literal.
    if (text.length - removed + count * value.length > maxBytes) {
      throw new RangeError("Replacement would exceed the 2 MiB document limit.");
    }
    const result = start === null ? text.replace(pattern, () => value) :
      text.slice(0, first) + value + text.slice(first + removed);
    if (encoder.encode(result).byteLength > maxBytes) {
      throw new RangeError("Replacement would exceed the 2 MiB document limit.");
    }
    return { text: result, count, start: first, length: value.length };
  }

  function attach(document, editor, blocked, onChange = () => {}) {
    const bar = document.querySelector("#find-bar");
    const input = document.querySelector("#find-input");
    const count = document.querySelector("#find-count");
    const toggle = document.querySelector("#find-document");
    const previous = document.querySelector("#find-previous");
    const next = document.querySelector("#find-next");
    const close = document.querySelector("#find-close");
    const matchCase = document.querySelector("#find-case");
    const replaceToggle = document.querySelector("#replace-document");
    const replaceRow = document.querySelector("#replace-row");
    const replacement = document.querySelector("#replace-input");
    const replaceOne = document.querySelector("#replace-one");
    const replaceAll = document.querySelector("#replace-all");
    const undoButton = document.querySelector("#replace-undo");
    let current = -1, total = 0, ordinal = 0, lastText = "", lastQuery = "", lastCase = false, composing = false;
    let replacementComposing = false, undoSnapshot = null;
    let mirror = null, mirrorText = null;
    bar.hidden = true;
    replaceRow.hidden = true;

    function scrollMatch(start, length) {
      if (start === 0 || editor.scrollHeight <= editor.clientHeight) { editor.scrollTop = 0; return; }
      if (typeof global.getComputedStyle !== "function" || !document.createRange) return;
      if (!mirror) {
        mirror = document.createElement("div");
        mirror.setAttribute("aria-hidden", "true");
        mirror.style.cssText = "position:fixed;left:-10000px;top:0;visibility:hidden;contain:layout style paint;white-space:pre-wrap;box-sizing:border-box;border:0;";
        document.body.appendChild(mirror);
      }
      // Reuse File Notes' browser layout measurement for wrapped text.
      const style = global.getComputedStyle(editor);
      for (const name of ["fontFamily", "fontSize", "fontWeight", "fontStyle", "lineHeight", "letterSpacing", "wordSpacing", "tabSize", "wordBreak", "overflowWrap", "textIndent", "paddingTop", "paddingRight", "paddingBottom", "paddingLeft"]) mirror.style[name] = style[name];
      mirror.style.width = `${editor.clientWidth}px`;
      if (mirrorText !== editor.value) { mirrorText = editor.value; mirror.textContent = mirrorText; }
      const range = document.createRange();
      range.setStart(mirror.firstChild, start);
      range.setEnd(mirror.firstChild, start + length);
      const top = range.getBoundingClientRect().top - mirror.getBoundingClientRect().top;
      editor.scrollTop = Math.max(0, top - (editor.clientHeight - parseFloat(style.lineHeight)) / 2);
    }

    function controls() {
      const disabled = blocked();
      toggle.disabled = replaceToggle.disabled = input.disabled = replacement.disabled = matchCase.disabled = disabled;
      previous.disabled = next.disabled = disabled || composing || !total;
      replaceOne.disabled = replaceAll.disabled = disabled || composing || replacementComposing || !total;
      undoButton.disabled = disabled || composing || replacementComposing || !canUndo();
      toggle.setAttribute("aria-expanded", String(!bar.hidden));
      replaceToggle.setAttribute("aria-expanded", String(!bar.hidden && !replaceRow.hidden));
      count.textContent = input.value ? `${ordinal.toLocaleString()} / ${total.toLocaleString()}` : "";
      replaceAll.title = `Replace all (${total.toLocaleString()})`;
    }
    function refresh() {
      if (!bar.hidden && !blocked() && !composing &&
          (lastText !== editor.value || lastQuery !== input.value || lastCase !== matchCase.checked)) {
        lastText = editor.value;
        lastQuery = input.value;
        lastCase = matchCase.checked;
        current = -1;
        ordinal = 0;
        total = find(editor.value, input.value, -1, false, matchCase.checked).count;
      }
      controls();
    }
    function move(backwards = false) {
      if (bar.hidden || blocked() || composing || !input.value) return;
      const result = find(editor.value, input.value, current, backwards, matchCase.checked);
      current = result.start;
      total = result.count;
      ordinal = result.index;
      lastText = editor.value;
      lastQuery = input.value;
      lastCase = matchCase.checked;
      if (current !== -1) {
        const focused = document.activeElement;
        editor.setSelectionRange(current, current + result.length);
        scrollMatch(current, result.length);
        editor.focus({ preventScroll: true });
        if (focused === input || focused === matchCase) focused.focus({ preventScroll: true });
      }
      controls();
    }
    function show(withReplace = false) {
      if (blocked() || composing || replacementComposing) return;
      bar.hidden = false;
      replaceRow.hidden = !withReplace;
      refresh();
      input.focus();
      input.select();
    }
    function hide() {
      bar.hidden = true;
      composing = false;
      replacementComposing = false;
      if (mirror) { mirror.remove(); mirror = mirrorText = null; }
      controls();
      editor.focus({ preventScroll: true });
    }
    function changed() {
      if (bar.hidden) { refresh(); return; }
      current = -1;
      if (blocked() || composing) { refresh(); return; }
      move();
      if (!input.value) { total = ordinal = 0; controls(); }
    }
    function canUndo() { return undoSnapshot !== null && editor.value === undoSnapshot.after; }
    function edited() { undoSnapshot = null; controls(); }
    function reset() { edited(); current = -1; lastText = null; refresh(); }
    function applyReplacement(all) {
      if (bar.hidden || replaceRow.hidden || blocked() || composing || replacementComposing) return;
      refresh();
      if (!total) return;
      if (!all && current < 0) move();
      const before = editor.value;
      const snapshot = { before, selectionStart: editor.selectionStart, selectionEnd: editor.selectionEnd, scrollTop: editor.scrollTop };
      let result;
      try { result = replace(before, input.value, replacement.value, matchCase.checked, all ? null : current); }
      catch (error) { onChange(error.message, false); return; }
      if (!result.count || result.text === before) { onChange("Document unchanged.", false); return; }
      undoSnapshot = { ...snapshot, after: result.text };
      editor.value = result.text;
      editor.setSelectionRange(result.start, result.start + result.length);
      onChange(`Replaced ${result.count.toLocaleString()} ${result.count === 1 ? "match" : "matches"}.`);
      if (!all) { current = result.start + result.length - 1; move(); }
      else refresh();
      controls();
    }
    function undo() {
      if (blocked() || composing || replacementComposing || !canUndo()) return false;
      const snapshot = undoSnapshot;
      undoSnapshot = null;
      editor.value = snapshot.before;
      editor.setSelectionRange(snapshot.selectionStart, snapshot.selectionEnd);
      editor.scrollTop = snapshot.scrollTop;
      onChange("Replacement undone.");
      controls();
      editor.focus({ preventScroll: true });
      return true;
    }
    function handleKey(event) {
      if (event.defaultPrevented || blocked()) return false;
      if (!event.isComposing && !composing && !replacementComposing && event.keyCode !== 229 &&
          event.ctrlKey && !event.altKey && !event.metaKey && !event.shiftKey &&
          event.code === "KeyZ" && event.target === editor && canUndo()) {
        event.preventDefault();
        if (!event.repeat) undo();
        return true;
      }
      if (bar.hidden) return false;
      if (event.isComposing) return true;
      if (event.key === "Escape" && !event.ctrlKey && !event.altKey && !event.metaKey) {
        event.preventDefault(); hide(); return true;
      }
      if (composing || replacementComposing || event.keyCode === 229) return true;
      if (event.ctrlKey || event.altKey || event.metaKey) return false;
      if (event.key === "Enter" && event.target === input) {
        event.preventDefault();
        if (!event.repeat) move(event.shiftKey);
        return true;
      }
      return false;
    }
    toggle.addEventListener("click", () => show());
    replaceToggle.addEventListener("click", () => show(true));
    replaceOne.addEventListener("click", () => applyReplacement(false));
    replaceAll.addEventListener("click", () => applyReplacement(true));
    undoButton.addEventListener("click", undo);
    replacement.addEventListener("keydown", handleKey, { capture: true });
    replacement.addEventListener("input", controls);
    replacement.addEventListener("compositionstart", () => { replacementComposing = true; controls(); });
    replacement.addEventListener("compositionend", () => { replacementComposing = false; controls(); });
    replacement.addEventListener("blur", () => { replacementComposing = false; controls(); });
    previous.addEventListener("click", () => move(true));
    next.addEventListener("click", () => move());
    close.addEventListener("click", hide);
    input.addEventListener("input", changed);
    input.addEventListener("keydown", handleKey, { capture: true });
    input.addEventListener("blur", () => { composing = false; controls(); });
    matchCase.addEventListener("change", changed);
    input.addEventListener("compositionstart", () => { composing = true; controls(); });
    input.addEventListener("compositionend", () => { composing = false; changed(); });
    controls();
    return Object.freeze({
      refresh, show, handleKey, edited, reset,
    });
  }

  global.EditorFind = Object.freeze({ find, replace, attach });
})(typeof window === "undefined" ? globalThis : window);
