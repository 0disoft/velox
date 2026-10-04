import type {
  AppInfo, ClipboardReadResult, ConnectedSaveResult, FolderListing, FolderResult,
  Method, MethodMap, Result, SaveResult, TextFileResult, VeloxAPI, VeloxError, WindowState,
} from "../../types/velox";

declare const api: VeloxAPI;
function expectType<T>(_value: T): void {}
type Equal<A, B> = (<T>() => T extends A ? 1 : 2) extends
  (<T>() => T extends B ? 1 : 2) ? true : false;
type Assert<T extends true> = T;
type InfoInference = Assert<Equal<ReturnType<typeof infoCall>, Promise<AppInfo>>>;
function infoCall() { return api.invoke("app.getInfo"); }

expectType<Promise<AppInfo>>(api.invoke("app.getInfo", {}));
expectType<Promise<WindowState>>(api.invoke("window.getState"));
expectType<Promise<null>>(api.invoke("window.minimize"));
expectType<Promise<null>>(api.invoke("window.maximize"));
expectType<Promise<null>>(api.invoke("window.restore"));
expectType<Promise<null>>(api.invoke("window.close"));
expectType<Promise<null>>(api.invoke("window.setTitle", { title: "Note" }));
expectType<Promise<null>>(api.invoke("window.requestAttention"));
api.invoke("window.requestAttention", { count: 5 });
api.invoke("window.cancelAttention", {});
api.invoke("window.setProgress", { state: "none" });
api.invoke("window.setProgress", { state: "indeterminate" });
api.invoke("window.setProgress", { state: "paused", value: 50 });
api.invoke("notification.show", { kind: "info", message: "Done" });
expectType<Promise<{ queued: true }>>(api.invoke("external.open", { url: "https://example.com/" }));
api.invoke("clipboard.writeText", { text: "Note" });
expectType<Promise<ClipboardReadResult>>(api.invoke("clipboard.readText"));
expectType<Promise<TextFileResult>>(api.invoke("file.openText"));
expectType<Promise<{ token: number }>>(api.invoke("file.beginSave", { name: "note.txt", bytes: 4 }));
expectType<Promise<{ bytes: number }>>(api.invoke("file.appendSave", { token: 1, offset: 0, text: "Note" }));
expectType<Promise<SaveResult>>(api.invoke("file.commitSave", { token: 1 }));
api.invoke("file.cancelSave", { token: 1 });
expectType<Promise<ConnectedSaveResult>>(api.invoke("file.commitSaveAs", { token: 1 }));
expectType<Promise<ConnectedSaveResult>>(api.invoke("file.commitSaveTo", { token: 1, target: 2 }));
api.invoke("file.releaseSaveTarget", { target: 2 });
expectType<Promise<FolderResult>>(api.invoke("folder.select"));
api.invoke("folder.release", { target: 1 });
expectType<Promise<FolderListing>>(api.invoke("folder.list", { target: 1 }));
expectType<Promise<TextFileResult>>(api.invoke("folder.openText", { target: 1, name: "note.txt" }));
expectType<Promise<SaveResult>>(api.saveText("Note"));
expectType<Promise<ConnectedSaveResult>>(api.saveTextAs("Note", "note.txt"));
expectType<Promise<ConnectedSaveResult>>(api.saveTextTo("Note", 1));

async function narrowing() {
  const saved = await api.saveTextAs("Note");
  if (!saved.cancelled) expectType<number>(saved.target);
  const clipboard = await api.invoke("clipboard.readText");
  if (!clipboard.cancelled) expectType<string>(clipboard.text);
  // @ts-expect-error Clipboard text is not available before cancellation narrowing.
  expectType<string>(clipboard.text);
}
// @ts-expect-error The bridge may be absent in a browser preview.
window.velox.invoke("app.getInfo");
if (window.velox) expectType<Promise<AppInfo>>(window.velox.invoke("app.getInfo"));
// @ts-expect-error The injected bridge is readonly.
window.velox = api;
// @ts-expect-error No unsupported method fallback.
api.invoke("shell.execute", { command: "example" });
// @ts-expect-error Required parameters cannot be omitted.
api.invoke("window.setTitle");
// @ts-expect-error Wrong parameter type.
api.invoke("window.setTitle", { title: 42 });
// @ts-expect-error Empty-parameter methods reject extra keys.
api.invoke("window.close", { force: true });
// @ts-expect-error Native count range is closed.
api.invoke("window.requestAttention", { count: 6 });
// @ts-expect-error Determinate progress requires value.
api.invoke("window.setProgress", { state: "normal" });
// @ts-expect-error Indeterminate progress does not accept value.
api.invoke("window.setProgress", { state: "indeterminate", value: 50 });
// @ts-expect-error Unsupported notification kind.
api.invoke("notification.show", { kind: "success", message: "Done" });
// @ts-expect-error Tokens are numbers, not file paths.
api.saveTextTo("Note", "note.txt");
// @ts-expect-error Unknown fields in object literals are rejected.
api.invoke("external.open", { url: "https://example.com/", silent: true });
declare const dynamicMethod: "window.setTitle" | "clipboard.writeText";
// @ts-expect-error A union method cannot be paired with parameters for only one member.
api.invoke(dynamicMethod, { title: "Note" });
declare const error: VeloxError;
// @ts-expect-error Native error codes are readonly.
error.code = "INTERNAL";

// Every public method is also exercised by a call above.
export type AllMethods = Method;
export type AllResults = { [K in Method]: Result<K> };
export type Contract = MethodMap;
