// lib/features/diagram/api.ts

import type { ApiEnvelope, CreateDiagramInput, Diagram, DiagramApiItem, UpdateDiagramInput } from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

async function apiFetch<T>(path: string, token: string, init: RequestInit = {}): Promise<T> {
  if (!API_BASE) throw new Error("API_ENDPOINT is not set");

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `${token}`,
      ...(init.headers ?? {}),
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `Request failed: ${res.status}`);
  }

  return (await res.json()) as T;
}

/**
 * Backend reality (from your examples):
 *  - GET    /diagram?projectId=...                (list)
 *  - GET    /diagram?projectId=...&diagramId=...  (detail)  -> returns { data: [ { id, name, data: {nodes,edges}, ... } ] }
 *  - POST   /diagram                              (create)
 *  - PUT    /diagram                              (update) -> body expects { projectId, diagramId, name, uiLayout:{nodes,edges} }
 *  - DELETE /diagram?projectId=...&diagramId=...  (delete)
 *
 * Frontend rule:
 *  - We normalize backend "data" -> frontend "uiLayout" so ReactFlow always uses:
 *      diagram.uiLayout.nodes / diagram.uiLayout.edges
 */

function normalizeDiagram(item: DiagramApiItem): Diagram {
  return {
    id: item.id,
    name: item.name,
    projectId: item.projectId,
    uiLayout: {
      nodes: item.data?.nodes ?? [],
      edges: item.data?.edges ?? [],
    },
    createdBy: item.createdBy,
    createdAt: item.createdAt,
    latestUpdateBy: item.latestUpdateBy,
    latestUpdateAt: item.latestUpdateAt,
  };
}

function normalizeMany(payload: DiagramApiItem | DiagramApiItem[]): Diagram[] {
  const arr = Array.isArray(payload) ? payload : [payload];
  return arr.map(normalizeDiagram);
}

export const diagramApi = {
  // GET /diagram?projectId=...
  // Backend may return data: DiagramApiItem[] or a single item; normalize to Diagram[]
  list: async (token: string, projectId: string): Promise<Diagram[]> => {
    const qs = new URLSearchParams({ projectId });

    const json = await apiFetch<ApiEnvelope<DiagramApiItem[] | DiagramApiItem>>(`/diagram?${qs.toString()}`, token, {
      method: "GET",
    });

    return normalizeMany(json.data);
  },

  // POST /diagram
  // If backend returns a DiagramApiItem or [DiagramApiItem], normalize to Diagram (first item)
  create: async (token: string, input: CreateDiagramInput): Promise<Diagram> => {
    const json = await apiFetch<ApiEnvelope<DiagramApiItem[] | DiagramApiItem>>("/diagram", token, {
      method: "POST",
      body: JSON.stringify(input),
    });

    const first = Array.isArray(json.data) ? json.data[0] : json.data;
    if (!first) throw new Error("Create returned no diagram");
    return normalizeDiagram(first);
  },

  // GET /diagram?projectId=...&diagramId=...
  // Backend returns data: [DiagramApiItem]
  get: async (token: string, projectId: string, diagramId: string): Promise<Diagram> => {
    const qs = new URLSearchParams({ projectId, diagramId });

    const json = await apiFetch<ApiEnvelope<DiagramApiItem[] | DiagramApiItem>>(`/diagram?${qs.toString()}`, token, {
      method: "GET",
    });

    const first = Array.isArray(json.data) ? json.data[0] : json.data;
    if (!first) throw new Error("Diagram not found");
    return normalizeDiagram(first);
  },

  // PUT /diagram
  update: async (token: string, input: UpdateDiagramInput): Promise<Diagram> => {
    const json = await apiFetch<ApiEnvelope<DiagramApiItem[] | DiagramApiItem | null | undefined>>(`/diagram`, token, {
      method: "PUT",
      body: JSON.stringify(input),
    });

    // 2) if backend returned the updated diagram, normalize it
    const returned = json?.data;
    const item = Array.isArray(returned) ? returned[0] : returned;

    if (item && typeof item === "object" && "id" in item) {
      return normalizeDiagram(item as DiagramApiItem);
    }

    return await diagramApi.get(token, input.projectId, input.diagramId);
  },

  // DELETE /diagram?projectId=...&diagramId=...
  delete: async (token: string, projectId: string, diagramId: string): Promise<void> => {
    const qs = new URLSearchParams({ projectId, diagramId });

    await apiFetch<ApiEnvelope<unknown>>(`/diagram?${qs.toString()}`, token, {
      method: "DELETE",
    });
  },
};
