// The live sync service. It listens for WebSockets, one room per diagram.
// URL: ws://<host>:<port>/<diagramId>?projectId=<projectId>
// Start it with: npm start

import { createServer, type IncomingMessage } from "node:http";
import type { Duplex } from "node:stream";
import { WebSocketServer, type WebSocket } from "ws";
import { checkAccess, type Access } from "./access.ts";
import { CLOSE } from "./codes.ts";
import { loadConfig, type Config } from "./config.ts";
import { handleConnection } from "./connection.ts";
import { Rooms } from "./rooms.ts";

export interface CollabServer {
  // The port the service listens on.
  port: number;
  rooms: Rooms;
  close(): Promise<void>;
}

// Start the service. "fetchFn" is how we call A's API. Tests pass a fake one.
export async function startServer(cfg: Config, fetchFn: typeof fetch = fetch): Promise<CollabServer> {
  const rooms = new Rooms(cfg);

  // Plain HTTP is only used for the health check.
  const http = createServer((req, res) => {
    if (req.method === "GET" && req.url === "/healthz") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end('{"status":"ok"}');
      return;
    }
    res.writeHead(404, { "Content-Type": "application/json" });
    res.end('{"message":"not found"}');
  });

  // "noServer" lets us run our checks before the WebSocket opens.
  // A message bigger than maxPayload closes the socket with code 1009.
  const wss = new WebSocketServer({ noServer: true, maxPayload: cfg.maxMessageBytes });

  // Decide if a new connection is allowed.
  const check = async (req: IncomingMessage, diagramId: string, projectId: string | null): Promise<Access> => {
    // The browser sends the login cookie from any website, so we check which
    // website opened the socket. Only our own website is allowed.
    const origin = req.headers.origin;
    if (!origin || !cfg.allowedOrigins.includes(origin)) {
      return { ok: false, code: CLOSE.forbiddenOrigin, reason: "origin not allowed" };
    }
    return checkAccess({ cookieHeader: req.headers.cookie, diagramId, projectId }, cfg, fetchFn);
  };

  // Close a socket we do not allow, after a short wait.
  // The y-websocket client reconnects 0.1 seconds after a close. Without the
  // wait, one refused browser tab would call A's API about 10 times a second.
  const refuse = (client: WebSocket, code: number, reason: string): void => {
    client.on("error", () => {});
    const timer = setTimeout(() => client.close(code, reason), cfg.refuseDelayMs);
    // If the client leaves first, there is nothing left to close.
    client.on("close", () => clearTimeout(timer));
  };

  const onUpgrade = async (req: IncomingMessage, socket: Duplex, head: Buffer): Promise<void> => {
    // If the client goes away while we wait for A's API, do not crash.
    socket.on("error", () => {});

    const url = new URL(req.url ?? "/", "http://localhost");
    const diagramId = url.pathname.slice(1);
    const access = await check(req, diagramId, url.searchParams.get("projectId"));
    if (socket.destroyed) return;

    wss.handleUpgrade(req, socket, head, (client) => {
      // The browser cannot read an HTTP error on a WebSocket.
      // So we always open the socket, and close it with a code if the answer is no.
      if (!access.ok) {
        refuse(client, access.code, access.reason);
        return;
      }
      const room = rooms.join(diagramId, client, { document: access.document, name: access.diagramName });
      if (!room) {
        refuse(client, CLOSE.full, "full");
        return;
      }
      handleConnection(client, room, access, { pingMs: cfg.pingMs, onClose: () => rooms.leave(room, client) });
    });
  };

  http.on("upgrade", (req, socket, head) => {
    onUpgrade(req, socket, head).catch((err: unknown) => {
      // Log only the error text. Never log the request, it holds the login cookie.
      console.error("collaboration: could not open a connection:", err instanceof Error ? err.message : "unknown error");
      socket.destroy();
    });
  });

  await new Promise<void>((resolve, reject) => {
    http.once("error", reject);
    http.listen(cfg.port, cfg.host, resolve);
  });
  const address = http.address();
  const port = typeof address === "object" && address ? address.port : cfg.port;

  const close = async (): Promise<void> => {
    // 1001 means "the server is going away". Clients reconnect on it.
    for (const client of wss.clients) client.close(1001, "server stopping");
    rooms.closeAll();
    wss.close();
    http.closeAllConnections();
    await new Promise<void>((resolve) => http.close(() => resolve()));
  };

  return { port, rooms, close };
}

// This part runs only when the file is started directly, not when a test imports it.
if (import.meta.main) {
  const cfg = loadConfig();
  const server = await startServer(cfg);
  console.log(`collaboration: listening on ws://${cfg.host}:${server.port}`);
  const stop = (): void => {
    server.close().then(
      () => process.exit(0),
      () => process.exit(1),
    );
  };
  process.on("SIGINT", stop);
  process.on("SIGTERM", stop);
}
