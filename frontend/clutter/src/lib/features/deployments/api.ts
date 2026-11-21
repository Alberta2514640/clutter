import { adapterFetch, adapterFetchText, seg } from "@/lib/adapterApi";
import type {
  Compatibility,
  CreateDeploymentInput,
  Deployment,
  DeploymentListItem,
  PreviewFiles,
  Run,
  RunAction,
  UpdateDeploymentInput,
} from "./types";

const org = (organizationId: string) => `/v1/organizations/${seg(organizationId)}`;

export const deploymentsApi = {
  listForDiagram: async (token: string, organizationId: string, diagramId: string): Promise<DeploymentListItem[]> => {
    const json = await adapterFetch<{ items: DeploymentListItem[] | null }>(
      `${org(organizationId)}/diagrams/${seg(diagramId)}/deployments`,
      token,
      { method: "GET" },
    );
    return json.items ?? [];
  },

  // The diagram goes in the path; the body holds the rest.
  create: (token: string, organizationId: string, input: CreateDeploymentInput): Promise<Deployment> => {
    const { diagram_id: diagramId, ...body } = input;
    return adapterFetch<Deployment>(`${org(organizationId)}/diagrams/${seg(diagramId)}/deployments`, token, {
      method: "POST",
      body: JSON.stringify(body),
    });
  },

  update: (token: string, organizationId: string, deploymentId: string, input: UpdateDeploymentInput): Promise<Deployment> =>
    adapterFetch<Deployment>(`${org(organizationId)}/deployments/${seg(deploymentId)}`, token, {
      method: "PATCH",
      body: JSON.stringify(input),
    }),

  // Answered with 204. B refuses (409) a deployment that may hold resources
  // or has a run queued or running.
  remove: (token: string, organizationId: string, deploymentId: string): Promise<void> =>
    adapterFetch<void>(`${org(organizationId)}/deployments/${seg(deploymentId)}`, token, { method: "DELETE" }),

  compatibility: (token: string, organizationId: string, deploymentId: string): Promise<Compatibility> =>
    adapterFetch<Compatibility>(`${org(organizationId)}/deployments/${seg(deploymentId)}/compatibility`, token, { method: "GET" }),

  // 501 (code no_preview) for a cloud without a preview.
  preview: async (token: string, organizationId: string, deploymentId: string): Promise<Record<string, string>> => {
    const json = await adapterFetch<PreviewFiles>(`${org(organizationId)}/deployments/${seg(deploymentId)}/preview`, token, {
      method: "GET",
    });
    return json.files ?? {};
  },

  // Answered with 202 and the queued run.
  startRun: (token: string, organizationId: string, deploymentId: string, action: RunAction): Promise<Run> =>
    adapterFetch<Run>(`${org(organizationId)}/deployments/${seg(deploymentId)}/runs`, token, {
      method: "POST",
      body: JSON.stringify({ action }),
    }),

  runLogs: (token: string, organizationId: string, runId: string): Promise<string> =>
    adapterFetchText(`${org(organizationId)}/runs/${seg(runId)}/logs`, token),
};
