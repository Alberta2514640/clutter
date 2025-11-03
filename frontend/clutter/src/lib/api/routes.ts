export const routes = {
  canvasBundle: (pid: string, cid: string) => `/api/projects/${pid}/canvases/${cid}/bundle`,
  node: (pid: string, nid: string) => `/api/projects/${pid}/nodes/${nid}`,
  edge: (pid: string, eid: string) => `/api/projects/${pid}/edges/${eid}`,
  runs: (pid: string) => `/api/projects/${pid}/runs`,
  run: (pid: string, rid: string) => `/api/projects/${pid}/runs/${rid}`,
  exportZip: (pid: string) => `/api/projects/${pid}/export`,
  lockAcquire: `/api/locks`,
  lockRelease: (lid: string) => `/api/locks/${lid}`,
};
