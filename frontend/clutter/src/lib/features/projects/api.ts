import type { Project } from "./types";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const clone = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;

let MOCK_PROJECTS: Project[] = [
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
  // keep signature same as your current hook
  listByTenant: async (_tenantId: string): Promise<Project[]> => {
    await sleep(250);
    return clone(MOCK_PROJECTS);
  },

  getById: async (projectId: string): Promise<Project> => {
    await sleep(200);
    const p = MOCK_PROJECTS.find((x) => x.projectId === projectId);
    if (!p) throw new Error("Project not found");
    return clone(p);
  },

  update: async (projectId: string, data: Partial<Project>): Promise<Project> => {
    await sleep(250);
    const idx = MOCK_PROJECTS.findIndex((x) => x.projectId === projectId);
    if (idx === -1) throw new Error("Project not found");

    const updated: Project = {
      ...MOCK_PROJECTS[idx],
      ...data,
      updatedAt: new Date().toISOString(),
    };

    MOCK_PROJECTS[idx] = updated;
    return clone(updated);
  },

  delete: async (projectId: string): Promise<void> => {
    await sleep(250);
    MOCK_PROJECTS = MOCK_PROJECTS.filter((x) => x.projectId !== projectId);
  },
};
