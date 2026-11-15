# Frontend

The Clutter web app: a Next.js 16 App Router project with React 19, React Flow (`@xyflow/react`) for the canvas, TanStack Query for server data, Zustand for UI state, Tailwind 4 and shadcn/ui components. It was bootstrapped with `create-next-app`.

Setup for the whole repo is in the root `README.md`. This page covers the frontend only. Commands run from the repo root.

## Run it

Use npm. The Dockerfile runs `npm ci`.

| Variant | Command | Result on 2026-11-04 |
| --- | --- | --- |
| With the local backend (the normal way) | `npm run dev` at the repo root, then open <http://localhost:3000/login> | works; checked in a headless browser |
| The website alone, against a backend you run yourself | `npm run dev:backend` once, then `npm run dev:frontend` in another terminal (Ctrl+C there stops only the website) | works; checked in a headless browser on 2026-11-14 |
| Production build with the local backend | `npm run dev:backend`, then `NEXT_PUBLIC_API_ENDPOINT=http://localhost:8787 NEXT_PUBLIC_ADAPTER_API_ENDPOINT=http://localhost:8790 npm --prefix frontend/clutter run build`, copy `.next/static` and `public` into `.next/standalone`, then in `frontend/clutter/.next/standalone` run `PORT=3100 HOSTNAME=127.0.0.1 node server.js` | works; checked in a headless browser |
| Alone, no backend | `NEXT_PUBLIC_API_ENDPOINT=http://localhost:9999 npm --prefix frontend/clutter run dev` | `/` and `/login` render; everything else redirects to `/login` |
| Docker image | `frontend/build-image.sh`, with `IMAGE_TAG` and the four `NEXT_PUBLIC_` build values (`npm run deploy -- prod --site-image` runs it with the cloud's values) | the Dockerfile takes A's and B's addresses, the Google client id and the contact address as build arguments with no default, and stops when one is missing; CI builds it with stand-in values (`.github/workflows/docker-image.yml`) |

Two variables name the backends, and Next reads both at build time, so a production build must be rebuilt to point somewhere else:

| Variable | Is | Locally (`dev.sh` sets both) |
| --- | --- | --- |
| `NEXT_PUBLIC_API_ENDPOINT` | A's API (Lambdas behind API Gateway) | `http://localhost:8787` |
| `NEXT_PUBLIC_ADAPTER_API_ENDPOINT` | B API, the `/v1` service behind B's screens (`backend/adapter/README.md`) | `http://localhost:8790` |

Without `NEXT_PUBLIC_ADAPTER_API_ENDPOINT`, A's screens work and B's screens say the deployment service is not configured.

## Check it

```bash
npx --prefix frontend/clutter tsc --noEmit -p frontend/clutter     # 0 errors
npm --prefix frontend/clutter run lint                             # 0 errors, 3 warnings (unused variables in A's logs panel and in useCreateDiagram)
(cd frontend/clutter && npx vitest run)                            # 21 files, 141 tests
NEXT_PUBLIC_API_ENDPOINT=http://localhost:8787 NEXT_PUBLIC_ADAPTER_API_ENDPOINT=http://localhost:8790 npm --prefix frontend/clutter run build   # 21 routes
```

The Vitest files sit beside the code in `src/lib/features/` (the diagram types and name rule, the building-block document, the manifest-to-form and manifest-to-display mappings, the compatibility line, the deploy buttons' reasons and "Add deployment" cases, the "View as" logic, the `clouds.yaml` reader, how a deployment setting chooses between the cloud's list and a text box, the function code upload's file check and progress states, and the local-build check behind **Link fake account**, which is never true in a production build). There is no `test` script; CI runs them with `npx vitest run` (`.github/workflows/checks.yml`). Testing Library and Playwright are installed; no test uses them yet.

## B's screens

B's screens live in the same website as A's and call B API with the same token.

| Screen | Where | What it does |
| --- | --- | --- |
| Building-block editor | `/projects/[projectId]/diagram/[diagramId]`, for a diagram whose saved document has `engine: "blocks"` | The palette, block forms and allowed connections come from B's catalog (each cloud's manifest). It saves through A's `PUT /diagram`. Any other diagram opens A's editor, unchanged except for a thin bar above it that names it the older AWS editor and links back to the project. Below 1024 pixels wide, the page says the editor needs a wider screen and links back to the project (the editor stays mounted, so unsaved work survives a narrower window). Beside Save, the editor always says "Saved", "Unsaved changes" or "Saving…". A new, empty diagram opens on the Block tab with a three-step hint in the canvas (add a block, connect blocks, then Deploy) that goes once a block exists; a diagram with blocks opens on the Deploy tab. Edges are drawn between the sides of two blocks that face each other, with an arrow for their direction. With unsaved changes, the editor asks before any link or the back button leaves it, and the browser asks before the tab closes or reloads. "View as" in the top bar shows the diagram as plain blocks, or as one cloud builds it (each cloud the diagram deploys to or the organization connects to): each block shows that cloud's service icon and name, and a block the cloud does not offer, or that is left out of its deployments, is dimmed with "Not on (cloud)" or "Left out". The choice is kept per diagram in this browser. In the blocks view, each block carries a small logo per cloud the diagram deploys to, faded with a small cross where it is not built. Each palette block lists every offered cloud by name, crossed out where that cloud does not build it. A VM's form has a "Startup script" text area (a monospace, multi-line field with the help line "Runs once as root when the VM first starts. Use it to install and start your app.", capped at 16 KB). A function's form has a "Handler" field and a "Code" section: "Upload .zip" for a saved function block, a progress bar while the file goes up, then the stored file's name, size and the start of its SHA-256, with Replace and Remove; it says when the code is loading, when there is none (the function then runs placeholder code), and why an upload or a removal failed (B's own message, such as a file that is not a zip, over 50 MB, or a block not saved yet). The file is checked in the browser first (a `.zip` name, not empty, at most 50 MB). |
| Deploy panel | the editor's Deploy tab | With no connection in the organization, one "Connect a cloud" button to Connections. Otherwise one row per deployment (one per linked cloud): its compatibility line (one line per cloud, in the order a run builds them, when blocks are placed on other clouds), a toggle to leave unsupported blocks out of that deployment, a note per key a permission edge makes ("Clutter creates a key that can only read files and puts it on web. ..."), "Where each block runs" with a Cloud choice per block (default: the deployment's own connection; the organization's other working connections; it can change only while the deployment holds nothing, and says why otherwise) and each placed cloud's own settings, the cloud's settings (where the connection's cloud lists its own values, such as OpenStack's images, flavors and external networks, a setting is a list of them, read when the settings open; "Type another name…" still allows a typed value, and when there is no list the text box says why), Preview Terraform, Plan and Deploy, then apart from them Destroy (shown while it may hold resources, with a confirm step) and Remove (always shown; off until it holds nothing, new or destroyed; removing frees the connection for a new deployment), live status, each cloud's progress in a run on several clouds (including "rolled back" after a later cloud failed), and the run log (monospace, opened at its end, with a result line such as "Apply complete: 24 added" above it). The card's header names every cloud a split deployment builds on. The compatibility line says "every block can be built here" rather than "ready", since it says nothing about what is deployed. The Terraform preview groups a split deployment's files by connection, with Copy and Download for the open file. A line under the buttons says what Plan, Deploy and Destroy do, and a button that is off says why, as a tooltip and as a line under the buttons. Plan, Deploy and Preview use the saved diagram, so unsaved changes hold them back; Destroy does not read the diagram. "Add deployment" says when there is no connection, when every connection already has a deployment here, and which free connection is not working. The Connections links choose the diagram's organization first. |
| Connections | `/settings/organization/connections` (sidebar and the organization settings tabs) | List, add (a form built from the cloud's credential spec; secrets are password fields and never shown again), verify and delete the organization's connections. For a cloud linked outside B (AWS, through A's CloudFormation flow), the form lists the linked accounts to use without pasting anything, and links a new account inline in two steps (open AWS to create the role, paste its ARN); with none linked, the two steps are open at once. In a development build on the local gateway, while the organization has no linked AWS account, the flow also shows the local-only **Link fake account** button (the same component as A's popup, `components/common/LinkFakeAccount.tsx`); it links an account at once, shows the same success notice, and chooses the account. A production build drops the button. "Manage AWS account links" goes to the page that lists and removes them. For a cloud whose credential fields are a `clouds.yaml` entry's application credential (OpenStack), the user can drop the `clouds.yaml` (and `secure.yaml`) the cloud's dashboard downloads: it is read in the browser, never uploaded or kept, and fills the auth URL (adding `/v3` when it is missing), the credential ID and secret, the region and an empty name. With several clouds in the file, the user picks one. A password entry is refused with how to create an application credential. |

Each cloud's logo and service icons come from keys in its manifest, resolved to files in `public/clouds/`; that folder's README lists each file's source and licence, and why AWS and OpenStack show a neutral logo with their name.

"New diagram" (the project's header and its empty state) opens a chooser: Blank, or one of B's examples (`GET /v1/examples`, each with its level and the clouds it fits). Either way it creates a building-block diagram: A's create route makes it, then the editor's marker, with the example's blocks and links, is saved with `PUT /diagram`. The project's diagram list marks a diagram without the marker "AWS only, older editor". The sidebar has an organization switcher and "New organization"; a new organization becomes the current one. The dashboard, new project, Connections, account-link and organization settings pages use the chosen organization, kept per browser and falling back to the first; pages inside a project use that project's own organization, so switching organization there goes to the dashboard. Deleting an organization moves to another of the user's organizations, and signs out only when none remain.

To see them locally: `npm run dev` at the repo root, sign in at <http://localhost:3000/dev/login>, create an organization and a project, then "New diagram". On Connections, add a Mock cloud connection with the token `mock-valid-token`, or choose Amazon Web Services and press **Link fake account**, then Check and connect. The mock builds nothing and has no Terraform preview. A local AWS deployment previews real Terraform; its runs apply on LocalStack when it was running at `npm run dev`, and otherwise stop with "needs a real AWS account". The full walk-through is the root `README.md` § A full local tour.

## What was checked in a browser

With the devserver and a headless Chrome, 2026-11-04:

| Page | Result |
| --- | --- |
| `/dev/login` (devserver only) | signs in, lands on `/dashboard` |
| `/dashboard` | renders projects, diagram and member counts, the project list and recent activity from the local database |
| `/projects/[projectId]/diagrams` | renders the real project and its diagrams |
| `/settings/organization/aws-account` | renders |
| `/projects/[projectId]/diagram/[diagramId]` (the canvas) | shows "AWS account required" until the organization has a linked account (locally, press **Link fake account** on that popup); then renders the palette with 5 resources and Save, Deploy and Destroy |
| Dropping nodes, naming them in the config panel, saving, reloading | works; both nodes persist (checked 2026-11-04 with scripted drag events) |
| Saving generates Terraform, and Preview Terraform reads it | works; the panel text equals the file on disk, with download. The button is disabled while there are unsaved changes |
| Deploy, Destroy, Ansible jobs | fail at their ECS or SQS call, as expected locally |

B's screens, with `npm run dev` and headless Chromium (Playwright 1.56.1), 2026-11-13. The labels are as they were that day: the button is now "New diagram", and the compatibility line now says "every block can be built here" where it said "ready" (§ B's screens).

| Step | Result |
| --- | --- |
| "Create New Diagram" | opens B's editor; the palette lists the 8 blocks the connected clouds offer |
| Six blocks (VM, network, firewall, volume, bucket, function) and their edges, save, reload | the saved document has `engine: "blocks"`, the six kinds and four edges; an edge no manifest allows is refused |
| Connections: Mock cloud, a wrong token, then "Use a linked account" for AWS | the mock verifies (its token is a password field); the wrong token shows the cloud's refusal; the linked fake AWS account verifies without pasting anything |
| Mock deployment | "Mock cloud: 2 blocks not supported (Function, Bucket), 1 other problem"; leaving both out makes it ready; Preview says the mock has none; Plan, Deploy and Destroy complete with live status and logs |
| Delete a connection with a live deployment | refused with "This connection has live deployments. Destroy them first." |
| AWS deployment | "Amazon Web Services: ready"; Preview shows 5 files and no credential; Deploy stops with "This needs a real AWS account." |
| An AWS diagram from `npm run seed` | opens in A's editor with its 5 AWS nodes |
| Organization switcher | switches the dashboard, projects and Connections page to the chosen organization |

## Where things are

- The public pages: the landing page `/` (`src/app/page.tsx`), `/privacy` and `/terms`, built from `src/components/landing/` with product screenshots in `public/landing/`; with `robots.ts`, `sitemap.ts`, `opengraph-image.png` and `icon.png` in `src/app/`, they are the only pages search engines index. Their contact address is the build setting `NEXT_PUBLIC_CONTACT_EMAIL`.
  - `public/landing/` holds WebP screenshots taken at device scale 2 from the running app (`npm run dev` with LocalStack, so the AWS run log is a real apply) and a muted demo clip (`demo.webm`, `demo.mp4`, about 18 seconds, 0.5 to 0.6 MB each, recorded with Playwright's video capture) with its still `demo-poster.webp`. `shots.ts` owns each file's size and alt text. Retake them all together when a screen they show changes.
  - The page's interactive parts are two small client components: `ViewAsDemo` (Blocks, AWS and OpenStack buttons with `aria-pressed` that swap three same-size screenshots) and `DemoClip` (plays only while on screen, never by itself under reduced motion, with a Play and Pause button). Motion is plain CSS in `motion.module.css`, and none runs under `prefers-reduced-motion`.
  - Lighthouse 13.6.0 on the production build, 2026-11-14: mobile performance 94 (three warm runs; 92 cold), desktop 100, and 100 for accessibility, best practices and SEO on both. `/` loads 453 KB in 31 requests on mobile before the clip plays. Google's sign-in script loads only on `/login` (its own layout), and Geist Mono is not preloaded. What is left is the render-blocking CSS (about 400 ms in Lighthouse's estimate); Next's experimental `inlineCss` would remove it, but it is global and costs returning users their cached stylesheet, so it is off.
- `src/app/`: routes, in four groups.
  - `(auth)`: `login` (signs in and signs up with Google) and `onboarding/create-org` (the first organization after sign-in).
  - `(dashboard)`: `dashboard`, `projects` (redirects to the dashboard), `projects/new`, and per project `diagrams` and `settings` (rename, delete).
  - `(diagram)`: the canvas at `projects/[projectId]/diagram/[diagramId]/`. `EditorSwitch` picks the editor from the saved document. A's `_components/` hold `DiagramEditor`, `Palette` (from `GET /resources`), `TopNav`, `TerraformPreview` (reads `GET /terraform-engine/files`), `AwsAccountRequiredModal`, `nodes/AwsServiceNode`, `nodes/ConfigPanel` and the `logs/` panel. B's editor and deploy panel are in `_components/blocks/`.
  - `(settings)`: `settings/organization/aws-account` (lists and removes every AWS account link, and links another; under the Connections tab, which shows the same link flow inline), `connections` (B's Connections page), `manage` (details, the member count and the signed-in member; the backend has no member names or invites yet) and `create-org` (a further organization).
- `src/lib/features/<feature>/`: one folder per backend area, each with `api.ts`, `types.ts`, `keys.ts`, `hooks.ts` and sometimes `uiStore.ts`. Features: `user` (sign-in, token, and `session.ts`: the fetch helper for A's API and what an expired session looks like), `organization` (the current-organization choice, and `rules.ts` for the name and description limits), `projects`, `diagram` (CRUD, save, deploy, and the building-block document), `resources`, `logs`, `runs` (Ansible jobs), `codeUpload`; and B's `catalog` (manifests, the manifest-to-form mapping, linked accounts), `connections`, `deployments` (deployments, compatibility, preview, B's runs and logs), `examples` (the built-in example diagrams "New diagram" offers) and `functionCode` (code uploaded for a function block; the upload uses `XMLHttpRequest` for its progress, with B's error envelope). Follow this shape for a new feature.
- `src/lib/adapterApi.ts`: B API's client and its error envelope; B's four features call through it.
- `src/providers/`: Google sign-in; `GOOGLE_CLIENT_ID` there is the one client id the website uses. `src/app/providers.tsx` sets up TanStack Query (60-second stale time, one retry) and sends any expired session, from A or B, to `/login`.
- `src/components/common/` (the sidebar, a drawer behind a menu button below 768 pixels; `ConfirmDialog` for every delete; `FirstSteps` for the path to a first deploy) and `src/components/ui/` (shadcn components). The root layout puts `dark` on `<html>`, so the shadcn tokens take their dark values, and loads Geist and Geist Mono with `next/font` (Geist Mono is not preloaded). Google's sign-in provider wraps `/login` only, in `src/app/(auth)/login/layout.tsx`. `src/app/not-found.tsx` is the 404 page for signed-in users (the route guard sends signed-out ones to `/login`).
- `src/proxy.ts`: the route guard. It checks only that the `clutter_token` cookie exists, not that it is valid.
- `src/components/clouds/`: `CloudMark`, a cloud's logo for B's screens.
- `public/`: A's AWS service icons (`aws/`), B's cloud logos and service icons (`clouds/`), Clutter's logos and the account-link guide.

## How sign-in and data flow

1. `/login` shows Google sign-in. The Google ID token goes to `POST /log-in`, which returns Clutter's JWT.
2. The JWT is stored twice: in `localStorage` as `google_data` (the API client reads it from there) and in the `clutter_token` cookie (the route guard reads it).
3. Every API call sends the JWT raw in the `Authorization` header, without `Bearer`, to A and to B alike.
4. When the token stops working (B answers 401; A's gateway answers 403 "not authorized" or 401), both copies are cleared and the browser goes to `/login`, which says "Your session expired. Sign in again." Sign out clears both copies too.
5. Logs and Ansible jobs are polled about every 10 seconds; B's runs every 2 seconds while one is queued or running. Nothing is pushed.

## Known gaps

They are listed in `.claude/findings/NOW.md`.

## Adding UI components

The capstone team added components with the shadcn CLI, for example:

```bash
npx shadcn@latest add input
```
