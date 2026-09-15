import { chmod, copyFile, mkdir, readFile, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { resolve } from "node:path";
import { execFileSync } from "node:child_process";

const [sharpModule, winresModule, source] = process.argv.slice(2);
if (!sharpModule || !winresModule) {
  throw new Error("usage: node scripts/generate-windows-icon.mjs <sharp-module> <cached-winres-module> [source.png]");
}
const sharp = createRequire(import.meta.url)(resolve(sharpModule));
const root = resolve(import.meta.dirname, "..");
const assets = resolve(root, "assets/branding");
const work = resolve(root, ".cache/windows-icon");
await mkdir(assets, { recursive: true });
await mkdir(work, { recursive: true });
const png = resolve(assets, "velox.png");
if (source) await copyFile(resolve(source), png);
const sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256];
const images = [];
for (const size of sizes) {
  images.push(await sharp(png).resize(size, size).ensureAlpha().png().toBuffer());
}
const header = Buffer.alloc(6 + 16 * images.length);
header.writeUInt16LE(1, 2);
header.writeUInt16LE(images.length, 4);
let offset = header.length;
for (const [i, data] of images.entries()) {
  const entry = 6 + i * 16;
  header[entry] = header[entry + 1] = sizes[i] % 256;
  header.writeUInt16LE(1, entry + 4);
  header.writeUInt16LE(32, entry + 6);
  header.writeUInt32LE(data.length, entry + 8);
  header.writeUInt32LE(offset, entry + 12);
  offset += data.length;
}
await writeFile(resolve(assets, "velox.ico"), Buffer.concat([header, ...images]));
const output = resolve(root, "cmd/velox-host/icon_windows_amd64.syso");
const ico = resolve(assets, "velox.ico");
const modfile = resolve(work, "go.mod");
const toolModule = await readFile(resolve(winresModule, "go.mod"), "utf8");
await writeFile(modfile, toolModule.replace("golang.org/x/image v0.12.0", "golang.org/x/image v0.41.0"));
const sumfile = resolve(work, "go.sum");
try { await chmod(sumfile, 0o600); } catch (error) { if (error.code !== "ENOENT") throw error; }
await writeFile(sumfile, await readFile(resolve(winresModule, "go.sum")));
execFileSync("go", ["run", "-mod=mod", `-modfile=${modfile}`, resolve(root, "scripts/windows-icon.go"), ico, output], {
  cwd: resolve(winresModule), windowsHide: true, stdio: "pipe", timeout: 90000,
  env: { ...process.env, GOCACHE: resolve(root, ".cache/go-build"), GOPROXY: "off", GOSUMDB: "off", GOTOOLCHAIN: "local", GOWORK: "off" },
});
const coff = await readFile(output);
if (coff.readUInt16LE(0) !== 0x8664) throw new Error("Expected x64 COFF resource object");
coff.writeUInt32LE(0, 4);
await writeFile(output, coff);
console.log(JSON.stringify({ sizes, output, timestamp: 0 }));
