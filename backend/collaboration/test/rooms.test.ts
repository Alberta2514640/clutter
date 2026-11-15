import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { readDoc } from "../src/blocks.ts";
import { Rooms, type SavedDiagram } from "../src/rooms.ts";

const limits = { maxRooms: 2, maxClientsPerRoom: 2, roomIdleMs: 1000 };

// A saved diagram with one VM node.
const saved = (nodeName: string): SavedDiagram => ({
  name: "Demo",
  document: { engine: "blocks", nodes: [{ id: "n1", type: "vm", position: { x: 0, y: 0 }, data: { name: nodeName, fields: {} } }], edges: [] },
});

// Any object can stand in for a client here.
const client = () => ({});

let rooms: Rooms;

beforeEach(() => {
  vi.useFakeTimers();
  rooms = new Rooms(limits);
});

afterEach(() => {
  rooms.closeAll();
  vi.useRealTimers();
});

describe("join", () => {
  it("creates the room and fills it from the saved copy", () => {
    const room = rooms.join("d1", client(), saved("web"));
    expect(room).not.toBeNull();
    expect(readDoc(room!.doc).document.nodes[0]!.data.name).toBe("web");
    expect(rooms.size).toBe(1);
  });

  it("gives the second person the same room and does not fill it again", () => {
    const first = rooms.join("d1", client(), saved("web"));
    first!.doc.getMap("meta").set("name", "Edited live");
    const second = rooms.join("d1", client(), saved("a different copy"));
    expect(second).toBe(first);
    expect(readDoc(second!.doc)).toMatchObject({ name: "Edited live" });
    expect(readDoc(second!.doc).document.nodes[0]!.data.name).toBe("web");
  });

  it("keeps two diagrams apart", () => {
    const one = rooms.join("d1", client(), saved("one"));
    const two = rooms.join("d2", client(), saved("two"));
    expect(one!.doc).not.toBe(two!.doc);
    one!.doc.getMap("meta").set("name", "changed");
    expect(readDoc(two!.doc).name).toBe("Demo");
  });

  it("gives the server no presence of its own", () => {
    const room = rooms.join("d1", client(), saved("web"));
    expect(room!.awareness.getStates().size).toBe(0);
  });

  it("refuses a new person when the room is full", () => {
    const a = client();
    rooms.join("d1", a, saved("web"));
    rooms.join("d1", client(), saved("web"));
    expect(rooms.join("d1", client(), saved("web"))).toBeNull();
    // Someone already in the room is not refused.
    expect(rooms.join("d1", a, saved("web"))).not.toBeNull();
  });

  it("refuses a new room when the service is full, but not an existing one", () => {
    rooms.join("d1", client(), saved("web"));
    rooms.join("d2", client(), saved("web"));
    expect(rooms.join("d3", client(), saved("web"))).toBeNull();
    expect(rooms.size).toBe(2);
    expect(rooms.join("d1", client(), saved("web"))).not.toBeNull();
  });
});

describe("leave", () => {
  it("keeps the room while someone is still in it", () => {
    const a = client();
    const room = rooms.join("d1", a, saved("web"))!;
    rooms.join("d1", client(), saved("web"));
    rooms.leave(room, a);
    vi.advanceTimersByTime(5000);
    expect(rooms.get("d1")).toBe(room);
  });

  it("drops an empty room after the idle time", () => {
    const a = client();
    const room = rooms.join("d1", a, saved("web"))!;
    rooms.leave(room, a);
    vi.advanceTimersByTime(999);
    expect(rooms.get("d1")).toBe(room);
    vi.advanceTimersByTime(1);
    expect(rooms.get("d1")).toBeUndefined();
    expect(rooms.size).toBe(0);
  });

  it("keeps the room and its edits when someone comes back in time", () => {
    const a = client();
    const room = rooms.join("d1", a, saved("web"))!;
    room.doc.getMap("meta").set("name", "Edited live");
    rooms.leave(room, a);
    vi.advanceTimersByTime(500);
    const again = rooms.join("d1", client(), saved("web"));
    vi.advanceTimersByTime(5000);
    expect(again).toBe(room);
    expect(rooms.get("d1")).toBe(room);
    expect(readDoc(room.doc).name).toBe("Edited live");
  });

  it("gives the next visitor the saved copy after the room was dropped", () => {
    const a = client();
    const room = rooms.join("d1", a, saved("web"))!;
    room.doc.getMap("meta").set("name", "Edited live, never saved");
    rooms.leave(room, a);
    vi.advanceTimersByTime(1000);
    const fresh = rooms.join("d1", client(), saved("web"))!;
    expect(fresh).not.toBe(room);
    expect(readDoc(fresh.doc).name).toBe("Demo");
  });

  it("drops the room at once when the idle time is 0", () => {
    const quick = new Rooms({ ...limits, roomIdleMs: 0 });
    const a = client();
    const room = quick.join("d1", a, saved("web"))!;
    quick.leave(room, a);
    expect(quick.size).toBe(0);
  });

  it("frees a place in the service when a room is dropped", () => {
    const a = client();
    const room = rooms.join("d1", a, saved("web"))!;
    rooms.join("d2", client(), saved("web"));
    rooms.leave(room, a);
    vi.advanceTimersByTime(1000);
    expect(rooms.join("d3", client(), saved("web"))).not.toBeNull();
  });
});

describe("closeAll", () => {
  it("drops every room and stops the timers", () => {
    const a = client();
    const room = rooms.join("d1", a, saved("web"))!;
    rooms.join("d2", client(), saved("web"));
    rooms.leave(room, a);
    rooms.closeAll();
    expect(rooms.size).toBe(0);
    expect(vi.getTimerCount()).toBe(0);
  });
});
