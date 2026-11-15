import { describe, expect, it } from "vitest";
import { loadConfig } from "../src/config.ts";

describe("loadConfig", () => {
  it("uses the local defaults", () => {
    const cfg = loadConfig({});
    expect(cfg.host).toBe("127.0.0.1");
    expect(cfg.port).toBe(8791);
    expect(cfg.apiUrl).toBe("http://localhost:8787");
    expect(cfg.allowedOrigins).toEqual(["http://localhost:3000"]);
  });

  it("reads several origins and trims paths", () => {
    const cfg = loadConfig({ COLLAB_ALLOWED_ORIGINS: "http://localhost:3000/, https://clutter.example/app" });
    expect(cfg.allowedOrigins).toEqual(["http://localhost:3000", "https://clutter.example"]);
  });

  it("refuses a host that is not this machine", () => {
    expect(() => loadConfig({ COLLAB_HOST: "0.0.0.0" })).toThrow(/COLLAB_HOST/);
  });

  it.each([
    ["COLLAB_PORT", "abc"],
    ["COLLAB_PORT", "0"],
    ["COLLAB_PORT", "70000"],
    ["COLLAB_MAX_ROOMS", "1.5"],
    ["COLLAB_API_URL", "not a url"],
    ["COLLAB_API_URL", "ftp://localhost"],
    ["COLLAB_ALLOWED_ORIGINS", " , "],
  ])("refuses %s=%s", (name, value) => {
    expect(() => loadConfig({ [name]: value })).toThrow(new RegExp(name));
  });
});
