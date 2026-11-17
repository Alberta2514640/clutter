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

// ─── Ansible job status ───────────────────────────────────────────────────────

// The statuses A writes for an Ansible job, owned by the backend's
// AllowedJobStatuses (backend/api/generic/job_statuses.go and job_status.go).
// A finished job is COMPLETED or FAILED.
export type AnsibleJobStatus = "QUEUED" | "STARTING" | "RUNNING" | "COMPLETED" | "FAILED" | string;

export const ANSIBLE_TERMINAL_STATUSES = new Set<string>(["COMPLETED", "FAILED"]);

export type AnsibleJob = {
  id: string;
  job_type: string;
  status: AnsibleJobStatus;
  playbook_s3_key: string;
  target_instance_ids: string[];
  extra_vars: Record<string, unknown>;
  task_arn: string | null;
  error_message: string | null;
  created_at: string;
  updated_at: string;
};

export type ApiEnvelope<T> = {
  data: T;
  message?: string;
};
