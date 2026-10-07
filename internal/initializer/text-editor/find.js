(function defineEditorFind(global) {
  "use strict";

  function find(text, query, anchor = -1, backwards = false, matchCase = false) {
    if (!query) return { count: 0, index: 0, start: -1, length: 0 };
    const literal = query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const pattern = new RegExp(literal, matchCase ? "gu" : "giu");
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

  function attach(document, editor, blocked) {
    const bar = document.querySelector("#find-bar");
    const input = document.querySelector("#find-input");
    const count = document.querySelector("#find-count");
    const toggle = document.querySelector("#find-document");
    const previous = document.querySelector("#find-previous");
    const next = document.querySelector("#find-next");
    const close = document.querySelector("#find-close");
    const matchCase = document.querySelector("#find-case");
    let current = -1, total = 0, ordinal = 0, lastText = "", lastQuery = "", lastCase = false, composing = false;
    let mirror = null, mirrorText = null;
    bar.hidden = true;

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
      toggle.disabled = input.disabled = matchCase.disabled = disabled;
      previous.disabled = next.disabled = disabled || composing || !total;
      toggle.setAttribute("aria-expanded", String(!bar.hidden));
      count.textContent = input.value ? `${ordinal.toLocaleString()} / ${total.toLocaleString()}` : "";
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
      if (blocked() || composing || !input.value) return;
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
    function show() {
      if (blocked() || composing) return;
      bar.hidden = false;
      refresh();
      input.focus();
      input.select();
    }
    function hide() {
      if (composing) return;
      bar.hidden = true;
      if (mirror) { mirror.remove(); mirror = mirrorText = null; }
      controls();
      editor.focus({ preventScroll: true });
    }
    function changed() {
      current = -1;
      if (blocked() || composing) { refresh(); return; }
      move();
      if (!input.value) { total = ordinal = 0; controls(); }
    }
    toggle.addEventListener("click", show);
    previous.addEventListener("click", () => move(true));
    next.addEventListener("click", () => move());
    close.addEventListener("click", hide);
    input.addEventListener("input", changed);
    matchCase.addEventListener("change", changed);
    input.addEventListener("compositionstart", () => { composing = true; controls(); });
    input.addEventListener("compositionend", () => { composing = false; changed(); });
    controls();
    return Object.freeze({
      refresh, show,
      handleKey(event) {
        if (composing) return true;
        if (bar.hidden || blocked() || event.ctrlKey || event.altKey || event.metaKey) return false;
        if (event.key === "Escape") { event.preventDefault(); hide(); return true; }
        if (event.key === "Enter" && event.target === input) {
          event.preventDefault();
          if (!event.repeat) move(event.shiftKey);
          return true;
        }
        return false;
      },
    });
  }

  global.EditorFind = Object.freeze({ find, attach });
})(typeof window === "undefined" ? globalThis : window);
