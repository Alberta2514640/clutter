import { adapterFetch, seg } from "@/lib/adapterApi";
import type { Connection, ConnectionOptions, CreateConnectionInput } from "./types";

const base = (organizationId: string) => `/v1/organizations/${seg(organizationId)}/connections`;

export const connectionsApi = {
  list: async (token: string, organizationId: string): Promise<Connection[]> => {
    const json = await adapterFetch<{ items: Connection[] | null }>(base(organizationId), token, { method: "GET" });
    return json.items ?? [];
  },

  // B verifies the credentials before it saves anything.
  create: (token: string, organizationId: string, input: CreateConnectionInput): Promise<Connection> =>
    adapterFetch<Connection>(base(organizationId), token, { method: "POST", body: JSON.stringify(input) }),

  verify: (token: string, organizationId: string, connectionId: string): Promise<Connection> =>
    adapterFetch<Connection>(`${base(organizationId)}/${seg(connectionId)}/verify`, token, { method: "POST" }),

  // The values the connection's cloud offers for its deployment settings.
  // Refused with 501 (code no_options) for a cloud that lists none, and 502
  // (code options_failed) when the cloud could not be asked.
  options: (token: string, organizationId: string, connectionId: string): Promise<ConnectionOptions> =>
    adapterFetch<ConnectionOptions>(`${base(organizationId)}/${seg(connectionId)}/options`, token, { method: "GET" }),

  // Refused with 409 (code live_deployments) while a deployment is live.
  delete: (token: string, organizationId: string, connectionId: string): Promise<void> =>
    adapterFetch<void>(`${base(organizationId)}/${seg(connectionId)}`, token, { method: "DELETE" }),
};
