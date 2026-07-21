import { lstat, rm } from "node:fs/promises";
import { resolve } from "node:path";

const root = resolve(import.meta.dir, "..");
const cli = resolve(root, "dist/release/velox-windows-x64/velox.exe");
const definitions = Object.freeze({
  "capability-probe": Object.freeze({ config: "examples/capability-probe/velox.json" }),
  deskboard: Object.freeze({ config: "examples/deskboard/velox.json" }),
  "file-notes": Object.freeze({ config: "examples/file-notes/velox.json" }),
});

export function definitionFor(name: string) {
  const definition = definitions[name as keyof typeof definitions];
  if (!definition) throw new Error(`unsupported example: ${name}`);
  return {
    config: resolve(root, definition.config),
    output: resolve(root, "dist", "examples", name),
  };
}

async function removeOwnedOutput(path: string) {
  try {
    const info = await lstat(path);
    if (info.isSymbolicLink()) throw new Error(`refusing to replace linked output: ${path}`);
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") return;
    throw error;
  }
  await rm(path, { recursive: true });
}

async function main() {
  const [name] = process.argv.slice(2);
  if (!name || process.argv.length !== 3) throw new Error("usage: bun scripts/build-example.ts <example>");
  const definition = definitionFor(name);
  await removeOwnedOutput(definition.output);
  const child = Bun.spawn([
    cli,
    "build",
    "--config", definition.config,
    "--out", definition.output,
    "--json",
  ], {
    cwd: root,
    env: process.env,
    stdin: "ignore",
    stdout: "inherit",
    stderr: "inherit",
  });
  const exitCode = await child.exited;
  if (exitCode !== 0) process.exit(exitCode);
}

if (import.meta.main) await main();
