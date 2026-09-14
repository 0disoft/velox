import { createHash, randomUUID } from "node:crypto";
import { spawnSync } from "node:child_process";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { join, resolve } from "node:path";

const root = resolve(import.meta.dir, "..");
const work = join(root, ".cache", "host-phases-" + randomUUID());
const env = { ...process.env, GOCACHE: join(root, ".cache/go-build"), GOPROXY: "off", GOSUMDB: "off" };
const hash = (bytes: string | Buffer) => createHash("sha256").update(bytes).digest("hex");
const originals: Record<string, string> = {};
const replacements: Record<string, string> = {};
const overlays: Record<string, string> = {};
const result: Record<string, unknown> = {
  schemaVersion: "velox.host-phase-probe/v1", outcome: "failure",
  startedAtUtc: new Date().toISOString(), releaseBinaryTested: false,
  fixture: "examples/hello", repetitions: 1,
};
await mkdir(work, { recursive: true });

function command(file: string, args: string[], timeout = 120_000, extraEnv = {}) {
  const child = spawnSync(file, args, {
    cwd: root, env: { ...env, ...extraEnv }, encoding: "utf8", windowsHide: true,
    stdio: ["ignore", "pipe", "pipe"], timeout, maxBuffer: 2 * 1024 * 1024,
  });
  if (child.error || child.status !== 0) {
    throw Error(`${file} failed: ${child.error ?? child.status}\n${child.stdout ?? ""}\n${child.stderr ?? ""}`);
  }
  return (child.stdout ?? "") + (child.stderr ?? "");
}

async function overlay(path: string, changes: [string, string][]) {
  const source = await readFile(join(root, path));
  originals[path] = hash(source);
  let text = source.toString("utf8").replace(/\r\n/g, "\n");
  for (const [before, after] of changes) {
    if (text.split(before).length !== 2) throw Error("Overlay anchor drift: " + path);
    text = text.replace(before, after);
  }
  const target = join(work, `overlay-${Object.keys(overlays).length}.go`);
  await writeFile(target, text);
  overlays[join(root, path)] = target;
  replacements[path] = hash(text);
}

type Point = { name: string; elapsedMs: number };
type Launch = { readyMs: number; startupTimeline: { phases: Point[] }; shutdownTimeline: { phases: Point[] } };
function summarize(launch: Launch) {
  function point(points: Point[], name: string) {
    const found = points.filter(p => p.name === name);
    if (found.length !== 1 || !Number.isFinite(found[0].elapsedMs)) throw Error("Missing or invalid point: " + name);
    return found[0].elapsedMs;
  }
  const startup = launch.startupTimeline.phases, shutdown = launch.shutdownTimeline.phases;
  const environment = point(startup, "environment-created");
  const entered = point(startup, "controller-callback-entered");
  const setup = point(startup, "controller-created");
  if (environment > entered || entered > setup) throw Error("Non-monotonic controller intervals");
  const close = shutdown.filter(p => /^controller-close-hresult-[0-9a-f]{8}$/.test(p.name));
  const refs = shutdown.filter(p => /^callback-owner-released-refs-\d+$/.test(p.name));
  if (close.length !== 1 || refs.length !== 1) throw Error("Missing teardown diagnostics");
  return {
    readyMs: launch.readyMs, environmentToCallbackMs: entered - environment,
    callbackToSetupMs: setup - entered,
    navigationToReadyMs: point(startup, "dom-2raf") - point(startup, "navigation-dispatched"),
    closeHRESULT: "0x" + close[0].name.slice(-8),
    callbackReferencesAfterOwnerRelease: Number(refs[0].name.split("-").at(-1)),
  };
}

try {
  command("git", ["diff", "--exit-code", "HEAD", "--", "cmd", "internal", "third_party", "tests/startup"]);
  result.baseCommit = command("git", ["rev-parse", "HEAD"]).trim();
  result.goVersion = command("go", ["version"]).trim();
  await overlay("third_party/go-webview2/pkg/edge/chromium.go", [
    ["func (e *Chromium) CreateCoreWebView2ControllerCompleted(res uintptr, controller *ICoreWebView2Controller) uintptr {",
      "func (e *Chromium) CreateCoreWebView2ControllerCompleted(res uintptr, controller *ICoreWebView2Controller) uintptr {\n\tif e.StartupPhase != nil { e.StartupPhase(\"controller-callback-entered\") }"],
    ["\tdefer e.releaseCallbackOwner()", `\tdefer func() {
        e.releaseCallbackOwner()
        callbackLifetimes.Lock()
        var refs uintptr
        if owner := callbackLifetimes.owners[e]; owner != nil { refs = owner.refs }
        callbackLifetimes.Unlock()
        e.markShutdown(fmt.Sprintf("callback-owner-released-refs-%d", refs))
      }()`],
    ["\tif err := controller.Close(); err != nil {", `\tcloseResult, _, _ := controller.vtbl.Close.Call(uintptr(unsafe.Pointer(controller)))
        e.markShutdown(fmt.Sprintf("controller-close-hresult-%08x", closeResult))
        if err := hresult(closeResult); err != nil {`],
  ]);
  // Extra points belong to diagnostic-only output, never the published lifecycle schema.
  await overlay("internal/benchmarker/timeline.go", [["velox.host-startup-timeline/v1", "velox.host-controller-startup-diagnostic/v1"]]);
  await overlay("internal/benchmarker/shutdown_timeline.go", [["velox.host-shutdown-timeline/v1", "velox.host-controller-shutdown-diagnostic/v1"]]);
  await overlay("tests/startup/lifecycle_evidence_windows_test.go", [["velox.startup-lifecycle/v3", "velox.startup-controller-diagnostic/v1"]]);
  const overlayPath = join(work, "overlay.json"), host = join(work, "velox-host.exe");
  await writeFile(overlayPath, JSON.stringify({ Replace: overlays }, null, 2));
  await writeFile(join(work, "build.log"), command("go", ["build", "-mod=readonly", "-overlay=" + overlayPath, "-trimpath", "-ldflags=-s -w -H windowsgui", "-o", host, "./cmd/velox-host"]));
  result.hostSHA256 = hash(await readFile(host));
  const evidencePath = join(work, "evidence.json");
  await writeFile(join(work, "native.log"), command("go", ["test", "-mod=readonly", "-overlay=" + overlayPath, "-v", "-count=1", "-timeout=60s", "-run=^TestStartupLifecycleEvidence$", "./tests/startup"], 75_000, {
    VELOX_BUILT_HOST: host, VELOX_STARTUP_LIFECYCLE_RESULT: evidencePath,
    VELOX_STARTUP_LIFECYCLE_REPETITIONS: "1",
  }));
  const evidenceBytes = await readFile(evidencePath);
  result.evidenceSHA256 = hash(evidenceBytes);
  const evidence = JSON.parse(evidenceBytes.toString("utf8"));
  if (evidence.schemaVersion !== "velox.startup-controller-diagnostic/v1" || evidence.outcome !== "success" || evidence.samples.length !== 1 || evidence.samples[0].outcome !== "success") throw Error("Incomplete native evidence");
  result.environment = evidence.environment;
  result.first = summarize(evidence.samples[0].first);
  result.immediate = summarize(evidence.samples[0].immediate);
  result.timeline = evidence.samples[0].timeline;
  if (hash(await readFile(host)) !== result.hostSHA256) throw Error("Host changed during execution");
  result.outcome = "success";
} catch (error) {
  result.error = String(error);
  process.exitCode = 1;
} finally {
  for (const [path, expected] of Object.entries(originals)) {
    if (hash(await readFile(join(root, path))) !== expected) {
      result.outcome = "failure";
      result.sourceDrift = path;
      process.exitCode = 1;
    }
  }
  result.originals = originals;
  result.overlays = replacements;
  result.finishedAtUtc = new Date().toISOString();
  await writeFile(join(work, "result.json"), JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));
  console.log("HOST_PHASE_EVIDENCE=" + work);
}
