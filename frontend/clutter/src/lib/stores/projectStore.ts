// src/lib/stores/projectStore.ts
import { create } from "zustand";

// Project types (moved from components)
export interface Project {
  projectId: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
  memberCount?: number;
  tenantId?: string;
}

export interface ProjectDataState {
  projects: Project[];
  currentProject: Project | null;
  isLoading: boolean;
  isSaving: boolean;
  error: string | null;
}

export interface ProjectActions {
  loadProjects: () => Promise<void>;
  loadProject: (projectId: string) => Promise<void>;
  createProject: (data: { name: string; description: string }) => Promise<void>;
  updateProject: (
    projectId: string,
    data: Partial<{ name: string; description: string }>,
  ) => Promise<void>;
  deleteProject: (projectId: string) => Promise<void>;
  reset: () => void;
}

export interface ProjectStore {
  state: ProjectDataState;
  actions: ProjectActions;
}

// Mock data
const MOCK_PROJECTS: Project[] = [
  {
    projectId: "1",
    name: "Web Application",
    description:
      "Production web application with Lambda, API Gateway, and DynamoDB",
    createdAt: "2025-01-15T10:00:00Z",
    updatedAt: "2025-01-20T14:30:00Z",
    memberCount: 3,
  },
  {
    projectId: "2",
    name: "Data Pipeline",
    description: "ETL pipeline with S3, Lambda, and Glue",
    createdAt: "2025-01-10T08:00:00Z",
    updatedAt: "2025-01-18T16:45:00Z",
    memberCount: 2,
  },
  {
    projectId: "3",
    name: "Monitoring Stack",
    description: "CloudWatch dashboards and alarms",
    createdAt: "2025-01-05T12:00:00Z",
    updatedAt: "2025-01-12T09:15:00Z",
    memberCount: 1,
  },
];

export const useProjectStore = create<ProjectStore>((set, get) => ({
  state: {
    projects: [],
    currentProject: null,
    isLoading: false,
    isSaving: false,
    error: null,
  },

  actions: {
    loadProjects: async () => {
      set((s) => ({
        state: { ...s.state, isLoading: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 300));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/projects');
        // const data = await response.json();

        set((s) => ({
          state: {
            ...s.state,
            projects: MOCK_PROJECTS,
            isLoading: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isLoading: false,
            error: e instanceof Error ? e.message : "Failed to load projects",
          },
        }));
      }
    },

    loadProject: async (projectId: string) => {
      set((s) => ({
        state: { ...s.state, isLoading: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 300));

        // TODO: Replace with actual API call
        // const response = await fetch(`/api/projects/${projectId}`);
        // const project = await response.json();

        const project =
          MOCK_PROJECTS.find((p) => p.projectId === projectId) || null;

        set((s) => ({
          state: {
            ...s.state,
            currentProject: project,
            isLoading: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isLoading: false,
            error: e instanceof Error ? e.message : "Failed to load project",
          },
        }));
      }
    },

    createProject: async (data: { name: string; description: string }) => {
      set((s) => ({
        state: { ...s.state, isSaving: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 500));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/projects', {
        //   method: 'POST',
        //   body: JSON.stringify(data)
        // });
        // const newProject = await response.json();

        const newProject: Project = {
          projectId: `p_${Date.now()}`,
          ...data,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          memberCount: 1,
        };

        set((s) => ({
          state: {
            ...s.state,
            projects: [...s.state.projects, newProject],
            isSaving: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isSaving: false,
            error: e instanceof Error ? e.message : "Failed to create project",
          },
        }));
      }
    },

    updateProject: async (
      projectId: string,
      data: Partial<{ name: string; description: string }>,
    ) => {
      set((s) => ({
        state: { ...s.state, isSaving: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 500));

        // TODO: Replace with actual API call
        // const response = await fetch(`/api/projects/${projectId}`, {
        //   method: 'PUT',
        //   body: JSON.stringify(data)
        // });

        set((s) => ({
          state: {
            ...s.state,
            projects: s.state.projects.map((p) =>
              p.projectId === projectId
                ? { ...p, ...data, updatedAt: new Date().toISOString() }
                : p,
            ),
            currentProject:
              s.state.currentProject?.projectId === projectId
                ? {
                    ...s.state.currentProject,
                    ...data,
                    updatedAt: new Date().toISOString(),
                  }
                : s.state.currentProject,
            isSaving: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isSaving: false,
            error: e instanceof Error ? e.message : "Failed to update project",
          },
        }));
      }
    },

    deleteProject: async (projectId: string) => {
      set((s) => ({
        state: { ...s.state, isSaving: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 500));

        // TODO: Replace with actual API call
        // await fetch(`/api/projects/${projectId}`, { method: 'DELETE' });

        set((s) => ({
          state: {
            ...s.state,
            projects: s.state.projects.filter((p) => p.projectId !== projectId),
            currentProject:
              s.state.currentProject?.projectId === projectId
                ? null
                : s.state.currentProject,
            isSaving: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isSaving: false,
            error: e instanceof Error ? e.message : "Failed to delete project",
          },
        }));
      }
    },

    reset: () =>
      set(() => ({
        state: {
          projects: [],
          currentProject: null,
          isLoading: false,
          isSaving: false,
          error: null,
        },
      })),
  },
}));

export const useProjectState = () => useProjectStore((s) => s.state);
export const useProjectActions = () => useProjectStore((s) => s.actions);
