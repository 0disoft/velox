import { describe, expect, test } from "bun:test";
import { resolve } from "node:path";
import { definitionFor } from "./build-example.ts";

const root = resolve(import.meta.dir, "..");

describe("build-example", () => {
  test("maps only repository-owned example outputs", () => {
    expect(definitionFor("deskboard")).toEqual({
      config: resolve(root, "examples/deskboard/velox.json"),
      output: resolve(root, "dist/examples/deskboard"),
    });
    expect(definitionFor("capability-probe")).toEqual({
      config: resolve(root, "examples/capability-probe/velox.json"),
      output: resolve(root, "dist/examples/capability-probe"),
    });
    expect(definitionFor("file-notes")).toEqual({
      config: resolve(root, "examples/file-notes/velox.json"),
      output: resolve(root, "dist/examples/file-notes"),
    });
  });

  test("rejects arbitrary output names", () => {
    expect(() => definitionFor("../release")).toThrow("unsupported example");
    expect(() => definitionFor("unknown")).toThrow("unsupported example");
  });
});
