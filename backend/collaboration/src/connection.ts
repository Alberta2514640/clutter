// Handles one WebSocket after it was allowed into a room.
// It passes document changes and presence between everyone in the room.
// The message format is the standard Yjs one, the same as the y-websocket client.

import * as decoding from "lib0/decoding";
import * as encoding from "lib0/encoding";
import type { WebSocket } from "ws";
import { applyAwarenessUpdate, encodeAwarenessUpdate, removeAwarenessStates } from "y-protocols/awareness";
import { readSyncMessage, writeSyncStep1, writeUpdate } from "y-protocols/sync";
import { CLOSE } from "./codes.ts";
import type { Room } from "./rooms.ts";

// The first number in every message says what kind it is.
const MESSAGE_SYNC = 0;
const MESSAGE_AWARENESS = 1;

// The longest wait Node allows for a timer, about 24 days.
const MAX_TIMER_MS = 2 ** 31 - 1;

export interface ConnectionUser {
  userId: string;
  expiresAt: number | null;
}

export interface ConnectionOptions {
  pingMs: number;
  // Called once when the socket closes.
  onClose: () => void;
}

// Who owns a presence entry. Presence entries are keyed by a Yjs client id.
interface Owner {
  client: WebSocket;
  userId: string;
}

// Extra state we keep for each room.
interface RoomState {
  owners: Map<number, Owner>;
}

const roomStates = new WeakMap<Room, RoomState>();

const isRecord = (v: unknown): v is Record<string, unknown> => !!v && typeof v === "object" && !Array.isArray(v);

// Send a message to one client. If sending fails, drop that client.
function send(client: WebSocket, message: Uint8Array): void {
  if (client.readyState !== client.OPEN) return;
  client.send(message, (err) => {
    if (err) client.terminate();
  });
}

// Send a message to everyone in the room except the one who caused it.
function broadcast(room: Room, message: Uint8Array, except: unknown): void {
  for (const client of room.clients) {
    if (client !== except) send(client as WebSocket, message);
  }
}

// Set up a room the first time someone connects to it.
// From then on, every change to the document or to presence is sent to the room.
function stateOf(room: Room): RoomState {
  const existing = roomStates.get(room);
  if (existing) return existing;
  const state: RoomState = { owners: new Map() };
  roomStates.set(room, state);

  room.doc.on("update", (update: Uint8Array, origin: unknown) => {
    const encoder = encoding.createEncoder();
    encoding.writeVarUint(encoder, MESSAGE_SYNC);
    writeUpdate(encoder, update);
    broadcast(room, encoding.toUint8Array(encoder), origin);
  });

  room.awareness.on("update", (changes: { added: number[]; updated: number[]; removed: number[] }, origin: unknown) => {
    for (const id of changes.removed) state.owners.delete(id);
    const changed = changes.added.concat(changes.updated, changes.removed);
    const encoder = encoding.createEncoder();
    encoding.writeVarUint(encoder, MESSAGE_AWARENESS);
    encoding.writeVarUint8Array(encoder, encodeAwarenessUpdate(room.awareness, changed));
    broadcast(room, encoding.toUint8Array(encoder), origin);
  });

  return state;
}

// Check a presence update before we accept it. Returns the client ids it sets.
// Returns null if the update must be ignored. The rules:
//   1. The "user.id" in the presence must be the signed-in user.
//   2. A user cannot change a presence entry that belongs to another user.
// Without these rules, anyone in the room could pretend to be someone else.
function checkPresence(update: Uint8Array, userId: string, owners: Map<number, Owner>): number[] | null {
  const decoder = decoding.createDecoder(update);
  const count = decoding.readVarUint(decoder);
  const claimed: number[] = [];
  for (let i = 0; i < count; i++) {
    const clientId = decoding.readVarUint(decoder);
    decoding.readVarUint(decoder); // The clock. We do not need it here.
    const presence: unknown = JSON.parse(decoding.readVarString(decoder));

    const owner = owners.get(clientId);
    if (owner && owner.userId !== userId) return null;
    // Null means "this person left". Anything else must name the right user.
    if (presence === null) continue;
    if (!isRecord(presence) || !isRecord(presence.user) || presence.user.id !== userId) return null;
    claimed.push(clientId);
  }
  return claimed;
}

// Start serving one client that is already in the room.
export function handleConnection(client: WebSocket, room: Room, user: ConnectionUser, options: ConnectionOptions): void {
  const state = stateOf(room);

  const onMessage = (data: Uint8Array): void => {
    const decoder = decoding.createDecoder(data);
    const kind = decoding.readVarUint(decoder);

    if (kind === MESSAGE_SYNC) {
      const encoder = encoding.createEncoder();
      encoding.writeVarUint(encoder, MESSAGE_SYNC);
      // This applies the client's changes and writes our answer, if there is one.
      readSyncMessage(decoder, encoder, room.doc, client);
      // A length of 1 means there is only the message kind and no answer.
      if (encoding.length(encoder) > 1) send(client, encoding.toUint8Array(encoder));
      return;
    }

    if (kind === MESSAGE_AWARENESS) {
      const update = decoding.readVarUint8Array(decoder);
      const claimed = checkPresence(update, user.userId, state.owners);
      if (claimed === null) return;
      // If the same user reconnects, the new socket takes over the entry.
      for (const id of claimed) state.owners.set(id, { client, userId: user.userId });
      applyAwarenessUpdate(room.awareness, update, client);
    }
    // Other kinds of message are ignored.
  };

  client.on("message", (data, isBinary) => {
    // Yjs messages are always binary and arrive as one Buffer.
    if (!isBinary || !Buffer.isBuffer(data)) {
      client.close(CLOSE.badRequest, "bad message");
      return;
    }
    try {
      onMessage(new Uint8Array(data.buffer, data.byteOffset, data.byteLength));
    } catch {
      // The message could not be read. Do not log it, it may hold diagram data.
      // Note: a broken change inside a good message does not get here.
      // The Yjs library skips that one by itself.
      client.close(CLOSE.badRequest, "bad message");
    }
  });

  // Ping the client from time to time. If it did not answer the last ping, drop it.
  let alive = true;
  client.on("pong", () => {
    alive = true;
  });
  const pingTimer = setInterval(() => {
    if (!alive) {
      client.terminate();
      return;
    }
    alive = false;
    client.ping();
  }, options.pingMs);

  // Close the socket when the login ends. The editor then asks the user to sign in.
  let expiryTimer: NodeJS.Timeout | undefined;
  if (user.expiresAt !== null) {
    const wait = Math.min(Math.max(user.expiresAt - Date.now(), 0), MAX_TIMER_MS);
    expiryTimer = setTimeout(() => client.close(CLOSE.unauthorized, "sign-in expired"), wait);
  }

  // Without this, a socket error would crash the service. "close" always follows.
  client.on("error", () => {});

  client.on("close", () => {
    clearInterval(pingTimer);
    clearTimeout(expiryTimer);
    // Remove this client's presence, so the others stop seeing them.
    const mine: number[] = [];
    for (const [id, owner] of state.owners) {
      if (owner.client === client) mine.push(id);
    }
    room.clients.delete(client);
    removeAwarenessStates(room.awareness, mine, null);
    options.onClose();
  });

  // Start the sync. The client answers with what it has, and we send what it lacks.
  const encoder = encoding.createEncoder();
  encoding.writeVarUint(encoder, MESSAGE_SYNC);
  writeSyncStep1(encoder, room.doc);
  send(client, encoding.toUint8Array(encoder));

  // Tell the new client who is already here.
  const present = [...room.awareness.getStates().keys()];
  if (present.length > 0) {
    const presence = encoding.createEncoder();
    encoding.writeVarUint(presence, MESSAGE_AWARENESS);
    encoding.writeVarUint8Array(presence, encodeAwarenessUpdate(room.awareness, present));
    send(client, encoding.toUint8Array(presence));
  }
}
