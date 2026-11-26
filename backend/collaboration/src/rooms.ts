// Keeps one room per diagram. A room holds the shared document and who is in it.
// A room is filled from the saved copy once, by the server, when it is created.
// An empty room is dropped after a short wait. The next visitor then gets the
// last saved copy again.

import { Awareness } from "y-protocols/awareness";
import * as Y from "yjs";
import { fillDoc } from "./blocks.ts";
import type { Config } from "./config.ts";

export interface SavedDiagram {
  document: unknown;
  name: string;
}

export interface Room {
  id: string;
  doc: Y.Doc;
  // Presence: who is here. It is shared live and never saved.
  awareness: Awareness;
  clients: Set<object>;
}

type Limits = Pick<Config, "maxRooms" | "maxClientsPerRoom" | "roomIdleMs">;

export class Rooms {
  private limits: Limits;
  private rooms = new Map<string, Room>();
  // Timers for empty rooms that are waiting to be dropped.
  private idleTimers = new Map<string, NodeJS.Timeout>();

  constructor(limits: Limits) {
    this.limits = limits;
  }

  get size(): number {
    return this.rooms.size;
  }

  get(id: string): Room | undefined {
    return this.rooms.get(id);
  }

  // Add a client to a diagram's room. Creates the room if it does not exist.
  // Returns null if the room or the service is full.
  // This has no "await" in it, so two people joining at the same moment
  // can never create two rooms or fill one room twice.
  join(id: string, client: object, saved: SavedDiagram): Room | null {
    let room = this.rooms.get(id);
    if (room) {
      if (room.clients.size >= this.limits.maxClientsPerRoom && !room.clients.has(client)) return null;
    } else {
      if (this.rooms.size >= this.limits.maxRooms) return null;
      room = this.create(id, saved);
    }

    // Someone is here again, so stop the drop timer.
    const timer = this.idleTimers.get(id);
    if (timer) {
      clearTimeout(timer);
      this.idleTimers.delete(id);
    }
    room.clients.add(client);
    return room;
  }

  // Remove a client. If the room is now empty, drop it after the idle time.
  leave(room: Room, client: object): void {
    room.clients.delete(client);
    if (room.clients.size > 0 || this.rooms.get(room.id) !== room) return;
    if (this.limits.roomIdleMs === 0) {
      this.drop(room);
      return;
    }
    if (this.idleTimers.has(room.id)) return;
    const timer = setTimeout(() => {
      this.idleTimers.delete(room.id);
      if (room.clients.size === 0) this.drop(room);
    }, this.limits.roomIdleMs);
    this.idleTimers.set(room.id, timer);
  }

  // Drop every room. Used when the service stops.
  closeAll(): void {
    for (const timer of this.idleTimers.values()) clearTimeout(timer);
    this.idleTimers.clear();
    for (const room of [...this.rooms.values()]) this.drop(room);
  }

  private create(id: string, saved: SavedDiagram): Room {
    const doc = new Y.Doc();
    fillDoc(doc, saved.document, saved.name);
    const awareness = new Awareness(doc);
    // The server is not a person, so it has no presence of its own.
    awareness.setLocalState(null);
    const room: Room = { id, doc, awareness, clients: new Set() };
    this.rooms.set(id, room);
    return room;
  }

  private drop(room: Room): void {
    this.rooms.delete(room.id);
    room.awareness.destroy();
    room.doc.destroy();
  }
}
