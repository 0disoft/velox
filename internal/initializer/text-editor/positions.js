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

  function createIndex(text) {
    if (text.length > maxUnits) throw new RangeError("Position index exceeds the document limit.");
    const length = text.length;
    let lineCount = 1, offset = -1;
    while ((offset = text.indexOf("\n", offset + 1)) >= 0) lineCount++;
    const lines = new Uint32Array(lineCount);
    let line = 1;
    offset = -1;
    while ((offset = text.indexOf("\n", offset + 1)) >= 0) lines[line++] = offset + 1;
    let ends = null;
    // ASCII without CR has one UTF-16 unit per grapheme, so no boundary array is needed.
    if (/[^\x00-\x7f]|\r/.test(text)) {
      const boundaries = new Uint32Array(length + 1);
      const segmenter = new Intl.Segmenter(undefined, { granularity: "grapheme" });
      let count = 1;
      for (const part of segmenter.segment(text)) boundaries[count++] = part.index + part.segment.length;
      ends = boundaries.subarray(0, count);
    }
    const clamp = offset => Math.min(length, Math.max(0, Number.isFinite(offset) ? Math.trunc(offset) : 0));
    return Object.freeze({
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
    });
  }

  global.EditorPosition = Object.freeze({ createIndex });
})(typeof window === "undefined" ? globalThis : window);
