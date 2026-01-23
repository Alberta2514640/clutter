// src/lib/stores/runStore.ts
import { create } from "zustand";

// Run types (moved from components)
export interface Run {
  runId: string;
  projectId: string;
  projectName: string;
  workspaceId: string;
  action: "plan" | "apply";
  status: "QUEUED" | "RUNNING" | "SUCCESS" | "FAILED";
  startedAt: string;
  endedAt?: string;
}

export interface RunDataState {
  runs: Run[];
  currentRun: Run | null;
  isLoading: boolean;
  error: string | null;
}

export interface RunActions {
  loadRecentRuns: () => Promise<void>;
  loadRun: (projectId: string, runId: string) => Promise<void>;
  reset: () => void;
}

export interface RunStore {
  state: RunDataState;
  actions: RunActions;
}

// Mock data
const MOCK_RUNS: Run[] = [
  {
    runId: "run_001",
    projectId: "p_web_app",
    projectName: "Web Application",
    workspaceId: "ws_prod",
    action: "apply",
    status: "SUCCESS",
    startedAt: "2025-01-20T14:00:00Z",
    endedAt: "2025-01-20T14:05:30Z",
  },
  {
    runId: "run_002",
    projectId: "p_web_app",
    projectName: "Web Application",
    workspaceId: "ws_staging",
    action: "plan",
    status: "SUCCESS",
    startedAt: "2025-01-19T10:30:00Z",
    endedAt: "2025-01-19T10:32:15Z",
  },
  {
    runId: "run_003",
    projectId: "p_data_pipeline",
    projectName: "Data Pipeline",
    workspaceId: "ws_prod",
    action: "apply",
    status: "RUNNING",
    startedAt: "2025-01-21T08:15:00Z",
  },
  {
    runId: "run_004",
    projectId: "p_monitoring",
    projectName: "Monitoring Stack",
    workspaceId: "ws_prod",
    action: "apply",
    status: "FAILED",
    startedAt: "2025-01-18T16:00:00Z",
    endedAt: "2025-01-18T16:03:45Z",
  },
];

export const useRunStore = create<RunStore>((set, get) => ({
  state: {
    runs: [],
    currentRun: null,
    isLoading: false,
    error: null,
  },

  actions: {
    loadRecentRuns: async () => {
      set((s) => ({
        state: { ...s.state, isLoading: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 300));

        // TODO: Replace with actual API call
        // const response = await fetch('/api/runs/recent');
        // const data = await response.json();

        set((s) => ({
          state: {
            ...s.state,
            runs: MOCK_RUNS,
            isLoading: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isLoading: false,
            error: e instanceof Error ? e.message : "Failed to load runs",
          },
        }));
      }
    },

    loadRun: async (projectId: string, runId: string) => {
      set((s) => ({
        state: { ...s.state, isLoading: true, error: null },
      }));

      try {
        // Simulate API call
        await new Promise((r) => setTimeout(r, 300));

        // TODO: Replace with actual API call
        // const response = await fetch(`/api/projects/${projectId}/runs/${runId}`);
        // const run = await response.json();

        const run =
          MOCK_RUNS.find(
            (r) => r.runId === runId && r.projectId === projectId,
          ) || null;

        set((s) => ({
          state: {
            ...s.state,
            currentRun: run,
            isLoading: false,
          },
        }));
      } catch (e) {
        set((s) => ({
          state: {
            ...s.state,
            isLoading: false,
            error: e instanceof Error ? e.message : "Failed to load run",
          },
        }));
      }
    },

    reset: () =>
      set(() => ({
        state: {
          runs: [],
          currentRun: null,
          isLoading: false,
          error: null,
        },
      })),
  },
}));

export const useRunState = () => useRunStore((s) => s.state);
export const useRunActions = () => useRunStore((s) => s.actions);
