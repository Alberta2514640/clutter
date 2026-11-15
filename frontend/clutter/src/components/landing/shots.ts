// The product screenshots and demo clip under public/landing, taken from the
// running app (devserver, AWS applied on LocalStack) with the sample "Web shop"
// diagram, at device scale 2 so they stay sharp on high-density screens.
// Width and height are each file's own pixel size; next/image serves smaller
// copies by the sizes each use declares.
export type Shot = { src: string; width: number; height: number; alt: string };

export const SHOTS = {
  editor: {
    src: "/landing/editor-blocks.webp",
    width: 2400,
    height: 1500,
    alt: "Clutter's editor with the Web shop diagram: a firewall, a VM named web, a volume, a network, a bucket and a function, joined by connections. The palette names the clouds that build each block. The Deploy tab shows the AWS deployment as Deployed, with its run log ending in Apply complete, 24 resources added, and a second deployment on a private OpenStack cloud.",
  },
  deploy: {
    src: "/landing/deploy-panel.webp",
    width: 1200,
    height: 1876,
    alt: "The Deploy tab with two deployments of the diagram. The AWS one is Deployed, says every block can be built there, and its run log ends with Apply complete, 24 resources added. The other runs on a private OpenStack cloud with some blocks on AWS, and both of its clouds say every block can be built there.",
  },
  connections: {
    src: "/landing/connections.webp",
    width: 2400,
    height: 1100,
    alt: "The Connections page of the Northwind Labs organization with two verified connections: Northwind AWS in ca-central-1, and Northwind private cloud on OpenStack in RegionOne. Below them are the clouds you can add.",
  },
} satisfies Record<string, Shot>;

// The three "View as" pictures of one diagram. They share one size, so
// switching between them never moves the page.
export type ViewShot = { id: string; label: string; caption: string; shot: Shot };

const VIEW_SIZE = { width: 2120, height: 1560 };

export const VIEW_SHOTS: ViewShot[] = [
  {
    id: "blocks",
    label: "Blocks",
    caption: "Plain building blocks. Each block is tagged with the clouds it deploys to.",
    shot: {
      src: "/landing/view-blocks.webp",
      ...VIEW_SIZE,
      alt: "The Web shop diagram as plain building blocks: firewall, VM, volume, network, bucket and function. Each block is tagged with the clouds it deploys to, and the palette crosses out OpenStack for the blocks it does not build.",
    },
  },
  {
    id: "aws",
    label: "AWS",
    caption: "Viewed as AWS: every block has an AWS service, and AWS builds all six.",
    shot: {
      src: "/landing/view-aws.webp",
      ...VIEW_SIZE,
      alt: "The same diagram viewed as AWS: the VM shows as EC2, the network as VPC, the volume as EBS, the firewall as a security group, the bucket as S3 and the function as Lambda. A note says AWS builds every block.",
    },
  },
  {
    id: "openstack",
    label: "OpenStack",
    caption: "Viewed as OpenStack: the bucket and the function are marked, so you know before you deploy.",
    shot: {
      src: "/landing/view-openstack.webp",
      ...VIEW_SIZE,
      alt: "The same diagram viewed as OpenStack: the VM shows as Server (Nova), the network as Network (Neutron), the volume as Volume (Cinder) and the firewall as a security group. The bucket and the function are dimmed and marked Not on OpenStack.",
    },
  },
];

// The demo clip: drawing two blocks, connecting them, saving and pressing
// Deploy. Muted, about 18 seconds, 1280 by 720.
export const DEMO_CLIP = {
  webm: "/landing/demo.webm",
  mp4: "/landing/demo.mp4",
  poster: "/landing/demo-poster.webp",
  width: 1280,
  height: 720,
};
