// The upload control's rules and state, kept apart from React so Vitest can
// check them. B checks every upload again (backend/adapter/internal/codestore).

import { MAX_CODE_BYTES, MAX_CODE_SIZE_TEXT } from "./types";

// zipFileError says why a chosen file cannot be uploaded, or null. It mirrors
// B's file name rule (codestore.ValidFileName) and size cap.
export function zipFileError(file: { name: string; size: number }): string | null {
  const name = file.name;
  if (!name.toLowerCase().endsWith(".zip") || name.length < 5) return "Choose a .zip file.";
  if (name.length > 255 || /[/\\\u0000-\u001f\u007f]/.test(name)) return "Rename the file: its name must be plain text of at most 255 characters.";
  if (file.size === 0) return "The file is empty.";
  if (file.size > MAX_CODE_BYTES) return `The file is ${formatBytes(file.size)}. A function's code can be at most ${MAX_CODE_SIZE_TEXT}.`;
  return null;
}

// formatBytes shows a size for people: bytes, KB or MB (1024-based).
export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

// shortHash is the start of a SHA-256, enough to tell two uploads apart.
export const shortHash = (sha: string) => sha.slice(0, 12);

export type UploadState =
  | { phase: "idle" }
  | { phase: "uploading"; fileName: string; percent: number }
  | { phase: "failed"; message: string };

export type UploadEvent =
  | { type: "start"; fileName: string }
  | { type: "progress"; loaded: number; total: number }
  | { type: "done" }
  | { type: "fail"; message: string }
  | { type: "dismiss" };

// uploadReducer moves the control between idle, uploading and failed. A
// finished upload returns to idle: the stored file then shows from B's list.
export function uploadReducer(state: UploadState, ev: UploadEvent): UploadState {
  switch (ev.type) {
    case "start":
      return { phase: "uploading", fileName: ev.fileName, percent: 0 };
    case "progress":
      if (state.phase !== "uploading" || ev.total <= 0) return state;
      // 100% only once B has checked and stored the file ("done").
      return { ...state, percent: Math.min(99, Math.max(state.percent, Math.floor((ev.loaded / ev.total) * 100))) };
    case "done":
      return { phase: "idle" };
    case "fail":
      return { phase: "failed", message: ev.message };
    case "dismiss":
      return state.phase === "failed" ? { phase: "idle" } : state;
  }
}
