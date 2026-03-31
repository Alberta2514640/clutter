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

export type CreatePlaybookUploadUrlInput = {
  file_name: string;
  project_id: string;
  diagram_id: string;
};

export type CreatePlaybookUploadUrlResponse = {
  upload_url?: string;
  url?: string;
  method?: string;
  key?: string;
  playbook_s3_key?: string;
  playbook_id?: string;
  fields?: Record<string, string>;
  [key: string]: unknown;
};

export type UploadPlaybookFileToS3Input = {
  upload_url: string;
  file: File;
};

export type SubmitAnsibleJobInput = {
  account_access_role_id: string;
  playbook_id: string;
  target_instance_ids: string[];
};

export type SubmitAnsibleJobResponse = {
  job_id: string;
  status: "QUEUED" | string;
  [key: string]: unknown;
};

export type ApiEnvelope<T> = {
  data: T;
  message?: string;
};
