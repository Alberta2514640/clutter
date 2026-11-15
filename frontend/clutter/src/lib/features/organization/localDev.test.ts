import { describe, expect, it } from "vitest";
import { localDevApi } from "./localDev";

describe("localDevApi", () => {
  it("is true for a development build on the local gateway", () => {
    for (const api of ["http://localhost:8787", "http://127.0.0.1:8787", "http://localhost:8787/", "http://localhost"]) {
      expect(localDevApi("development", api)).toBe(true);
    }
  });

  it("is never true for a production or test build, even on the local gateway", () => {
    for (const env of ["production", "test", undefined]) {
      expect(localDevApi(env, "http://localhost:8787")).toBe(false);
      expect(localDevApi(env, "http://127.0.0.1:8787")).toBe(false);
    }
  });

  it("is false for any API that is not plain http on loopback", () => {
    for (const api of [
      undefined,
      "",
      "https://localhost:8787",
      "https://api.clutter.example",
      "http://api.clutter.example",
      "http://localhost.evil.example",
      "http://127.0.0.1.evil.example:8787",
      "http://10.0.0.5:8787",
      "http://localhost:8787/prod",
    ]) {
      expect(localDevApi("development", api)).toBe(false);
    }
  });
});
