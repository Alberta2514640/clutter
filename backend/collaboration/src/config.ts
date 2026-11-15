// Settings for the service. They are read once when it starts.
// If a value is bad, the service refuses to start.

export interface Config {
  host: string;
  port: number;
  // Address of A's API. We call it to check access.
  apiUrl: string;
  // Addresses of the website. Sockets from any other page are refused.
  allowedOrigins: string[];
  // How long to wait for A's API, in ms.
  accessTimeoutMs: number;
  maxMessageBytes: number;
  maxClientsPerRoom: number;
  maxRooms: number;
  // How long to keep an empty room, in ms.
  roomIdleMs: number;
  // How often to ping each client, in ms. A client that does not answer is closed.
  pingMs: number;
}

type Env = Record<string, string | undefined>;

// Only this machine is allowed for now. There is no TLS yet.
const LOOPBACK_HOSTS = new Set(["127.0.0.1", "::1", "localhost"]);

// Read a whole number from the environment. It must be between min and max.
function int(env: Env, name: string, fallback: number, min: number, max: number): number {
  const raw = env[name];
  if (raw === undefined || raw === "") return fallback;
  const n = Number(raw);
  if (!Number.isInteger(n) || n < min || n > max) {
    throw new Error(`${name} must be a whole number from ${min} to ${max}`);
  }
  return n;
}

// Check that a value is an http or https address. Keep only its origin part.
function httpOrigin(name: string, raw: string): string {
  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    throw new Error(`${name} must be an http or https address`);
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    throw new Error(`${name} must be an http or https address`);
  }
  return url.origin;
}

// Build the settings from environment variables. Missing ones get a default.
export function loadConfig(env: Env = process.env): Config {
  const host = env.COLLAB_HOST || "127.0.0.1";
  if (!LOOPBACK_HOSTS.has(host)) {
    throw new Error("COLLAB_HOST must be 127.0.0.1, ::1 or localhost: the service only listens on this machine");
  }
  const allowedOrigins = (env.COLLAB_ALLOWED_ORIGINS || "http://localhost:3000")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean)
    .map((s) => httpOrigin("COLLAB_ALLOWED_ORIGINS", s));
  if (allowedOrigins.length === 0) throw new Error("COLLAB_ALLOWED_ORIGINS must name at least one address");

  return {
    host,
    port: int(env, "COLLAB_PORT", 8791, 1, 65535),
    apiUrl: httpOrigin("COLLAB_API_URL", env.COLLAB_API_URL || "http://localhost:8787"),
    allowedOrigins,
    accessTimeoutMs: int(env, "COLLAB_ACCESS_TIMEOUT_MS", 5000, 100, 60_000),
    maxMessageBytes: int(env, "COLLAB_MAX_MESSAGE_BYTES", 1024 * 1024, 1024, 64 * 1024 * 1024),
    maxClientsPerRoom: int(env, "COLLAB_MAX_CLIENTS_PER_ROOM", 20, 1, 1000),
    maxRooms: int(env, "COLLAB_MAX_ROOMS", 100, 1, 100_000),
    roomIdleMs: int(env, "COLLAB_ROOM_IDLE_MS", 30_000, 0, 3_600_000),
    pingMs: int(env, "COLLAB_PING_MS", 30_000, 1000, 600_000),
  };
}
