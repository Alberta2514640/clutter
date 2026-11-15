# collaboration

The live sync service. It lets several signed-in users edit one building-block diagram at the same time. It is a small Node service that speaks the Yjs WebSocket protocol, with one room per diagram.

It is for the Fall 2026 demo and runs on one machine only. `npm run dev` starts it with the rest of Clutter. The website does not connect to it yet.

## How it fits

```text
browser (blocks editor) --WebSocket--> collaboration (127.0.0.1:8791)
                                            |
                                            | GET /diagram, with the user's own token
                                            v
                                       A's API (127.0.0.1:8787)
```

- The browser opens `ws://localhost:8791/<diagramId>?projectId=<projectId>`. It sends the `clutter_token` cookie by itself.
- For every connection, the service calls A's `GET /diagram` with that token. A decides if the user may open the diagram. The service has no access rule of its own.
- The service holds no JWT secret and no database password. It never writes to the database. Saving stays with the editor's Save button.
- Only building-block diagrams (`engine: "blocks"`) get a room. A's older AWS diagrams are refused.

## Run it

Use Node 22 (`nvm use` at the repo root). Node runs the TypeScript files directly, so there is no build step.

`npm run dev` at the repo root starts the service with the rest of Clutter, and `npm run stop` stops it. `npm run setup` installs its packages. The start message has a `Live sync:` line. It shows the address, or says why the service is off. The service is optional: Clutter starts without it (`backend/devserver/README.md` § Live sync locally).

To run it by hand, for example while changing its code:

```bash
cd backend/collaboration
npm ci
npm start
```

`npm start` prints `collaboration: listening on ws://127.0.0.1:8791`. Stop it with Ctrl+C. It needs A's API running for the access check (`npm run dev:backend`). A later `npm run dev` leaves a service that was started by hand as it is.

Check that it runs:

```bash
curl http://127.0.0.1:8791/healthz     # {"status":"ok"}
```

## Check it

Run these from `backend/collaboration`.

```bash
npm run typecheck     # no output after the "tsc --noEmit" line means no errors
npm test              # Test Files  5 passed (5), Tests  105 passed (105)
```

The tests need no database and no running app. They use a fake of A's API and real WebSocket clients on a free port.

The tests cover the settings, the access check and its close codes, the conversion and the saved hash, rooms, and the running service (refusals, syncing, presence, bad messages and pings).

Last checked against the real app on 2026-11-15, with A's API running locally and the real `y-websocket` client: two clients synced a saved blocks diagram, edits and presence went both ways, and each refusal gave the expected close code.

Not covered yet:

- A second real user from another organization. The local sign-in has one user. This case is tested only with the fake API.
- A real browser. The checks so far run the client in Node.

## Settings

All settings are optional environment variables. `loadConfig` owns them and their limits. A bad value stops the service at start with a message that names the variable.

| Variable | Default | Meaning |
| --- | --- | --- |
| `COLLAB_HOST` | `127.0.0.1` | The address to listen on. Only `127.0.0.1`, `::1` and `localhost` are allowed. |
| `COLLAB_PORT` | `8791` | The port to listen on. |
| `COLLAB_API_URL` | `http://localhost:8787` | A's API, used for the access check. |
| `COLLAB_ALLOWED_ORIGINS` | `http://localhost:3000` | The website addresses that may connect, separated by commas. |
| `COLLAB_ACCESS_TIMEOUT_MS` | `5000` | How long to wait for A's API. |
| `COLLAB_MAX_MESSAGE_BYTES` | `1048576` | The largest message a client may send. |
| `COLLAB_MAX_CLIENTS_PER_ROOM` | `20` | The most connections in one room. |
| `COLLAB_MAX_ROOMS` | `100` | The most rooms at one time. |
| `COLLAB_ROOM_IDLE_MS` | `30000` | How long an empty room is kept. |
| `COLLAB_PING_MS` | `30000` | How often each client is pinged. |
| `COLLAB_REFUSE_DELAY_MS` | `1000` | How long a refused socket is held before it is closed. |

## Close codes

A browser cannot read the HTTP status of a failed WebSocket. So the service always opens the socket, and closes it with one of these codes when the answer is no. The `CLOSE` list in the code owns them.

| Code | Meaning |
| --- | --- |
| 4400 | `projectId` is missing, the diagram is not a building-block diagram, or a message could not be read |
| 4401 | not signed in, or the login ended during the session |
| 4403 | the page's Origin is not allowed |
| 4404 | the diagram does not exist, or the user has no access to it |
| 4429 | the room or the service is full |
| 4500 | A's API could not be reached |

The `y-websocket` client retries 0.1 seconds after every close. The editor must stop it on 4400, 4401, 4403 and 4404. The service also waits `COLLAB_REFUSE_DELAY_MS` before it closes a refused socket, to slow a client that does not stop.

## What happens on a connection

1. `startServer` checks the page's Origin.
2. `checkAccess` calls A's `GET /diagram` with the user's token. The client cannot send anything while this runs.
3. If the answer is no, the socket is opened and then closed with a close code.
4. If the answer is yes, `Rooms.join` puts the client in the diagram's room. The first client makes the room, and `fillDoc` fills it from the saved diagram.
5. `handleConnection` sends the client the document and who is in the room. From then on it passes every change to everyone else in the room.
6. When the last client leaves, the room is dropped after `COLLAB_ROOM_IDLE_MS`.

## The shared document

The saved format belongs to B (`frontend/clutter/src/lib/features/diagram/blocksDocument.ts`). The service does not change it. The shared Yjs document holds the same facts in three maps:

```text
nodes : node id -> { kind, name, position: {x, y}, fields: a map with one entry per field }
edges : edge id -> { source, target }
meta  : { name, saved: { hash, by, at } }
```

- Only the service fills a room, from the saved diagram. A browser never does.
- `meta.saved.hash` is the hash of the diagram as it was last saved. The editor shows "Saved" when the current hash equals it. `savedHash` defines the hash.
- The sample file in `test/fixtures` holds one sample diagram in both formats, with its hash. The frontend's copy of the conversion must pass against the same file.

## Rules to keep

- **Every connection is checked**, not only the first one in a room. Knowing a diagram id grants nothing.
- **Never log a token, a cookie or a diagram's content.** The service logs its start line and the text of an unexpected error, and nothing else.
- **Presence must name the signed-in user.** The service ignores presence whose `user.id` is not the connected user, and does not let one user change another user's presence.
- **Keep Yjs on version 13**, the same as `frontend/clutter`. A server on Yjs 14 does not work with Yjs 13 clients.
- **`vitest` is pinned to 4.0.8**, the version the frontend uses. That version has two open advisories in its mocking helper. The fixed version, 4.1.12, fails to install with npm 10.9.9. The packages the service runs with have no known advisories (`npm audit --omit=dev`).

## Known limits

- Edits that nobody saved are lost when a room is dropped, 30 seconds after the last person leaves. This is by design for the demo: the saved diagram stays the single source for deploys.
- A user who loses access to a diagram stays in its room until they disconnect or their login ends.
- It listens on this machine only and has no TLS. Hosting it is a later decision (ADR-023).
