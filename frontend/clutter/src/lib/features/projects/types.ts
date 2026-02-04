export interface Project {
  projectId: string;
  organizationId: string;
  creatorId: string;
  name: string;
  description: string;
  createdAt: string;
  // memberCount?: number;
}

export type CreateProjectInput = {
  organizationId: string;
  name: string;
  description?: string;
};

export type UpdateProjectInput = {
  organizationId?: string;
  name?: string;
  description?: string;
};

export type ApiEnvelope<T> = {
  data: T;
  message?: string;
};
