// lib/features/organization/uiStore.ts
import { create } from "zustand";
import type { Organization } from "./types";

type OrgUiState = {
  isEditOpen: boolean;
  draft: Partial<Organization>;
};

type OrgUiActions = {
  open: (org?: Organization | null) => void;
  close: () => void;
  patch: (p: Partial<Organization>) => void;
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
          draft: org ? { name: org.name, slug: org.slug, timeZone: org.timeZone } : {},
        },
      })),
    close: () => set((s) => ({ state: { ...s.state, isEditOpen: false, draft: {} } })),
    patch: (p) => set((s) => ({ state: { ...s.state, draft: { ...s.state.draft, ...p } } })),
    reset: () => set(() => ({ state: { isEditOpen: false, draft: {} } })),
  },
}));
