import { adapterFetch } from "@/lib/adapterApi";
import type { Example, ExamplesResponse } from "./types";

export const examplesApi = {
  // GET /v1/examples
  list: async (token: string): Promise<Example[]> => {
    const json = await adapterFetch<ExamplesResponse>("/v1/examples", token, { method: "GET" });
    return json.items ?? [];
  },
};
