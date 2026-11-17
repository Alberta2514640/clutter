import { adapterFetch, seg } from "@/lib/adapterApi";
import type { CatalogEntry, CatalogResponse, LinkedAccount } from "./types";

export const catalogApi = {
  // GET /v1/catalog
  list: async (token: string): Promise<CatalogEntry[]> => {
    const json = await adapterFetch<CatalogResponse>("/v1/catalog", token, { method: "GET" });
    return json.items ?? [];
  },

  // GET /v1/organizations/{org}/connectors/{connector}/linked-accounts
  linked: async (token: string, organizationId: string, connector: string): Promise<LinkedAccount[]> => {
    const json = await adapterFetch<{ items: LinkedAccount[] | null }>(
      `/v1/organizations/${seg(organizationId)}/connectors/${seg(connector)}/linked-accounts`,
      token,
      { method: "GET" },
    );
    return json.items ?? [];
  },
};
