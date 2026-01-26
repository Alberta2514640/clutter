export type RunAction = "plan" | "apply";
export type RunStatus = "QUEUED" | "RUNNING" | "SUCCESS" | "FAILED";

export interface Run {
  runId: string;
  projectId: string;
  projectName: string;
  workspaceId: string;
  action: RunAction;
  status: RunStatus;
  startedAt: string;
  endedAt?: string;
}
