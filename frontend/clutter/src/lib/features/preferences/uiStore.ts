// lib/features/preferences/uiStore.ts
import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

// What a viewer told onboarding and where they last worked, kept in this
// browser only and never sent to the API. Each value only changes defaults:
// a cleared browser falls back to the middle experience level, no first
// cloud, and no "continue where you left off".

export type Experience = "new" | "console" | "terraform";

// A connector id from B's catalog ("aws", "general-openstack"),
// NOT_SURE_CLOUD for "Not sure yet", or null when never answered. Every
// reader checks the value against the clouds it offers, so "Not sure yet"
// reads as no preference.
export type FirstCloud = string | null;

export const NOT_SURE_CLOUD = "not-sure";

export type LastDiagram = { projectId: string; diagramId: string; name: string; openedAt: string };

// The building-block editor's side panel width, in pixels: at least 380, 440
// to start, at most 760, and 24 per arrow key press.
export const INSPECTOR_WIDTH = { min: 380, start: 440, max: 760, step: 24 } as const;

export const clampInspectorWidth = (w: number) => Math.round(Math.min(INSPECTOR_WIDTH.max, Math.max(INSPECTOR_WIDTH.min, w)));

// The key the editor kept the width under before it moved into this store.
// Read once, moved here, then removed.
const LEGACY_INSPECTOR_KEY = "clutter.inspector-width";

// The building-block editor kept each diagram's "View as" choice under its
// own key (this prefix and the diagram id) before it moved into this store.
// Each is read once, moved here, then removed.
const LEGACY_VIEW_PREFIX = "clutter.blocks-view.";

// The longest view value kept: a connector id ("general-openstack") or
// "blocks". resolveView in deployments/views.ts still decides what shows.
const MAX_VIEW_LENGTH = 64;

// viewFrom reads a stored "View as" choice: a short non-empty string, or null.
export function viewFrom(raw: unknown): string | null {
  return typeof raw === "string" && raw.trim() !== "" && raw.length <= MAX_VIEW_LENGTH ? raw : null;
}

// takeLegacyViews reads every old per-diagram key that holds a usable view,
// by diagram id, and removes all the old keys.
function takeLegacyViews(storage: Storage): Record<string, string> {
  const keys: string[] = [];
  for (let i = 0; i < storage.length; i++) {
    const k = storage.key(i);
    if (k?.startsWith(LEGACY_VIEW_PREFIX)) keys.push(k);
  }
  const views: Record<string, string> = {};
  for (const k of keys) {
    const diagramId = k.slice(LEGACY_VIEW_PREFIX.length);
    const view = viewFrom(storage.getItem(k));
    if (diagramId && view) views[diagramId] = view;
    storage.removeItem(k);
  }
  return views;
}

// inspectorWidthFrom reads a stored width: a number in range, or null for
// "start width".
export function inspectorWidthFrom(raw: unknown): number | null {
  const n = typeof raw === "number" ? raw : typeof raw === "string" && raw.trim() !== "" ? Number(raw) : NaN;
  return Number.isFinite(n) && n >= INSPECTOR_WIDTH.min && n <= INSPECTOR_WIDTH.max ? Math.round(n) : null;
}

type PreferencesState = {
  experience: Experience | null;
  firstCloud: FirstCloud;
  // Keyed by organization id.
  lastDiagram: Record<string, LastDiagram>;
  // Organization ids whose first-deploy checklist the viewer closed, or saw
  // finish with a first live deployment (it never comes back after either).
  checklistDismissed: Record<string, true>;
  // Whether the viewer folded the checklist to its pill, by organization id.
  // Missing means the experience level decides (checklistStartsOpen).
  checklistCollapsed: Record<string, boolean>;
  // Organization ids where the viewer opened a Terraform preview.
  previewOpened: Record<string, true>;
  // The editor's side panel width this viewer chose; null is the start width.
  inspectorWidth: number | null;
  // The building-block editor's "View as" choice, by diagram id. Missing
  // means the cloud chosen at onboarding, else the plain blocks view.
  blocksView: Record<string, string>;
  // Whether this browser has shown the small logo flourish for a deployment
  // that turned Live. It plays once per browser, so it stays on sign-out.
  firstLiveShown: boolean;
  setExperience: (e: Experience | null) => void;
  setFirstCloud: (c: FirstCloud) => void;
  rememberDiagram: (organizationId: string, d: Omit<LastDiagram, "openedAt">) => void;
  dismissChecklist: (organizationId: string) => void;
  setChecklistCollapsed: (organizationId: string, collapsed: boolean) => void;
  markPreviewOpened: (organizationId: string) => void;
  setInspectorWidth: (w: number) => void;
  setBlocksView: (diagramId: string, view: string) => void;
  markFirstLiveShown: () => void;
  // Sign-out forgets what belongs to the signed-in person's teams (the last
  // diagram, the checklist's state, the preview flag, each diagram's view),
  // so the next person in this browser does not see it. The onboarding
  // answers and the panel width stay; the theme is kept under its own key
  // (components/common/theme.ts).
  forgetUserData: () => void;
};

// What forgetUserData clears.
const NO_USER_DATA = { lastDiagram: {}, checklistDismissed: {}, checklistCollapsed: {}, previewOpened: {}, blocksView: {} } as const;

export const usePreferencesStore = create<PreferencesState>()(
  persist(
    (set) => ({
      experience: null,
      firstCloud: null,
      ...NO_USER_DATA,
      inspectorWidth: null,
      firstLiveShown: false,
      setExperience: (experience) => set({ experience }),
      setFirstCloud: (firstCloud) => set({ firstCloud }),
      rememberDiagram: (organizationId, d) =>
        set((s) => ({ lastDiagram: { ...s.lastDiagram, [organizationId]: { ...d, openedAt: new Date().toISOString() } } })),
      dismissChecklist: (organizationId) =>
        set((s) => ({ checklistDismissed: { ...s.checklistDismissed, [organizationId]: true } })),
      setChecklistCollapsed: (organizationId, collapsed) =>
        set((s) => ({ checklistCollapsed: { ...s.checklistCollapsed, [organizationId]: collapsed } })),
      markPreviewOpened: (organizationId) =>
        set((s) => ({ previewOpened: { ...s.previewOpened, [organizationId]: true } })),
      setInspectorWidth: (w) => set({ inspectorWidth: clampInspectorWidth(w) }),
      setBlocksView: (diagramId, view) => {
        const v = viewFrom(view);
        if (v) set((s) => ({ blocksView: { ...s.blocksView, [diagramId]: v } }));
      },
      markFirstLiveShown: () => set({ firstLiveShown: true }),
      forgetUserData: () => set({ ...NO_USER_DATA }),
    }),
    {
      name: "clutter-preferences",
      storage: createJSONStorage(() => localStorage),
      partialize: (s) => ({
        experience: s.experience,
        firstCloud: s.firstCloud,
        lastDiagram: s.lastDiagram,
        checklistDismissed: s.checklistDismissed,
        checklistCollapsed: s.checklistCollapsed,
        previewOpened: s.previewOpened,
        inspectorWidth: s.inspectorWidth,
        blocksView: s.blocksView,
        firstLiveShown: s.firstLiveShown,
      }),
      // A width or a view kept under the editor's old keys moves here once;
      // the old keys go either way. Storage can be blocked (a private
      // window): then there is nothing to move.
      onRehydrateStorage: () => (state) => {
        try {
          const storage = window.localStorage;
          const legacy = storage.getItem(LEGACY_INSPECTOR_KEY);
          if (legacy !== null) {
            const width = inspectorWidthFrom(legacy);
            if (state && state.inspectorWidth === null && width !== null) state.setInspectorWidth(width);
            storage.removeItem(LEGACY_INSPECTOR_KEY);
          }
          // A view the store already holds for a diagram wins over its old key.
          for (const [diagramId, view] of Object.entries(takeLegacyViews(storage))) {
            if (state && !(diagramId in state.blocksView)) state.setBlocksView(diagramId, view);
          }
        } catch {
          // Storage is blocked; the start width and default views apply.
        }
      },
    },
  ),
);

// The level that drives defaults: skipped onboarding means "console".
export function effectiveExperience(e: Experience | null): Experience {
  return e ?? "console";
}

// Whether the first-deploy checklist shows its steps: the viewer's own choice
// for this organization, else open for new and console users and folded for
// Terraform users.
export function checklistStartsOpen(collapsed: boolean | undefined, e: Experience | null): boolean {
  return collapsed === undefined ? effectiveExperience(e) !== "terraform" : !collapsed;
}
