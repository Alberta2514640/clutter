// The product screenshots and demo clip under public/landing, taken from the
// running app (devserver) in a demo team, Northwind, in the dark theme at
// device scale 2 so they stay sharp on high-density screens. The media
// relabel a local test connection as AWS and OpenStack to show the product as
// it ran on real AWS and VEXXHOST on 2026-11-20: in the browser only, the Mock
// connection reads as "Northwind AWS" in ca-central-1 (with AWS's NAT gateway
// cost note on the deploy confirmation), and the connection list gains a
// display-only OpenStack entry, "Northwind private cloud". The diagrams, the
// deployments and their runs are real local runs. The Terraform preview is of
// a deployment on another demo team's AWS connection, rendered locally. Width
// and height are each file's own pixel size; next/image serves smaller copies
// by the sizes each use declares. Because of that relabelling, no alt text
// says a real AWS or OpenStack deploy happened in the picture: it names the
// example connection ("a connection named Northwind AWS"), and the proof
// strip says the screenshots use example names and a local test cloud
// (shots.test.ts holds the alt text to this).
export type Shot = { src: string; width: number; height: number; alt: string };

export const SHOTS = {
  editor: {
    src: "/landing/editor-blocks.webp",
    width: 2400,
    height: 1500,
    alt: "Clutter's editor with the Two-tier app diagram in the Northwind team's Web shop project. Two VMs, web and db, sit inside a dashed network box named app-net, linked to two firewalls, web-fw and db-fw, and a volume, db-data, and every block is marked Live. The palette on the left lists the eight building blocks, each in its own colour, and a line under the canvas reads: Builds 2 servers, 1 network, 2 firewalls, 1 disk. The Deploy tab shows a deployment on an example connection named Northwind AWS, marked Live: Deploy finished, Built 6 blocks, and What was built with the website's address and an Open button.",
  },
  deploy: {
    src: "/landing/deploy-panel.webp",
    width: 880,
    height: 1280,
    alt: "The Deploy tab of the Two-tier app diagram. A deployment on an example connection named Northwind AWS is marked Live and Up to date, with Plan and Preview code beside it. Deploy finished: Built 6 blocks, with a Show log button. What was built shows the website's address with an Open button. A note says every block can be built here, and Destroy and Remove from diagram sit at the bottom.",
  },
  preview: {
    src: "/landing/terraform-preview.webp",
    width: 1486,
    height: 812,
    alt: "The Terraform preview for a deployment on an example AWS connection named Northwind AWS. Nothing is built by a preview. Tabs list the files Clutter writes, among them code.tf, defaults.tf, iam.tf, main.tf, outputs.tf and override.tf, with Copy and Download buttons. main.tf is open at the network block app-net: an availability zones lookup and an aws_vpc resource with the range 10.40.0.0/24.",
  },
  connections: {
    src: "/landing/connections.webp",
    width: 2880,
    height: 840,
    alt: "The Connections page of an example team, Northwind, with two connections marked Verified and given example names: Northwind AWS, shown as Amazon Web Services, and Northwind private cloud, shown as OpenStack. Each has Verify and Delete buttons, and a Connect a cloud button sits at the top right.",
  },
} satisfies Record<string, Shot>;

// The sign-in carousel's own crops, cut to its 16:10 frame. Its first slide
// uses SHOTS.editor, which already has that shape.
export const SIGNIN_SHOTS = {
  connect: {
    src: "/landing/signin-connect.webp",
    width: 1472,
    height: 920,
    alt: "The Connect a cloud panel. Choose a cloud: Amazon Web Services (your own account, through a role you create there) or OpenStack (VEXXHOST and other OpenStack clouds).",
  },
  deploy: {
    src: "/landing/signin-deploy.webp",
    width: 1820,
    height: 1137,
    alt: "A Two-tier app deployment on an example connection named Northwind AWS, marked Live. Its web and db VMs show Live on the canvas, and What was built shows the website's address with an Open button.",
  },
} satisfies Record<string, Shot>;

// The three "View as" pictures of one diagram. They share one size, so
// switching between them never moves the page.
export type ViewShot = { id: string; label: string; caption: string; shot: Shot };

const VIEW_SIZE = { width: 2320, height: 1688 };

export const VIEW_SHOTS: ViewShot[] = [
  {
    id: "blocks",
    label: "Blocks",
    caption: "Plain building blocks. A VM inside a dashed network box runs in that network.",
    shot: {
      src: "/landing/view-blocks.webp",
      ...VIEW_SIZE,
      alt: "A diagram made from the Full app example, as plain building blocks: two VMs, web and db, inside the dashed app-net network box, with two firewalls, a volume, a bucket, an HTTP API, a function and a table, each kind in its own colour. The View as switch offers Blocks, AWS and OpenStack. A line under the canvas reads: Has 10 blocks.",
    },
  },
  {
    id: "aws",
    label: "AWS",
    caption: "Viewed as AWS: every block has an AWS service, and AWS builds all ten.",
    shot: {
      src: "/landing/view-aws.webp",
      ...VIEW_SIZE,
      alt: "The same diagram viewed as AWS: the VMs show as EC2, the volume as EBS, the firewalls as security groups, the bucket as S3, the HTTP API as API Gateway, the function as Lambda and the table as DynamoDB. A note reads: Viewing as Amazon Web Services (no deployment yet): AWS builds every block.",
    },
  },
  {
    id: "openstack",
    label: "OpenStack",
    caption: "Viewed as OpenStack: the four blocks OpenStack does not build are marked before you deploy, and a deployment can run them on AWS instead.",
    shot: {
      src: "/landing/view-openstack.webp",
      ...VIEW_SIZE,
      alt: "The same diagram viewed as OpenStack: the VMs show as Server (Nova), the volume as Volume (Cinder) and the firewalls as security groups. The bucket, the HTTP API, the function and the table are dimmed and marked Not on OpenStack, and a note says 4 blocks are not on OpenStack.",
    },
  },
];

// The demo clip: adding a VM named app to the Storefront diagram, dragging it
// into the network box, reading a link's card, saving, and deploying to
// Northwind AWS (the relabelled test connection above) through the
// confirmation until What was built appears, with the run log closed. Muted,
// 1280 by 720. The local test cloud deploys in seconds, faster than a real
// cloud, and the caption beside it (DrawOnce.tsx) says so.
export const DEMO_CLIP = {
  webm: "/landing/demo.webm",
  mp4: "/landing/demo.mp4",
  poster: "/landing/demo-poster.webp",
  width: 1280,
  height: 720,
};
