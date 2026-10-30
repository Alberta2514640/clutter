# Frontend

The Clutter web app: a Next.js 16 App Router project with React 19, React Flow (`@xyflow/react`) for the canvas, TanStack Query for server data, Zustand for UI state, Tailwind 4 and shadcn/ui components. It was bootstrapped with `create-next-app`.

Setup and checks for the whole repo are in the root `README.md`. This page covers the frontend only.

## Run it

Use npm. The Dockerfile runs `npm ci`, and `pnpm-lock.yaml` is stale.

```bash
npm ci
NEXT_PUBLIC_API_ENDPOINT=http://localhost:9999 npm run dev
```

Open <http://localhost:3000>. `NEXT_PUBLIC_API_ENDPOINT` is the base URL of the Clutter API. No API is running today (the capstone deployment is gone), so only `/` and `/login` render; every other page redirects to `/login` without a `clutter_token` cookie.

## Check it

```bash
npx tsc --noEmit
npm run lint
NEXT_PUBLIC_API_ENDPOINT=http://localhost:9999 npm run build
```

There are no frontend tests yet.

## Where things are

- `src/app/`: routes, grouped as `(auth)`, `(dashboard)`, `(diagram)` and `(settings)`. The canvas is under `(diagram)/projects/[projectId]/diagram/[diagramId]/`.
- `src/lib/features/<feature>/`: the API client and hooks for each feature (organization, projects, diagram, runs, logs, codeUpload, resources, members, user).
- `src/providers/`: React Query, Google sign-in and other providers.
- `src/proxy.ts`: the route guard (it checks only that the cookie exists, not that it is valid).

Known gaps: member management is a mock, `settings/organization/create-org` and the workspaces page are stubs, and `/register` posts to a dead host. The full list is in `.claude/findings/NOW.md`.

## Adding UI components

The capstone team added components with the shadcn CLI, for example:

```bash
npx shadcn@latest add https://21st.dev/r/kokonutd/shape-landing-hero
npx shadcn@latest add input
```

## Do not run `init.bash`

`init.bash` is an early scaffold script from the start of the capstone. It overwrites `src/app/layout.tsx`, `src/app/page.tsx`, the providers and other real files with placeholders. It is kept only as history.
