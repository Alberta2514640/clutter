import { describe, expect, it } from "vitest";
import { formatBytes, shortHash, uploadReducer, zipFileError, type UploadState } from "./upload";
import { MAX_CODE_BYTES } from "./types";

describe("zipFileError", () => {
  it("takes a .zip of at most 50 MB", () => {
    expect(zipFileError({ name: "code.zip", size: 1024 })).toBeNull();
    expect(zipFileError({ name: "Function.ZIP", size: MAX_CODE_BYTES })).toBeNull();
  });

  it("refuses what B would refuse", () => {
    expect(zipFileError({ name: "code.tar.gz", size: 10 })).toBe("Choose a .zip file.");
    expect(zipFileError({ name: ".zip", size: 10 })).toBe("Choose a .zip file.");
    expect(zipFileError({ name: "code.zip", size: 0 })).toBe("The file is empty.");
    expect(zipFileError({ name: "code.zip", size: MAX_CODE_BYTES + 1 })).toBe("The file is 50.0 MB. A function's code can be at most 50 MB.");
    expect(zipFileError({ name: "a\nb.zip", size: 10 })).toMatch(/^Rename the file/);
    expect(zipFileError({ name: "x".repeat(252) + ".zip", size: 10 })).toMatch(/^Rename the file/);
  });
});

describe("formatBytes and shortHash", () => {
  it("shows sizes and hashes for people", () => {
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(2048)).toBe("2.0 KB");
    expect(formatBytes(5 * 1024 * 1024 + 1)).toBe("5.0 MB");
    expect(shortHash("0123456789abcdef".repeat(4))).toBe("0123456789ab");
  });
});

describe("uploadReducer", () => {
  const idle: UploadState = { phase: "idle" };

  it("goes from idle through progress to idle when B has stored the file", () => {
    let s = uploadReducer(idle, { type: "start", fileName: "code.zip" });
    expect(s).toEqual({ phase: "uploading", fileName: "code.zip", percent: 0 });
    s = uploadReducer(s, { type: "progress", loaded: 50, total: 100 });
    expect(s).toEqual({ phase: "uploading", fileName: "code.zip", percent: 50 });
    // Every byte sent is not yet stored: the bar waits at 99 for B's answer.
    s = uploadReducer(s, { type: "progress", loaded: 100, total: 100 });
    expect(s).toMatchObject({ percent: 99 });
    // Progress never goes back.
    s = uploadReducer(s, { type: "progress", loaded: 10, total: 100 });
    expect(s).toMatchObject({ percent: 99 });
    expect(uploadReducer(s, { type: "done" })).toEqual(idle);
  });

  it("shows a failure until it is dismissed or a new upload starts", () => {
    const failed = uploadReducer({ phase: "uploading", fileName: "a.zip", percent: 40 }, { type: "fail", message: "The file is not a .zip archive." });
    expect(failed).toEqual({ phase: "failed", message: "The file is not a .zip archive." });
    expect(uploadReducer(failed, { type: "progress", loaded: 1, total: 2 })).toBe(failed);
    expect(uploadReducer(failed, { type: "dismiss" })).toEqual(idle);
    expect(uploadReducer(failed, { type: "start", fileName: "b.zip" })).toMatchObject({ phase: "uploading", fileName: "b.zip" });
  });

  it("ignores progress with no total and dismiss while uploading", () => {
    const up: UploadState = { phase: "uploading", fileName: "a.zip", percent: 10 };
    expect(uploadReducer(up, { type: "progress", loaded: 5, total: 0 })).toBe(up);
    expect(uploadReducer(up, { type: "dismiss" })).toBe(up);
  });
});
