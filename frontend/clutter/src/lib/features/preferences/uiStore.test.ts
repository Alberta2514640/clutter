import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clampInspectorWidth, inspectorWidthFrom, INSPECTOR_WIDTH, viewFrom } from "./uiStore";

// A browser's localStorage, in memory, so the store persists and hydrates as
// it does in the app.
function memoryStorage(seed: Record<string, string> = {}): Storage {
  const m = new Map(Object.entries(seed));
  return {
    get length() {
      return m.size;
    },
    clear: () => m.clear(),
    getItem: (k) => m.get(k) ?? null,
    key: (i) => [...m.keys()][i] ?? null,
    removeItem: (k) => void m.delete(k),
    setItem: (k, v) => void m.set(k, String(v)),
  };
}

// A fresh store, hydrated from the given storage, as on a page load.
async function loadStore(storage: Storage) {
  vi.stubGlobal("localStorage", storage);
  vi.stubGlobal("window", { localStorage: storage });
  vi.resetModules();
  return (await import("./uiStore")).usePreferencesStore;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("inspector width", () => {
  it("reads only a width in range", () => {
    expect(inspectorWidthFrom("520")).toBe(520);
    expect(inspectorWidthFrom(600)).toBe(600);
    expect(inspectorWidthFrom("100")).toBeNull();
    expect(inspectorWidthFrom("9000")).toBeNull();
    expect(inspectorWidthFrom("wide")).toBeNull();
    expect(inspectorWidthFrom("")).toBeNull();
    expect(inspectorWidthFrom(null)).toBeNull();
  });

  it("keeps a chosen width inside the bounds", () => {
    expect(clampInspectorWidth(100)).toBe(INSPECTOR_WIDTH.min);
    expect(clampInspectorWidth(9000)).toBe(INSPECTOR_WIDTH.max);
    expect(clampInspectorWidth(500.4)).toBe(500);
  });

  it("moves the old key's width into the store once, then removes the key", async () => {
    const storage = memoryStorage({ "clutter.inspector-width": "520" });
    const store = await loadStore(storage);
    expect(store.getState().inspectorWidth).toBe(520);
    expect(storage.getItem("clutter.inspector-width")).toBeNull();
    expect(JSON.parse(storage.getItem("clutter-preferences") ?? "{}").state.inspectorWidth).toBe(520);
  });

  it("keeps the store's own width over the old key, and still removes the key", async () => {
    const storage = memoryStorage({
      "clutter.inspector-width": "520",
      "clutter-preferences": JSON.stringify({ state: { inspectorWidth: 600 }, version: 0 }),
    });
    const store = await loadStore(storage);
    expect(store.getState().inspectorWidth).toBe(600);
    expect(storage.getItem("clutter.inspector-width")).toBeNull();
  });

  it("drops an old key that holds no usable width", async () => {
    const storage = memoryStorage({ "clutter.inspector-width": "huge" });
    const store = await loadStore(storage);
    expect(store.getState().inspectorWidth).toBeNull();
    expect(storage.getItem("clutter.inspector-width")).toBeNull();
  });
});

describe("View as, by diagram", () => {
  it("reads only a short non-empty view", () => {
    expect(viewFrom("general-openstack")).toBe("general-openstack");
    expect(viewFrom("")).toBeNull();
    expect(viewFrom("  ")).toBeNull();
    expect(viewFrom("x".repeat(65))).toBeNull();
    expect(viewFrom(3)).toBeNull();
  });

  it("keeps each diagram's choice in the store's one key", async () => {
    const storage = memoryStorage();
    const store = await loadStore(storage);
    store.getState().setBlocksView("d-1", "aws");
    store.getState().setBlocksView("d-2", "blocks");
    store.getState().setBlocksView("d-2", "");
    expect(store.getState().blocksView).toEqual({ "d-1": "aws", "d-2": "blocks" });
    expect(JSON.parse(storage.getItem("clutter-preferences") ?? "{}").state.blocksView).toEqual({ "d-1": "aws", "d-2": "blocks" });
    expect(storage.length).toBe(1);
  });

  it("moves each old per-diagram key into the store once, then removes the keys", async () => {
    const storage = memoryStorage({
      "clutter.blocks-view.d-1": "aws",
      "clutter.blocks-view.d-2": "mock",
      "clutter.blocks-view.d-3": "",
      "clutter-theme": "dark",
    });
    const store = await loadStore(storage);
    expect(store.getState().blocksView).toEqual({ "d-1": "aws", "d-2": "mock" });
    expect(storage.getItem("clutter.blocks-view.d-1")).toBeNull();
    expect(storage.getItem("clutter.blocks-view.d-2")).toBeNull();
    expect(storage.getItem("clutter.blocks-view.d-3")).toBeNull();
    expect(storage.getItem("clutter-theme")).toBe("dark");
    expect(JSON.parse(storage.getItem("clutter-preferences") ?? "{}").state.blocksView).toEqual({ "d-1": "aws", "d-2": "mock" });
  });

  it("keeps the store's own view over an old key, and still removes the key", async () => {
    const storage = memoryStorage({
      "clutter.blocks-view.d-1": "aws",
      "clutter-preferences": JSON.stringify({ state: { blocksView: { "d-1": "blocks" } }, version: 0 }),
    });
    const store = await loadStore(storage);
    expect(store.getState().blocksView).toEqual({ "d-1": "blocks" });
    expect(storage.getItem("clutter.blocks-view.d-1")).toBeNull();
  });
});

describe("forgetUserData", () => {
  let storage: Storage;
  beforeEach(() => {
    storage = memoryStorage({ "clutter-theme": "light" });
  });

  it("clears the signed-in person's team data and keeps the onboarding answers, panel width and theme", async () => {
    const store = await loadStore(storage);
    const s = store.getState();
    s.setExperience("terraform");
    s.setFirstCloud("aws");
    s.setInspectorWidth(500);
    s.rememberDiagram("org-1", { projectId: "p", diagramId: "d", name: "web" });
    s.dismissChecklist("org-1");
    s.setChecklistCollapsed("org-1", true);
    s.markPreviewOpened("org-1");
    s.setBlocksView("d-1", "aws");

    store.getState().forgetUserData();
    const after = store.getState();
    expect(after.blocksView).toEqual({});
    expect(after.lastDiagram).toEqual({});
    expect(after.checklistDismissed).toEqual({});
    expect(after.checklistCollapsed).toEqual({});
    expect(after.previewOpened).toEqual({});
    expect(after.experience).toBe("terraform");
    expect(after.firstCloud).toBe("aws");
    expect(after.inspectorWidth).toBe(500);
    expect(storage.getItem("clutter-theme")).toBe("light");
    expect(JSON.parse(storage.getItem("clutter-preferences") ?? "{}").state.lastDiagram).toEqual({});
  });
});
