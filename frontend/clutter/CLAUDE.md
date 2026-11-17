# frontend/clutter

Next.js 16 (App Router), React 19, React Flow (`@xyflow/react`), TanStack Query, Zustand, Tailwind 4, shadcn-style components in `src/components/ui`. Code rules: `engineering-conventions` § Frontend.

Traps:

- **npm is the package manager.** The Dockerfile runs `npm ci` against `package-lock.json`. Install with `npm ci`.
- **Next 16 and React 19 APIs differ from what you remember.** Request interception lives in `src/proxy.ts` (the Next 16 name for middleware). Check current docs before using a Next or React API.
- **The canvas is AWS-shaped.** `Palette.tsx`, `nodes/AwsServiceNode.tsx` and `nodes/ConfigPanel.tsx` hard-code AWS services, icons and option lists. These are A's screens and stay AWS-only (ADR-006). B's editor and Connections page are new and read each connector's manifest from B's API, never these lists. An A resource change follows `add-cloud-resource`.
- **Option lists here must match the backend.** The Lambda runtime list copies `generic.LambdaRuntimes` in the backend, which refuses any other runtime. The account-link page offers no region: it opens AWS in `ca-central-1` (`DEFAULT_AWS_REGION` in `organization/api.ts`), and `runTerraform` never sends a region, so every A deploy runs in the backend's `generic.DefaultRegion` (Clutter's own region, from the Lambda's `AWS_REGION`). Ansible job statuses (`AnsibleJobStatus` in `runs/types.ts`) copy the backend's `AllowedJobStatuses`; polling stops on `COMPLETED` or `FAILED`.
- **A diagram's editor is chosen by its saved document.** `engine: "blocks"` opens B's editor, anything else A's. `normalizeDiagram` in `diagram/api.ts` must keep `engine`, or every building-block diagram opens in A's editor. A's create route ignores `data`, so `useCreateDiagram` saves the marker with a second `PUT /diagram`.
- **B's screens never read A's AWS lists.** Blocks, fields, defaults, allowed values, edges and settings come from B's catalog (`catalog/forms.ts`); credential forms from each connector's credential spec. B's errors come in one envelope, read once in `src/lib/adapterApi.ts`.
- **A's API is called through one helper, and its error message stays the raw body.** `apiFetch` in `user/session.ts` throws an `ApiError` with the status; A's editor parses `error.message` as JSON for the diagram's problems, so do not rewrite it. Pages show their own plain sentence, never `String(err)`. An expired token (B's 401, A's gateway 403 "not authorized") is caught once, in `src/app/providers.tsx`; `logs`, `runs`, `resources` and `codeUpload` still keep their own fetch copies.
- **There is no `test` script.** The Vitest files sit beside the code in `lib/features/` (for example `diagram/types.test.ts`); run them with `npx vitest run`; CI runs them too (`.github/workflows/checks.yml`). Vitest runs without the `@/` alias, so a module a test imports reaches its siblings by relative path. `lint` is the only script check. To see a change against real data, run the app locally (`frontend/clutter/README.md` § Run it).

Server data lives in `src/lib/features/<feature>/` (`api.ts`, `types.ts`, `keys.ts`, `hooks.ts`, optional `uiStore.ts`). Follow that shape for a new feature.
