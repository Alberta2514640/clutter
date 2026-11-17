import { ADAPTER_API_BASE, AdapterApiError, adapterFetch, errorFromBody, NOT_CONFIGURED_MESSAGE, seg, UNREACHABLE_MESSAGE } from "@/lib/adapterApi";
import type { FunctionCode, FunctionCodeList } from "./types";

const codePath = (organizationId: string, diagramId: string) =>
  `/v1/organizations/${seg(organizationId)}/diagrams/${seg(diagramId)}/code`;

// The upload goes through XMLHttpRequest, not adapterFetch, because fetch reports
// no upload progress. It uses adapterApi's base address, messages and error envelope.

export const functionCodeApi = {
  list: async (token: string, organizationId: string, diagramId: string): Promise<FunctionCode[]> => {
    const json = await adapterFetch<FunctionCodeList>(codePath(organizationId, diagramId), token, { method: "GET" });
    return json.items ?? [];
  },

  // Answered with 204, or 404 when the block has no code.
  remove: (token: string, organizationId: string, diagramId: string, nodeId: string): Promise<void> =>
    adapterFetch<void>(`${codePath(organizationId, diagramId)}/${seg(nodeId)}`, token, { method: "DELETE" }),

  // PUT the zip as the request body. B refuses with 413 over 50 MB, 415 for
  // a body that is not sent as a zip, and 422 for a file that is not a zip,
  // holds an unsafe path, or belongs to a block that is not a saved function.
  upload: (
    token: string,
    organizationId: string,
    diagramId: string,
    nodeId: string,
    file: File,
    onProgress: (loaded: number, total: number) => void,
  ): Promise<FunctionCode> =>
    new Promise((resolve, reject) => {
      if (!ADAPTER_API_BASE) {
        reject(new AdapterApiError(0, "not_configured", NOT_CONFIGURED_MESSAGE));
        return;
      }
      const xhr = new XMLHttpRequest();
      xhr.open("PUT", `${ADAPTER_API_BASE}${codePath(organizationId, diagramId)}/${seg(nodeId)}?file_name=${encodeURIComponent(file.name)}`);
      xhr.setRequestHeader("Authorization", token);
      xhr.setRequestHeader("Content-Type", "application/zip");
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) onProgress(e.loaded, e.total);
      };
      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            resolve(JSON.parse(xhr.responseText) as FunctionCode);
          } catch {
            reject(new AdapterApiError(xhr.status, "error", "Clutter stored the code but its answer could not be read. Reload to see it."));
          }
          return;
        }
        reject(errorFromBody(xhr.status, xhr.responseText, `The upload failed (${xhr.status}).`));
      };
      xhr.onerror = () => reject(new AdapterApiError(0, "unreachable", UNREACHABLE_MESSAGE));
      xhr.onabort = () => reject(new AdapterApiError(0, "aborted", "The upload was stopped."));
      xhr.send(file);
    }),
};
