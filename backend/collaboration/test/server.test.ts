import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { CLOSE } from "../src/codes.ts";
import { loadConfig, type Config } from "../src/config.ts";
import { startServer, type CollabServer } from "../src/server.ts";
import { fakeApi, rawPresence, TestClient, tokenFor, until, type ClientOptions, type FakeDiagram } from "./helpers.ts";

// Three users. Alice and Bob are in one organization. Carol is in another.
const ALICE_ID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
const BOB_ID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb";
const CAROL_ID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc";
const ALICE = tokenFor({ sub: ALICE_ID });
const BOB = tokenFor({ sub: BOB_ID });
const CAROL = tokenFor({ sub: CAROL_ID });

const PROJECT = "11111111-1111-4111-8111-111111111111";
const DIAGRAM_1 = "d1d1d1d1-d1d1-4d1d-8d1d-d1d1d1d1d1d1";
const DIAGRAM_2 = "d2d2d2d2-d2d2-4d2d-8d2d-d2d2d2d2d2d2";
const AWS_DIAGRAM = "a3a3a3a3-a3a3-4a3a-8a3a-a3a3a3a3a3a3";

const blocks = (nodeId: string) => ({
  engine: "blocks",
  nodes: [{ id: nodeId, type: "vm", position: { x: 0, y: 0 }, data: { name: nodeId, fields: {} } }],
  edges: [],
});

const diagrams = (): Record<string, FakeDiagram> => ({
  [DIAGRAM_1]: { projectId: PROJECT, name: "One", data: blocks("saved-1"), members: [ALICE, BOB] },
  [DIAGRAM_2]: { projectId: PROJECT, name: "Two", data: blocks("saved-2"), members: [ALICE, BOB] },
  [AWS_DIAGRAM]: { projectId: PROJECT, name: "Old", data: { nodes: [], edges: [] }, members: [ALICE] },
});

let api: ReturnType<typeof fakeApi>;
let server: CollabServer;
let clients: TestClient[];

// Start the service on a free port, with small limits so the tests are quick.
async function start(over: Partial<Config> = {}): Promise<void> {
  server = await startServer({ ...loadConfig({}), port: 0, roomIdleMs: 0, refuseDelayMs: 0, ...over }, api.fetchFn);
}

// Open a client. It is closed again after the test.
function connect(token: string | null, diagramId = DIAGRAM_1, options: ClientOptions = {}, query = `?projectId=${PROJECT}`): TestClient {
  const client = new TestClient(`ws://127.0.0.1:${server.port}/${diagramId}${query}`, { token, ...options });
  clients.push(client);
  return client;
}

const presenceOf = (id: string, name: string) => ({ user: { id, name, color: "#c2410c" } });

beforeEach(() => {
  api = fakeApi([ALICE, BOB, CAROL], diagrams());
  clients = [];
});

afterEach(async () => {
  for (const client of clients) client.close();
  await server.close();
});

describe("the health check", () => {
  it("answers on /healthz and nowhere else", async () => {
    await start();
    const ok = await fetch(`http://127.0.0.1:${server.port}/healthz`);
    expect(ok.status).toBe(200);
    expect(await ok.json()).toEqual({ status: "ok" });
    expect((await fetch(`http://127.0.0.1:${server.port}/other`)).status).toBe(404);
  });
});

describe("refusing a connection", () => {
  it.each([
    ["another website", "http://evil.example"],
    ["no Origin header", null],
  ])("closes with 4403 for %s, and does not call the API", async (_name, origin) => {
    await start();
    const closed = await connect(ALICE, DIAGRAM_1, { origin }).waitClosed();
    expect(closed.code).toBe(CLOSE.forbiddenOrigin);
    expect(api.state.calls).toBe(0);
  });

  it("closes with 4401 when there is no login cookie", async () => {
    await start();
    expect((await connect(null).waitClosed()).code).toBe(CLOSE.unauthorized);
  });

  it("closes with 4401 when the API does not accept the token", async () => {
    await start();
    const stranger = tokenFor({ sub: ALICE_ID, forged: true });
    expect((await connect(stranger).waitClosed()).code).toBe(CLOSE.unauthorized);
  });

  it("closes with 4404 for a user from another organization, and creates no room", async () => {
    await start();
    expect((await connect(CAROL).waitClosed()).code).toBe(CLOSE.notFound);
    expect(server.rooms.size).toBe(0);
  });

  it("closes with 4404 for a diagram that does not exist", async () => {
    await start();
    const missing = "99999999-9999-4999-8999-999999999999";
    expect((await connect(ALICE, missing).waitClosed()).code).toBe(CLOSE.notFound);
  });

  it("closes with 4400 when projectId is missing", async () => {
    await start();
    expect((await connect(ALICE, DIAGRAM_1, {}, "").waitClosed()).code).toBe(CLOSE.badRequest);
  });

  it("closes with 4400 for an old AWS diagram", async () => {
    await start();
    expect((await connect(ALICE, AWS_DIAGRAM).waitClosed()).code).toBe(CLOSE.badRequest);
    expect(server.rooms.size).toBe(0);
  });

  it("closes with 4500 when the API cannot be reached", async () => {
    await start();
    api.state.down = true;
    expect((await connect(ALICE).waitClosed()).code).toBe(CLOSE.upstream);
  });

  it("closes with 4429 when the room is full", async () => {
    await start({ maxClientsPerRoom: 1 });
    await connect(ALICE).waitSynced();
    expect((await connect(BOB).waitClosed()).code).toBe(CLOSE.full);
  });

  it("waits before it closes a refused socket, and sends nothing while it waits", async () => {
    await start({ refuseDelayMs: 150 });
    const started = Date.now();
    const carol = connect(CAROL);
    const closed = await carol.waitClosed();
    expect(closed.code).toBe(CLOSE.notFound);
    expect(Date.now() - started).toBeGreaterThanOrEqual(140);
    // Carol got no part of the diagram.
    expect(carol.synced).toBe(false);
    expect(carol.nodeIds()).toEqual([]);
  });

  it("does not keep a refused socket's timer when the client leaves first", async () => {
    await start({ refuseDelayMs: 60_000 });
    const carol = connect(CAROL);
    await until(() => carol.ws.readyState === carol.ws.OPEN, "the socket to open");
    carol.ws.close();
    expect((await carol.waitClosed()).code).not.toBe(CLOSE.notFound);
  });

  it("checks every connection, not only the first one in a room", async () => {
    await start();
    await connect(ALICE).waitSynced();
    expect((await connect(CAROL).waitClosed()).code).toBe(CLOSE.notFound);
    expect(api.state.calls).toBe(2);
  });
});

describe("syncing a diagram", () => {
  it("gives a new client the saved diagram", async () => {
    await start();
    const alice = connect(ALICE);
    await alice.waitSynced();
    expect(alice.nodeIds()).toEqual(["saved-1"]);
    expect(alice.doc.getMap("meta").get("name")).toBe("One");
  });

  it("sends one person's change to the other, both ways", async () => {
    await start();
    const alice = connect(ALICE);
    const bob = connect(BOB);
    await alice.waitSynced();
    await bob.waitSynced();

    alice.addNode("from-alice");
    bob.addNode("from-bob");
    const all = ["from-alice", "from-bob", "saved-1"];
    await until(() => alice.nodeIds().length === 3 && bob.nodeIds().length === 3, "both to have three nodes");
    expect(alice.nodeIds()).toEqual(all);
    expect(bob.nodeIds()).toEqual(all);
  });

  it("gives a late joiner the edits made before they came", async () => {
    await start();
    const alice = connect(ALICE);
    await alice.waitSynced();
    alice.addNode("early");
    await until(() => server.rooms.get(DIAGRAM_1)!.doc.getMap("nodes").has("early"), "the server to get the edit");

    const bob = connect(BOB);
    await bob.waitSynced();
    expect(bob.nodeIds()).toEqual(["early", "saved-1"]);
  });

  it("keeps two diagrams apart", async () => {
    await start();
    const one = connect(ALICE, DIAGRAM_1);
    const two = connect(BOB, DIAGRAM_2);
    await one.waitSynced();
    await two.waitSynced();

    one.addNode("only-in-one");
    one.setPresence(presenceOf(ALICE_ID, "Alice"));
    // A second client in diagram 1 tells us when the change has gone around.
    const witness = connect(BOB, DIAGRAM_1);
    await until(() => witness.nodeIds().includes("only-in-one") && witness.others().length === 1, "the witness to see it");

    expect(two.nodeIds()).toEqual(["saved-2"]);
    expect(two.others()).toEqual([]);
  });

  it("lets a client catch up after it was away, and sends its own offline edits", async () => {
    await start();
    const alice = connect(ALICE);
    const bob = connect(BOB);
    await alice.waitSynced();
    await bob.waitSynced();

    bob.ws.close();
    await bob.waitClosed();
    alice.addNode("while-bob-was-away");
    bob.addNode("made-offline");

    // Bob comes back with the same document.
    const bobAgain = connect(BOB, DIAGRAM_1, { doc: bob.doc });
    const all = ["made-offline", "saved-1", "while-bob-was-away"];
    await until(() => bobAgain.nodeIds().length === 3 && alice.nodeIds().length === 3, "both to catch up");
    expect(bobAgain.nodeIds()).toEqual(all);
    expect(alice.nodeIds()).toEqual(all);
  });

  it("drops an empty room, so the next visitor gets the saved copy", async () => {
    await start();
    const alice = connect(ALICE);
    await alice.waitSynced();
    alice.addNode("never-saved");
    await until(() => server.rooms.get(DIAGRAM_1)!.doc.getMap("nodes").has("never-saved"), "the server to get the edit");
    alice.ws.close();
    await until(() => server.rooms.size === 0, "the room to be dropped");

    const bob = connect(BOB);
    await bob.waitSynced();
    expect(bob.nodeIds()).toEqual(["saved-1"]);
  });
});

describe("presence", () => {
  it("shows each person to the others, and removes them when they leave", async () => {
    await start();
    const alice = connect(ALICE);
    const bob = connect(BOB);
    await alice.waitSynced();
    await bob.waitSynced();

    alice.setPresence(presenceOf(ALICE_ID, "Alice"));
    bob.setPresence(presenceOf(BOB_ID, "Bob"));
    await until(() => alice.others().length === 1 && bob.others().length === 1, "both to see each other");
    expect(bob.others()).toEqual([presenceOf(ALICE_ID, "Alice")]);
    expect(alice.others()).toEqual([presenceOf(BOB_ID, "Bob")]);

    bob.ws.close();
    await until(() => alice.others().length === 0, "Bob to disappear");
  });

  it("tells a new client who is already here", async () => {
    await start();
    const alice = connect(ALICE);
    await alice.waitSynced();
    alice.setPresence(presenceOf(ALICE_ID, "Alice"));
    await until(() => server.rooms.get(DIAGRAM_1)!.awareness.getStates().size === 1, "the server to get the presence");

    const bob = connect(BOB);
    await until(() => bob.others().length === 1, "Bob to see Alice");
    expect(bob.others()).toEqual([presenceOf(ALICE_ID, "Alice")]);
  });

  it.each([
    ["another user's id", presenceOf(ALICE_ID, "Fake Alice")],
    ["no user", { cursor: { x: 1, y: 2 } }],
    ["an empty presence", {}],
  ])("ignores presence with %s", async (_name, presence) => {
    await start();
    const alice = connect(ALICE);
    const bob = connect(BOB);
    await alice.waitSynced();
    await bob.waitSynced();

    bob.setPresence(presence);
    // A good update sent after the bad one arrives after it.
    alice.setPresence(presenceOf(ALICE_ID, "Alice"));
    await until(() => bob.others().length === 1, "Bob to see Alice");

    expect(alice.others()).toEqual([]);
    expect(server.rooms.get(DIAGRAM_1)!.awareness.getStates().size).toBe(1);
    // Bob is still connected. Bad presence is ignored, not punished.
    expect(bob.closed).toBeNull();
  });

  it("does not let one user change or remove another user's presence", async () => {
    await start();
    const alice = connect(ALICE);
    const bob = connect(BOB);
    await alice.waitSynced();
    await bob.waitSynced();
    alice.setPresence(presenceOf(ALICE_ID, "Alice"));
    await until(() => bob.others().length === 1, "Bob to see Alice");

    // Bob tries to take over Alice's entry, then to remove it.
    const aliceEntry = alice.doc.clientID;
    bob.sendPresence(rawPresence(aliceEntry, 99, presenceOf(BOB_ID, "Bob as Alice")));
    bob.sendPresence(rawPresence(aliceEntry, 100, null));
    // Bob's own real presence goes through after the two bad ones.
    bob.setPresence(presenceOf(BOB_ID, "Bob"));
    await until(() => alice.others().length === 1, "Alice to see Bob");

    const room = server.rooms.get(DIAGRAM_1)!;
    expect(room.awareness.getStates().get(aliceEntry)).toEqual(presenceOf(ALICE_ID, "Alice"));
    expect(alice.others()).toEqual([presenceOf(BOB_ID, "Bob")]);
  });

  it("keeps a user's presence when they reconnect before the old socket is closed", async () => {
    await start();
    const alice = connect(ALICE);
    const bob = connect(BOB);
    await alice.waitSynced();
    await bob.waitSynced();
    bob.setPresence(presenceOf(BOB_ID, "Bob"));
    await until(() => alice.others().length === 1, "Alice to see Bob");

    // Bob opens a second socket with the same document, then the old one closes.
    const bobAgain = connect(BOB, DIAGRAM_1, { doc: bob.doc });
    await bobAgain.waitSynced();
    bobAgain.sendPresence(rawPresence(bob.doc.clientID, 50, presenceOf(BOB_ID, "Bob again")));
    await until(() => JSON.stringify(alice.others()).includes("Bob again"), "Alice to see the new presence");
    bob.ws.close();
    await bob.waitClosed();

    // A change from Alice goes around, so we know the close was handled.
    alice.addNode("after-close");
    await until(() => bobAgain.nodeIds().includes("after-close"), "the edit to arrive");
    expect(alice.others()).toEqual([presenceOf(BOB_ID, "Bob again")]);
  });
});

describe("closing a connection that misbehaves", () => {
  it("closes with 4400 for a text message", async () => {
    await start();
    const alice = connect(ALICE);
    await alice.waitSynced();
    alice.ws.send("hello");
    expect((await alice.waitClosed()).code).toBe(CLOSE.badRequest);
  });

  it.each([
    ["an empty message", []],
    ["a sync message of an unknown type", [0, 9]],
    ["a presence message that is cut short", [1, 200]],
  ])("closes with 4400 for %s", async (_name, bytes) => {
    await start();
    const alice = connect(ALICE);
    await alice.waitSynced();
    alice.ws.send(new Uint8Array(bytes));
    expect((await alice.waitClosed()).code).toBe(CLOSE.badRequest);
  });

  it("ignores a kind of message it does not know", async () => {
    await start();
    const alice = connect(ALICE);
    await alice.waitSynced();
    alice.ws.send(new Uint8Array([3]));
    alice.addNode("still-works");
    await until(() => server.rooms.get(DIAGRAM_1)!.doc.getMap("nodes").has("still-works"), "the edit to arrive");
    expect(alice.closed).toBeNull();
  });

  it("closes a message that is too big, and the room carries on", async () => {
    await start({ maxMessageBytes: 1024 });
    const alice = connect(ALICE);
    const bob = connect(BOB);
    await alice.waitSynced();
    await bob.waitSynced();
    alice.ws.send(new Uint8Array(4096));
    expect((await alice.waitClosed()).code).toBe(1009);

    bob.addNode("after");
    await until(() => server.rooms.get(DIAGRAM_1)!.doc.getMap("nodes").has("after"), "the edit to arrive");
  });

  it("drops a client that stops answering pings", async () => {
    await start({ pingMs: 30 });
    const dead = connect(ALICE, DIAGRAM_1, { autoPong: false });
    const live = connect(BOB);
    await dead.waitSynced();
    await live.waitSynced();
    await dead.waitClosed();
    expect(live.closed).toBeNull();
    expect(server.rooms.get(DIAGRAM_1)!.clients.size).toBe(1);
  });

  it("closes with 4401 when the login ends during the session", async () => {
    // A token for Alice that ends in 0.3 seconds.
    const soon = tokenFor({ sub: ALICE_ID, exp: Date.now() / 1000 + 0.3 });
    const all = diagrams();
    all[DIAGRAM_1]!.members.push(soon);
    api = fakeApi([soon], all);
    await start();

    const alice = connect(soon);
    await alice.waitSynced();
    expect((await alice.waitClosed()).code).toBe(CLOSE.unauthorized);
  });
});

describe("stopping the service", () => {
  it("closes every client with 1001, which tells them to reconnect", async () => {
    await start();
    const alice = connect(ALICE);
    await alice.waitSynced();
    await server.close();
    expect((await alice.waitClosed()).code).toBe(1001);
    expect(server.rooms.size).toBe(0);
  });
});
