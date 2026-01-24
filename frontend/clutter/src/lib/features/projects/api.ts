// src/lib/features/projects/api.ts
import type { Project } from "./types";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;

const MOCK_PROJECTS: Project[] = [
  {
    projectId: "1",
    name: "Web Application",
    description: "Production web application with Lambda, API Gateway, and DynamoDB",
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

export const projectsApi = {
  // keep tenantId parameter for future backend integration
  listByTenant: async (_tenantId: string): Promise<Project[]> => {
    await sleep(300);
    return clone(MOCK_PROJECTS);
  },
};
