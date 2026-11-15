// Close codes we send when we shut a WebSocket.
// The browser cannot read the HTTP status of a failed WebSocket.
// So we open the socket first, then close it with one of these codes.
// The editor stops retrying on 4400, 4401, 4403 and 4404.
export const CLOSE = {
  badRequest: 4400,
  unauthorized: 4401,
  forbiddenOrigin: 4403,
  // Used for both "does not exist" and "no access", same as A.
  // This way nobody can tell if another org's diagram exists.
  notFound: 4404,
  full: 4429,
  upstream: 4500,
} as const;

export type CloseCode = (typeof CLOSE)[keyof typeof CLOSE];
