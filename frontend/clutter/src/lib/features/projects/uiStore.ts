import { create } from "zustand";

//this store is needed for the form states in the settings tab in the projects page

type Draft = {
  projectId: string | null;
  name: string;
  description: string;
};

type DraftStore = {
  draft: Draft;
  startDraft: (projectId: string, name: string, description: string) => void;
  setName: (name: string) => void;
  setDescription: (description: string) => void;
  reset: () => void;
};

export const useProjectSettingsDraft = create<DraftStore>((set) => ({
  draft: { projectId: null, name: "", description: "" },

  startDraft: (projectId, name, description) => set({ draft: { projectId, name, description } }),

  setName: (name) => set((s) => ({ draft: { ...s.draft, name } })),
  setDescription: (description) => set((s) => ({ draft: { ...s.draft, description } })),

  reset: () => set({ draft: { projectId: null, name: "", description: "" } }),
}));
