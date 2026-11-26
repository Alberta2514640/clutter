// Helpers for the server tests: a small Yjs client and a fake of A's API.

import * as decoding from "lib0/decoding";
import * as encoding from "lib0/encoding";
import { WebSocket } from "ws";
import { Awareness, applyAwarenessUpdate, encodeAwarenessUpdate } from "y-protocols/awareness";
import { messageYjsSyncStep2, readSyncMessage, writeSyncStep1, writeUpdate } from "y-protocols/sync";
import * as Y from "yjs";

const MESSAGE_SYNC = 0;
const MESSAGE_AWARENESS = 1;

export const ORIGIN = "http://localhost:3000";

// Make a fake login token. The service does not check the signature.
export function tokenFor(claims: Record<string, unknown>): string {
  return `e30.${Buffer.from(JSON.stringify(claims)).toString("base64url")}.sig`;
}

// Wait until a check passes. Fails the test if it takes too long.
export async function until(check: () => boolean, what: string, timeoutMs = 2000): Promise<void> {
  const start = Date.now();
  while (!check()) {
    if (Date.now() - start > timeoutMs) throw new Error(`timed out waiting for: ${what}`);
    await new Promise((resolve) => setTimeout(resolve, 5));
  }
}

export interface ClientOptions {
  token?: string | null;
  origin?: string | null;
  // Use an existing document. This is how a test reconnects.
  doc?: Y.Doc;
  // Set to false to stop answering pings, like a dead connection.
  autoPong?: boolean;
}

// A small client that speaks the same messages as the y-websocket client.
export class TestClient {
  doc: Y.Doc;
  awareness: Awareness;
  ws: WebSocket;
  // True after the server sent us the full document.
  synced = false;
  // Set when the socket closes.
  closed: { code: number; reason: string } | null = null;

  constructor(url: string, options: ClientOptions = {}) {
    this.doc = options.doc ?? new Y.Doc();
    this.awareness = new Awareness(this.doc);
    this.awareness.setLocalState(null);

    const headers: Record<string, string> = {};
    const origin = options.origin === undefined ? ORIGIN : options.origin;
    if (origin !== null) headers.Origin = origin;
    if (options.token) headers.Cookie = `clutter_token=${encodeURIComponent(options.token)}`;
    this.ws = new WebSocket(url, { headers, autoPong: options.autoPong ?? true });

    this.ws.on("open", () => {
      const encoder = encoding.createEncoder();
      encoding.writeVarUint(encoder, MESSAGE_SYNC);
      writeSyncStep1(encoder, this.doc);
      this.ws.send(encoding.toUint8Array(encoder));
    });
    this.ws.on("message", (data: Buffer) => this.onMessage(new Uint8Array(data.buffer, data.byteOffset, data.byteLength)));
    this.ws.on("close", (code, reason) => {
      this.closed = { code, reason: reason.toString() };
    });
    this.ws.on("error", () => {});

    // Send our own document changes to the server.
    this.doc.on("update", (update: Uint8Array, origin: unknown) => {
      if (origin === this || this.ws.readyState !== WebSocket.OPEN) return;
      const encoder = encoding.createEncoder();
      encoding.writeVarUint(encoder, MESSAGE_SYNC);
      writeUpdate(encoder, update);
      this.ws.send(encoding.toUint8Array(encoder));
    });
  }

  private onMessage(data: Uint8Array): void {
    const decoder = decoding.createDecoder(data);
    const kind = decoding.readVarUint(decoder);
    if (kind === MESSAGE_SYNC) {
      const encoder = encoding.createEncoder();
      encoding.writeVarUint(encoder, MESSAGE_SYNC);
      const step = readSyncMessage(decoder, encoder, this.doc, this);
      if (encoding.length(encoder) > 1) this.ws.send(encoding.toUint8Array(encoder));
      if (step === messageYjsSyncStep2) this.synced = true;
    } else if (kind === MESSAGE_AWARENESS) {
      applyAwarenessUpdate(this.awareness, decoding.readVarUint8Array(decoder), this);
    }
  }

  // Set our presence and send it to the server.
  setPresence(presence: Record<string, unknown> | null): void {
    this.awareness.setLocalState(presence);
    this.sendPresence(encodeAwarenessUpdate(this.awareness, [this.doc.clientID]));
  }

  // Send a raw presence update. Tests use this to try to cheat.
  sendPresence(update: Uint8Array): void {
    const encoder = encoding.createEncoder();
    encoding.writeVarUint(encoder, MESSAGE_AWARENESS);
    encoding.writeVarUint8Array(encoder, update);
    this.ws.send(encoding.toUint8Array(encoder));
  }

  // The presence of everyone else, as a list.
  others(): unknown[] {
    return [...this.awareness.getStates()].filter(([id]) => id !== this.doc.clientID).map(([, presence]) => presence);
  }

  nodeIds(): string[] {
    return [...this.doc.getMap("nodes").keys()].sort();
  }

  // Add a simple VM node to the shared document.
  addNode(id: string): void {
    this.doc.transact(() => {
      const node = new Y.Map<unknown>();
      node.set("kind", "vm");
      node.set("name", id);
      node.set("position", { x: 0, y: 0 });
      node.set("fields", new Y.Map());
      this.doc.getMap("nodes").set(id, node);
    });
  }

  async waitSynced(): Promise<void> {
    await until(() => this.synced, "the first sync");
  }

  async waitClosed(): Promise<{ code: number; reason: string }> {
    await until(() => this.closed !== null, "the socket to close");
    return this.closed!;
  }

  close(): void {
    this.awareness.destroy();
    this.ws.close();
  }
}

// Build a presence update by hand, for any client id.
export function rawPresence(clientId: number, clock: number, presence: unknown): Uint8Array {
  const encoder = encoding.createEncoder();
  encoding.writeVarUint(encoder, 1);
  encoding.writeVarUint(encoder, clientId);
  encoding.writeVarUint(encoder, clock);
  encoding.writeVarString(encoder, JSON.stringify(presence));
  return encoding.toUint8Array(encoder);
}

export interface FakeDiagram {
  projectId: string;
  name: string;
  data: unknown;
  // The tokens that may open this diagram.
  members: string[];
}

// A fake of A's GET /diagram. It answers like A does:
// 403 for a token it does not know, 404 for no access, 200 with the diagram.
export function fakeApi(knownTokens: string[], diagrams: Record<string, FakeDiagram>) {
  const state = { calls: 0, down: false };
  const fetchFn = (async (input: unknown, init?: RequestInit) => {
    state.calls++;
    if (state.down) throw new TypeError("fetch failed");
    const url = new URL(String(input));
    const token = new Headers(init?.headers).get("Authorization") ?? "";
    if (!knownTokens.includes(token)) return new Response(null, { status: 403 });
    const id = url.searchParams.get("diagramId") ?? "";
    const diagram = diagrams[id];
    if (!diagram || diagram.projectId !== url.searchParams.get("projectId") || !diagram.members.includes(token)) {
      return new Response(JSON.stringify({ message: "not found" }), { status: 404 });
    }
    return new Response(JSON.stringify({ message: "ok", data: { id, name: diagram.name, data: diagram.data } }), { status: 200 });
  }) as typeof fetch;
  return { state, fetchFn };
}
