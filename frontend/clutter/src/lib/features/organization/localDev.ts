// Whether the website talks to the local dev gateway (A2), whose own routes,
// such as POST /dev/link-fake-account, exist only on a developer's machine.
// True only for a development build whose API is plain http on a loopback
// address; a production build is never local, whatever its API address.
export function localDevApi(nodeEnv: string | undefined, apiBase: string | undefined): boolean {
  return nodeEnv === "development" && /^http:\/\/(localhost|127\.0\.0\.1)(:\d+)?\/?$/.test(apiBase ?? "");
}
