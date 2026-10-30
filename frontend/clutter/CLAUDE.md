# frontend/clutter

Next.js 16 (App Router), React 19, React Flow (`@xyflow/react`), TanStack Query, Zustand, Tailwind 4, shadcn-style components in `src/components/ui`. Code rules: `engineering-conventions` § Frontend.

Traps:
- **npm is the package manager.** The Dockerfile runs `npm ci` against `package-lock.json`; `pnpm-lock.yaml` is stale. Install with `npm ci`.
- **Next 16 and React 19 APIs differ from what you remember.** Request interception lives in `src/proxy.ts` (the Next 16 name for middleware). Check current docs before using a Next or React API.
- **The canvas is AWS-shaped.** `Palette.tsx`, `nodes/AwsServiceNode.tsx` and `nodes/ConfigPanel.tsx` hard-code AWS services, icons and option lists. Until the provider manifest is served by the API, a resource change also follows `add-cloud-resource`.
- **Option lists here must match the backend.** The Lambda runtimes (`nodejs24.x`) do not match code upload (`nodejs24`) today, so Node uploads fail; the region list offers 2 of the backend's 17.
- **No tests and no `test` script exist.** Vitest and Playwright are installed but unused. `lint` is the only script check.

Server data lives in `src/lib/features/<feature>/` (`api.ts`, `types.ts`, `keys.ts`, `hooks.ts`, optional `uiStore.ts`). Follow that shape for a new feature.
