# Website UX patterns from comparable products

Read 2026-11-20 for the website redesign (ADR-028). Official vendor pages only, read through a summarising fetch tool, so descriptions of visuals are second-hand. The redesign's decisions are in ADR-028; this note keeps the evidence behind them.

## First run and core flow

| Pattern | Product | Source |
| --- | --- | --- |
| Design before credentials: draw and see the Terraform first, add cloud credentials only when deploying | Brainboard | <https://docs.brainboard.co/getting-started/fast-track.md> |
| Changes staged on the canvas (cards tagged New, Edited, Removed), then one Deploy | Railway | <https://docs.railway.com/environments> |
| Settings split by scope: project, service, volume | Railway | <https://docs.railway.com/overview/the-basics> |
| Named run states (Planning, Needs confirmation, Applying, Applied, Errored) with Confirm and apply or Discard | HCP Terraform | <https://developer.hashicorp.com/terraform/cloud-docs/run/states> |
| Three entry paths to a first deploy, sign-up not a gate | Vercel | <https://vercel.com/docs/getting-started-with-vercel> |
| Command-first landing page with no hero visual | Railway | <https://railway.com> |

## Linking a cloud account

| Pattern | Product | Source |
| --- | --- | --- |
| Role ARN pasted, connectivity tested, success or failure detail shown | Brainboard | <https://docs.brainboard.co/settings/integrations/cloud-providers/aws.md> |
| Role ARN saved with no verification step | Cloudcraft | <https://docs.datadoghq.com/cloudcraft/getting-started/connect-aws-account-with-cloudcraft/> |
| "Verify all permissions" button and an "update recommended" banner when permissions drift | Northflank | <https://northflank.com/docs/v1/application/cloud-integrations/connect-aws.md> |
| AWS through OIDC and a trust policy, optional separate plan and apply roles | HCP Terraform | <https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/aws-configuration> |

None of the products read links AWS through a one-click CloudFormation stack, as Clutter does. No product read shows an OpenStack application-credential link flow.

## Brainboard's features against Clutter's

| Brainboard feature | Clutter on 2026-11-20 |
| --- | --- |
| Visual designer that generates Terraform | Have: building blocks, read-only Terraform preview |
| Multi-cloud (AWS, Azure, GCP, OCI) | Partly: AWS and OpenStack |
| Deploy engine with logs | Have: plan, deploy, destroy, live log |
| One design, many environments | Partly: one diagram, one deployment per cloud |
| Templates | Partly: built-in examples |
| Cost estimation | Partly: the deploy confirmation names a costly part only |
| Git integration, modules catalog, drift detection, import, RBAC and audit, live collaboration, self-hosted runner, AI | No |

Sources: <https://www.brainboard.co> and its feature, use-case, pricing and integrations pages, read 2026-11-20. Clutter strengths not shown on Brainboard's pages: one deployment split across clouds, deploys into the user's own account through a narrow checked role with no stored AWS keys, OpenStack as a full target, and the "View as" compatibility check (evidence: `docs/reports/2026-11-20-real-cloud-verification.pdf`).

## Accessibility

- Anything done by dragging must also work with a single pointer without dragging (WCAG 2.2 SC 2.5.7): <https://www.w3.org/WAI/WCAG22/Understanding/dragging-movements.html>.
- Targets at least 24 by 24 CSS pixels (WCAG 2.2 SC 2.5.8): <https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html>.
- React Flow supports keyboard selection and movement of nodes and edges, and a live region: <https://reactflow.dev/learn/advanced-use/accessibility>.

## Not found

Empty states, onboarding checklists and phone-width behaviour of any canvas tool; Pulumi Cloud's UI steps (guide page returned 404); Fly.io was not read.
