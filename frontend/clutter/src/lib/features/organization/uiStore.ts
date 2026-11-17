// lib/features/organization/uiStore.ts
import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";
import type { Organization } from "./types";

// The organization this viewer is working in, kept in this browser only. It
// is a convenience: useCurrentOrganization falls back to the first
// organization when it is unset, unreadable or no longer one of the user's.
type CurrentOrgState = {
  organizationId: string | null;
  select: (organizationId: string) => void;
};

export const useCurrentOrganizationStore = create<CurrentOrgState>()(
  persist(
    (set) => ({
      organizationId: null,
      select: (organizationId) => set({ organizationId }),
    }),
    {
      name: "clutter-current-organization",
      storage: createJSONStorage(() => localStorage),
      partialize: (s) => ({ organizationId: s.organizationId }),
    },
  ),
);

type OrgUiState = {
  isEditOpen: boolean;
  draft: { description?: string };
};

type OrgUiActions = {
  open: (org?: Organization | null) => void;
  close: () => void;
  patch: (p: { description?: string }) => void;
  reset: () => void;
};

export const useOrganizationUiStore = create<{ state: OrgUiState; actions: OrgUiActions }>((set) => ({
  state: { isEditOpen: false, draft: {} },
  actions: {
    open: (org) =>
      set((s) => ({
        state: {
          ...s.state,
          isEditOpen: true,
          draft: org ? { description: org.description ?? "" } : {},
        },
      })),
    close: () => set((s) => ({ state: { ...s.state, isEditOpen: false, draft: {} } })),
    patch: (p) => set((s) => ({ state: { ...s.state, draft: { ...s.state.draft, ...p } } })),
    reset: () => set(() => ({ state: { isEditOpen: false, draft: {} } })),
  },
}));
