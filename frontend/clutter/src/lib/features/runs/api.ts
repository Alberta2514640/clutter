// src/lib/features/runs/api.ts
import type { Run } from "./types";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;

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

export const runsApi = {
  // keep tenantId param for future backend wiring
  listRecentByTenant: async (_tenantId: string): Promise<Run[]> => {
    await sleep(300);
    return clone(MOCK_RUNS);
  },
};
