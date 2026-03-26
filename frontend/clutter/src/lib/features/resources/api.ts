import type { ApiEnvelope, SupportedResource } from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

export const resourcesApi = {
  list: async (): Promise<SupportedResource[]> => {
    if (!API_BASE) throw new Error("API_ENDPOINT is not set");

    const res = await fetch(`${API_BASE}/resources`, { method: "GET" });

    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(text || `Request failed: ${res.status}`);
    }

    const json = (await res.json()) as ApiEnvelope<SupportedResource[]>;
    return json.data ?? [];
  },
};
