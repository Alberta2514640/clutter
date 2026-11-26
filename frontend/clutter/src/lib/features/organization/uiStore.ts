// lib/features/organization/uiStore.ts
import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

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
