import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { runInNewContext } from "node:vm";

const context = {};
runInNewContext(readFileSync(new URL("../examples/file-notes/web/model.js", import.meta.url), "utf8"), context);
const previous = state => ({ lines: state.text.length === 0 ? 1 : state.text.split(/\r\n|\r|\n/).length,
  characters: Array.from(state.text.replace(/\r\n?/g, "\n")).length });
function median(values) { return values.sort((a, b) => a - b)[Math.floor(values.length / 2)]; }
const results = [];
for (const [name, unit] of [["mixed-unicode", "Hello \uD55C\uAE00 \uD83D\uDE42\r\n"], ["dense-lines", "\n"]]) {
  const state = { text: unit.repeat(Math.floor(2 * 1024 * 1024 / Buffer.byteLength(unit))) };
  assert.deepEqual({ ...context.FileNotesModel.stats(state) }, previous(state));
  const times = [[], []];
  const functions = [previous, context.FileNotesModel.stats];
  for (let iteration = 0; iteration < 10; iteration++) {
    for (const index of iteration % 2 ? [1, 0] : [0, 1]) {
      const start = performance.now();
      functions[index](state);
      if (iteration >= 2) times[index].push(performance.now() - start);
    }
  }
  results.push({ name, utf8Bytes: Buffer.byteLength(state.text), samples: times[0].length,
    previousMedianMs: median(times[0]), currentMedianMs: median(times[1]) });
}
console.log(JSON.stringify({ runtime: process.version, results }));
