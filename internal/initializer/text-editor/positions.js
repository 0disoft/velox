(function defineEditorPosition(global) {
  "use strict";
  const maxUnits = 2 * 1024 * 1024;

  function bound(values, offset, inclusive) {
    let low = 0, high = values.length;
    while (low < high) {
      const middle = (low + high) >>> 1;
      if (values[middle] < offset || inclusive && values[middle] === offset) low = middle + 1;
      else high = middle;
    }
    return low;
  }

  function prepareIndex(text) {
    if (text.length > maxUnits) throw new RangeError("Position index exceeds the document limit.");
    const length = text.length;
    let lineCount = 1, offset = -1;
    while ((offset = text.indexOf("\n", offset + 1)) >= 0) lineCount++;
    const lines = new Uint32Array(lineCount);
    let line = 1;
    offset = -1;
    while ((offset = text.indexOf("\n", offset + 1)) >= 0) lines[line++] = offset + 1;
    let ends = null, boundaries = null, parts = null, count = 1;
    // ASCII without CR has one UTF-16 unit per grapheme, so no boundary array is needed.
    if (/[^\x00-\x7f]|\r/.test(text)) {
      boundaries = new Uint32Array(length + 1);
      const segmenter = new Intl.Segmenter(undefined, { granularity: "grapheme" });
      parts = segmenter.segment(text)[Symbol.iterator]();
    }
    const clamp = offset => Math.min(length, Math.max(0, Number.isFinite(offset) ? Math.trunc(offset) : 0));
    function finish() { return Object.freeze({
      at(start, end = start, direction = "none") {
        start = clamp(start); end = clamp(end);
        if (start > end) [start, end] = [end, start];
        const caret = direction === "backward" ? start : end;
        const line = bound(lines, caret, true) - 1;
        const before = offset => bound(ends, offset, true) - 1;
        return {
          line: line + 1,
          column: ends === null ? caret - lines[line] + 1 : before(caret) - before(lines[line]) + 1,
          selected: start === end ? 0 : ends === null ? end - start :
            bound(ends, end, false) - before(start),
        };
      },
    }); }
    return {
      step(limit) {
        if (parts) {
          for (let i = 0; i < limit; i++) {
            const part = parts.next();
            if (part.done) { ends = boundaries.subarray(0, count); parts = null; break; }
            boundaries[count++] = part.value.index + part.value.segment.length;
          }
          if (parts) return null;
        }
        return finish();
      },
    };
  }

  function createIndex(text) { return prepareIndex(text).step(Infinity); }

  function attach(document, editor, composing) {
    const label = document.querySelector("#cursor-position");
    let index = null, lastText = null, pending = false, timer = null;
    function refresh() {
      if (pending) { label.textContent = "Ln ..., Col ..."; return; }
      if (index === null) { label.textContent = "Position unavailable"; return; }
      const position = index.at(editor.selectionStart, editor.selectionEnd, editor.selectionDirection);
      label.textContent = `Ln ${position.line.toLocaleString()}, Col ${position.column.toLocaleString()}` +
        (position.selected ? ` | Selected ${position.selected.toLocaleString()}` : "");
    }
    function update() {
      global.clearTimeout(timer);
      timer = null;
      if (composing()) { pending = true; refresh(); return; }
      const text = editor.value;
      if (text === lastText) { pending = false; refresh(); return; }
      pending = true;
      refresh();
      let builder = null;
      const rebuild = () => {
        timer = null;
        if (composing()) return;
        try {
          builder ??= prepareIndex(text);
          const nextIndex = builder.step(text.length > 64 * 1024 ? 4096 : Infinity);
          if (nextIndex === null) { timer = global.setTimeout(rebuild, 0); return; }
          index = nextIndex;
        }
        catch { index = null; }
        lastText = text;
        pending = false;
        refresh();
      };
      if (text.length > 64 * 1024 && text.length <= maxUnits) timer = global.setTimeout(rebuild, 80);
      else rebuild();
    }
    for (const event of ["select", "selectionchange", "keyup", "click", "focus"]) editor.addEventListener(event, refresh);
    document.addEventListener("selectionchange", () => { if (document.activeElement === editor) refresh(); });
    update();
    return Object.freeze({ update });
  }

  global.EditorPosition = Object.freeze({ createIndex, attach });
})(typeof window === "undefined" ? globalThis : window);
