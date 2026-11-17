# Cloud logos and service icons

B's screens show a cloud's logo and a block's service icon from keys in each connector's manifest (`Display` in `backend/adapter/connectors/manifest.go`). A key resolves to `<connector id>/<key>.svg` in this folder (`src/lib/features/catalog/display.ts`). The connector contract tests fail if a manifest names a key with no file here. An empty icon key means the block's neutral icon (`KindIcon.tsx`).

A's own icons in `public/aws/` (PNG) are A's and stay as they are. B's AWS icons are the SVG versions of the same official icons, from the current package.

## Files

| File | What it is | Source | Terms |
| --- | --- | --- | --- |
| `aws/ec2.svg`, `aws/vpc.svg`, `aws/ebs.svg`, `aws/s3.svg`, `aws/lambda.svg`, `aws/dynamodb.svg`, `aws/api-gateway.svg` | AWS Architecture Icons (service icons, 64 px, unmodified) for EC2, VPC, EBS, S3, Lambda, DynamoDB and API Gateway | Asset package `Icon-package_07312026` from <https://aws.amazon.com/architecture/icons/>, folder `Architecture-Service-Icons_07312026` | AWS: "We allow customers and partners to use these toolkits and assets to create architecture diagrams." Clutter's editor is a tool for drawing architecture diagrams. The files are byte-for-byte the package's. |
| `aws/logo.svg` | A neutral tile with the text "AWS", drawn for Clutter | Clutter | The AWS logo is not used. AWS's Trademark Guidelines (updated 2026-11-05, <https://aws.amazon.com/trademark-guidelines/>) license the AWS marks only within their programs, and say fair use "should be in plain text only (no logos)". The "Powered by AWS" logo may not be used to show affiliation, so it does not fit a cloud picker. |
| `general-openstack/logo.svg` | A neutral tile with the text "OS", drawn for Clutter | Clutter | The OpenStack logo is not used. The OpenInfra Foundation "generally do[es] not allow the use of the OpenStack logo"; community use is for non-commercial activities only, and a commercial product needs a signed trademark license (<https://www.openstack.org/brand/>, <https://openinfra.org/legal/trademark-policy>). |
| `mock/logo.svg` | A neutral dashed cloud, drawn for Clutter | Clutter | Clutter's own drawing. |

All sources were read on 2026-11-13.

## What is not here, and why

- **An AWS security group icon.** AWS's package has none (diagrams draw a security group as a plain box), so AWS's firewall block keeps its neutral shield icon with the name "Security group".
- **OpenStack service marks (Nova, Neutron, Cinder).** The project mascots are licensed CC BY-ND (the OpenStack mailing list, February 2017, and the `openstack/i18n` mascot licence file), but they are the Foundation's marks, the trademark policy bars branding a commercial product with its marks without a license, and the mascots (an animal or a star per project) do not tell a user "server", "network" or "volume". OpenStack blocks keep their neutral icons, with the service names "Server (Nova)", "Network (Neutron)", "Security group" and "Volume (Cinder)" as plain text.

Using a real logo for AWS or OpenStack needs written permission from AWS, or an "OpenStack Compatible" license from the OpenInfra Foundation. Add the file here and change the manifest's logo key; nothing else changes.

## Adding a cloud or an icon

1. Read the owner's current terms, and record the source, the terms and the date read in the table above.
2. Add `<connector id>/<key>.svg`. Keys are lowercase letters, digits and hyphens (`connectors.IconKeyPattern`).
3. Name the key in the connector's manifest: `Display.Logo` for the cloud, `BlockSpec.WithDisplay(name, key)` for a block.
4. Run `go -C backend/adapter test ./connectors/...`; the contract tests check that the file exists.
