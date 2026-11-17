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
    alt: "Clutter's editor with the Web shop diagram. Two VMs, web and db, sit inside a dashed network box named app-net, linked to two firewalls, a volume and a bucket, with an HTTP API, a function and a table below. The pointer rests on the link from web to db, and its card says the VM gets the other VM's address when it first starts. A line under the canvas reads: Builds 2 servers, 1 network, 2 firewalls, 1 disk, 1 bucket, 1 function, 1 table, 1 API. The Deploy tab shows the Northwind AWS deployment as Deployed, with Built 10 blocks (58 cloud resources) and What was built: the API's address with Copy and Open buttons. Below it is a second deployment on Northwind private cloud with some blocks on Northwind AWS.",
  },
  deploy: {
    src: "/landing/deploy-panel.webp",
    width: 1200,
    height: 1876,
    alt: "The Deploy tab with two deployments of the diagram. Northwind AWS is Deployed. Its Deploy confirmation says it builds 2 servers, 1 network, 2 firewalls, 1 disk, 1 bucket, 1 function, 1 table and 1 API on Northwind AWS, and that internet access for private servers adds a NAT gateway on AWS at about $40 a month plus data. Below it, the last deploy built 10 blocks (58 cloud resources), and What was built shows the API's address with Copy and Open buttons. The second deployment runs on Northwind private cloud with some blocks on Northwind AWS, and every block can be built across the two.",
  },
  connections: {
    src: "/landing/connections.webp",
    width: 2400,
    height: 1100,
    alt: "The Connections page of the Northwind Labs organization with two verified connections: Northwind AWS on Amazon Web Services in ca-central-1, and Northwind private cloud on OpenStack in RegionOne. Below them are the clouds you can add.",
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
    caption: "Plain building blocks. A VM inside a dashed network box runs in that network.",
    shot: {
      src: "/landing/view-blocks.webp",
      ...VIEW_SIZE,
      alt: "The Web shop diagram as plain building blocks: two VMs, web and db, inside the dashed app-net network box, with two firewalls, a volume, a bucket, an HTTP API, a function and a table. Each block is tagged with the clouds it deploys to. A line under the canvas reads: Builds 2 servers, 1 network, 2 firewalls, 1 disk, 1 bucket, 1 function, 1 table, 1 API.",
    },
  },
  {
    id: "aws",
    label: "AWS",
    caption: "Viewed as AWS: every block has an AWS service, and AWS builds all ten.",
    shot: {
      src: "/landing/view-aws.webp",
      ...VIEW_SIZE,
      alt: "The same diagram viewed as AWS: the VMs show as EC2, the volume as EBS, the firewalls as security groups, the bucket as S3, the HTTP API as API Gateway, the function as Lambda and the table as DynamoDB. A note says AWS builds every block.",
    },
  },
  {
    id: "openstack",
    label: "OpenStack",
    caption: "Viewed as OpenStack: the four blocks OpenStack does not build are marked to run on another cloud, so you know before you deploy.",
    shot: {
      src: "/landing/view-openstack.webp",
      ...VIEW_SIZE,
      alt: "The same diagram viewed as OpenStack: the VMs show as Server (Nova), the volume as Volume (Cinder) and the firewalls as security groups. The bucket, the HTTP API, the function and the table are dimmed and marked On another cloud, and a note says 4 blocks run on another cloud.",
    },
  },
];

// The demo clip: dragging a VM into the Web shop diagram's network box,
// hovering a link, saving, and deploying to AWS through the confirmation
// until What was built appears. The wait while Terraform runs is cut. Muted,
// about 18 seconds, 1280 by 720.
export const DEMO_CLIP = {
  webm: "/landing/demo.webm",
  mp4: "/landing/demo.mp4",
  poster: "/landing/demo-poster.webp",
  width: 1280,
  height: 720,
};
