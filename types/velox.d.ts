/** Declaration-only IPC v1 surface; native validation and permissions remain authoritative. */
export type EmptyParams = Record<string, never>;
export type WindowState = "normal" | "minimized" | "maximized";
export interface AppInfo {
  id: string;
  name: string;
  version: string;
  platform: string;
}

export interface TextFileResult {
  cancelled: boolean;
  name: string;
  text: string;
  bytes: number;
}
export interface SaveResult {
  cancelled: boolean;
  name: string;
  bytes: number;
}
/** Targets are document-scoped positive uint32 values, not paths or persisted grants. */
export type ConnectedSaveResult =
  | (SaveResult & { cancelled: true; target?: never })
  | (SaveResult & { cancelled: false; target: number });
export type ClipboardReadResult =
  | { cancelled: true; text?: never }
  | { cancelled: false; text: string };
export interface FolderResult {
  cancelled: boolean;
  name: string;
  target: number;
}
export interface FolderListing {
  entries: { name: string; kind: "file" | "directory" }[];
  truncated: boolean;
  skipped: number;
}
export type ProgressParams =
  | { state: "none" | "indeterminate"; value?: never }
  | { state: "normal" | "error" | "paused"; value: number };

export type MethodMap = {
  "app.getInfo": { params: EmptyParams; result: AppInfo };
  "window.getState": { params: EmptyParams; result: WindowState };
  "window.minimize": { params: EmptyParams; result: null };
  "window.maximize": { params: EmptyParams; result: null };
  "window.restore": { params: EmptyParams; result: null };
  "window.close": { params: EmptyParams; result: null };
  "window.setTitle": { params: { title: string }; result: null };
  "window.requestAttention": { params: { count?: 1 | 2 | 3 | 4 | 5 }; result: null };
  "window.cancelAttention": { params: EmptyParams; result: null };
  "window.setProgress": { params: ProgressParams; result: null };
  "notification.show": { params: { kind: "info" | "warning" | "error"; message: string }; result: null };
  "external.open": { params: { url: string }; result: { queued: true } };
  "clipboard.writeText": { params: { text: string }; result: null };
  "clipboard.readText": { params: EmptyParams; result: ClipboardReadResult };
  "file.openText": { params: EmptyParams; result: TextFileResult };
  "file.beginSave": { params: { name: string; bytes: number }; result: { token: number } };
  "file.appendSave": { params: { token: number; offset: number; text: string }; result: { bytes: number } };
  "file.commitSave": { params: { token: number }; result: SaveResult };
  "file.cancelSave": { params: { token: number }; result: null };
  "file.commitSaveAs": { params: { token: number }; result: ConnectedSaveResult };
  "file.commitSaveTo": { params: { token: number; target: number }; result: ConnectedSaveResult };
  "file.releaseSaveTarget": { params: { target: number }; result: null };
  "folder.select": { params: EmptyParams; result: FolderResult };
  "folder.release": { params: { target: number }; result: null };
  "folder.list": { params: { target: number }; result: FolderListing };
  "folder.openText": { params: { target: number; name: string }; result: TextFileResult };
}

export type Method = keyof MethodMap;
export type Params<M extends Method> = MethodMap[M]["params"];
export type Result<M extends Method> = MethodMap[M]["result"];
export type InvokeArgs<M extends Method = Method> = {
  [K in M]: {} extends Params<K>
    ? [method: K, params?: Params<K>]
    : [method: K, params: Params<K>];
}[M];
type UnionToIntersection<U> =
  (U extends unknown ? (value: U) => void : never) extends
  (value: infer I) => void ? I : never;
// One overload per method prevents mismatched union-method/parameter pairs.
export type Invoke = UnionToIntersection<{
  [K in Method]: (...args: InvokeArgs<K>) => Promise<Result<K>>;
}[Method]>;

export type ErrorCode =
  | "INVALID_REQUEST" | "INVALID_PARAMS" | "METHOD_NOT_FOUND"
  | "PERMISSION_DENIED" | "PAYLOAD_TOO_LARGE" | "TOO_MANY_REQUESTS"
  | "DUPLICATE_REQUEST_ID" | "UNSUPPORTED_VERSION" | "SHUTTING_DOWN"
  | "NATIVE_OPERATION_FAILED" | "UNSUPPORTED_FILE" | "SAVE_RECOVERY_REQUIRED"
  | "FILE_CHANGED" | "SAVE_TARGET_INVALID" | "UNSUPPORTED_FOLDER"
  | "FOLDER_TARGET_INVALID" | "CLIPBOARD_BUSY" | "UNSUPPORTED_TEXT"
  | "INVALID_RESPONSE" | "INTERNAL";
/** Rejections are Error objects with a code; catch values still require a runtime check. */
export interface VeloxError extends Error {
  readonly code: ErrorCode;
}

export interface VeloxAPI {
  readonly invoke: Invoke;
  saveText(text: string, name?: string): Promise<SaveResult>;
  saveTextAs(text: string, name?: string): Promise<ConnectedSaveResult>;
  saveTextTo(text: string, target: number): Promise<ConnectedSaveResult>;
}

declare global {
  interface Window {
    /** Only injected into a trusted top-level Velox document; absent in browser previews. */
    readonly velox?: Readonly<VeloxAPI>;
  }
}
