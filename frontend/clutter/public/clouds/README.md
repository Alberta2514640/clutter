# Cloud logos and service icons

B's screens show a cloud's logo and a block's service icon from keys in each connector's manifest (`Display` in `backend/adapter/connectors/manifest.go`). A key resolves to `<connector id>/<key>.svg` in this folder (`src/lib/features/catalog/display.ts`). The connector contract tests fail if a manifest names a key with no file here. An empty icon key means the block's neutral icon (`KindIcon.tsx`).

A cloud's logo is drawn by one component, `CloudLogo` (`src/components/common/CloudLogo.tsx`), on every screen that names a cloud. Its registry lists each cloud's official logo and its neutral tile. One flag in that file, `SHOW_OFFICIAL_CLOUD_LOGOS`, switches every place to the neutral tiles at once.

A's own icons in `public/aws/` (PNG) are A's and stay as they are. B's AWS icons are the SVG versions of the same official icons, from the current package.

## Files

| File | What it is | Source | Terms |
| --- | --- | --- | --- |
| `aws/ec2.svg`, `aws/vpc.svg`, `aws/ebs.svg`, `aws/s3.svg`, `aws/lambda.svg`, `aws/dynamodb.svg`, `aws/api-gateway.svg` | AWS Architecture Icons (service icons, 64 px, unmodified) for EC2, VPC, EBS, S3, Lambda, DynamoDB and API Gateway | Asset package `Icon-package_07312026` from <https://aws.amazon.com/architecture/icons/>, folder `Architecture-Service-Icons_07312026` | AWS: "We allow customers and partners to use these toolkits and assets to create architecture diagrams." Clutter's editor is a tool for drawing architecture diagrams. The files are byte-for-byte the package's. |
| `aws/logo.svg` | The AWS logo, white on AWS's dark square (the "AWS Cloud" group icon, 32 px, unmodified) | The same asset package, `Icon-package_07312026.5846e92413caa21490223536cc97f1269e44fa92.zip` from <https://aws.amazon.com/architecture/icons/>, file `Architecture-Group-Icons_07312026/AWS-Cloud-logo_32.svg`; SHA-256 starts `63d73df1aac59dab` | The toolkit terms above. AWS's Trademark Guidelines (<https://aws.amazon.com/trademark-guidelines/>, read 2026-11-13) license the AWS marks only within their programs and say fair use "should be in plain text only (no logos)". The logo is shown by the team's decision (ADR-022, revised). |
| `aws/neutral.svg` | A neutral tile with the text "AWS", drawn for Clutter | Clutter | Clutter's own drawing. |
| `general-openstack/logo.svg` | The OpenStack logo mark ("One Color Logo Mark Red", unmodified) | <https://www.openinfra.dev/brand/logos>, OpenStack section, SVG file <https://object-storage-ca-ymq-1.vexxhost.net/swift/v1/6e4619c416ff4bd19e1c087f27a43eea/www-images-prod/openstack-logo/2016R/OpenStack-Logo-Mark.svg>; SHA-256 starts `6cb72c01d77f4df5` | The page says each mark is "subject to the terms and conditions found in the OpenInfra Trademark Policy". <https://www.openstack.org/brand/> says "We generally do not allow the use of the OpenStack logo" and that a commercial logo program needs a signed license (<https://openinfra.org/legal/trademark-policy>). The logo is shown by the team's decision (ADR-022, revised). |
| `general-openstack/neutral.svg` | A neutral tile with the text "OS", drawn for Clutter | Clutter | Clutter's own drawing. |
| `mock/logo.svg` | A neutral dashed cloud, drawn for Clutter | Clutter | Clutter's own drawing. |

The service icons and the neutral tiles were read on 2026-11-13. The two logos were downloaded and their pages read on 2026-11-20.

## What is not here, and why

- **An AWS security group icon.** AWS's package has none (diagrams draw a security group as a plain box), so AWS's firewall block keeps its neutral shield icon with the name "Security group".
- **OpenStack service marks (Nova, Neutron, Cinder).** The project mascots are licensed CC BY-ND (the OpenStack mailing list, February 2017, and the `openstack/i18n` mascot licence file), but they are the Foundation's marks, the trademark policy bars branding a commercial product with its marks without a license, and the mascots (an animal or a star per project) do not tell a user "server", "network" or "volume". OpenStack blocks keep their neutral icons, with the service names "Server (Nova)", "Network (Neutron)", "Security group" and "Volume (Cinder)" as plain text.

To show the neutral tiles everywhere instead of the two logos, set `SHOW_OFFICIAL_CLOUD_LOGOS` to `false` in `CloudLogo.tsx`; nothing else changes.

## Adding a cloud or an icon

1. Read the owner's current terms, and record the source, the terms and the date read in the table above.
2. Add `<connector id>/<key>.svg`. Keys are lowercase letters, digits and hyphens (`connectors.IconKeyPattern`).
3. Name the key in the connector's manifest: `Display.Logo` for the cloud, `BlockSpec.WithDisplay(name, key)` for a block.
4. For a cloud's logo, add the connector to `CLOUD_LOGOS` in `src/components/common/CloudLogo.tsx`, with its official file (if one may be used) and a neutral tile.
5. Run `go -C backend/adapter test ./connectors/...`; the contract tests check that the file exists.
