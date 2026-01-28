// lib/features/organization/uiStore.ts
import { create } from "zustand";
import type { Organization } from "./types";

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
