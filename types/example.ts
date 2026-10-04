import type { AppInfo } from "./velox";

// Call from an app action; the manifest must grant app.info.
export async function applicationInfo(): Promise<AppInfo | undefined> {
  return window.velox?.invoke("app.getInfo");
}

export type SaveOutcome =
  | { kind: "saved"; name: string; target: number }
  | { kind: "cancelled" }
  | { kind: "error"; message: string };

// Requires file.save. Keep the editor buffer for cancelled/error outcomes.
// Reuse a target only within its original document, never from persisted data.
export async function saveNote(text: string, target?: number): Promise<SaveOutcome> {
  const api = window.velox;
  if (!api) return { kind: "error", message: "Native host unavailable." };
  try {
    const saved = target === undefined
      ? await api.saveTextAs(text, "note.txt")
      : await api.saveTextTo(text, target);
    if (saved.cancelled) return { kind: "cancelled" };
    return { kind: "saved", name: saved.name, target: saved.target };
  } catch (error: unknown) {
    if (error instanceof Error && "code" in error) {
      if (error.code === "PERMISSION_DENIED") {
        return { kind: "error", message: "File-save permission is not granted." };
      }
      if (error.code === "FILE_CHANGED" || error.code === "SAVE_TARGET_INVALID") {
        return { kind: "error", message: "Keep your text and use Save as." };
      }
    }
    return { kind: "error", message: "Save failed; keep your text." };
  }
}
