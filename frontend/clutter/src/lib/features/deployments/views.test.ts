import { describe, expect, it } from "vitest";
import type { CatalogEntry, Manifest } from "../catalog/types";
import type { Connection } from "../connections/types";
import type { DeploymentListItem } from "./types";
import { BLOCKS_VIEW, cloudBlockState, nodeLook, resolveView, viewClouds, viewSummary } from "./views";

const manifest = (connector: string, title: string, shortName: string, blocks: Manifest["blocks"]): Manifest => ({
  connector,
  title,
  version: "1",
  blocks,
  edges: null,
  settings: null,
  display: { short_name: shortName, logo: "logo" },
});

const aws = manifest("aws", "Amazon Web Services", "AWS", [
  { kind: "vm", treatment: "as-is", title: "VM", display: { name: "EC2", icon: "ec2" } },
  { kind: "bucket", treatment: "as-is", title: "Bucket", display: { name: "S3", icon: "s3" } },
]);
const openstack = manifest("general-openstack", "OpenStack", "OpenStack", [
  { kind: "vm", treatment: "as-is", title: "VM", display: { name: "Server (Nova)" } },
  { kind: "bucket", treatment: "removed", title: "Bucket" },
]);
const mock = manifest("mock", "Mock cloud", "Mock", [{ kind: "vm", treatment: "as-is", title: "VM", display: { name: "VM" } }]);

const catalog: CatalogEntry[] = [aws, openstack, mock].map((m) => ({
  manifest: m,
  credentials: { kind: "stored", fields: null, linked_only: false },
  has_linked_accounts: false,
}));

const deployment = (id: string, connector: string, connectionName: string, exclusions: string[] | null): DeploymentListItem => ({
  id,
  diagram_id: "d1",
  connection_id: `c-${id}`,
  settings: null,
  exclusions,
  status: "new",
  created_at: "",
  updated_at: "",
  connector,
  connection_name: connectionName,
  region: "r",
  latest_run: null,
});

const connection = (connector: string): Connection => ({
  id: `conn-${connector}`,
  organization_id: "o1",
  connector,
  name: connector,
  region: "r",
  status: "working",
  status_detail: null,
  created_at: "",
  updated_at: "",
  verified_at: null,
});

const vm = { id: "n1", kind: "vm" };
const bucket = { id: "n2", kind: "bucket" };
const titles: Record<string, string> = { vm: "VM", bucket: "Bucket" };
const title = (k: string) => titles[k] ?? k;

describe("viewClouds", () => {
  it("lists clouds with a deployment or a connection, in catalog order, and skips clouds the catalog lacks", () => {
    const views = viewClouds(catalog, [deployment("p1", "general-openstack", "lab", null)], [connection("aws"), connection("gone")]);
    expect(views.map((v) => v.connector)).toEqual(["aws", "general-openstack"]);
    expect(views[0].deployments).toEqual([]);
    expect(views[1].deployments).toEqual([{ id: "p1", connectionName: "lab", exclusions: [], only: null, away: [] }]);
  });

  it("lists a split deployment on each of its clouds, with the blocks each runs", () => {
    const split: DeploymentListItem = {
      ...deployment("s1", "general-openstack", "devstack", null),
      placements: [
        { connection_id: "c-aws", connection_name: "Local AWS", connector: "aws", region: "r", node_ids: ["n2"], settings: null },
        { connection_id: "c-empty", connection_name: "Spare", connector: "mock", region: "r", node_ids: [], settings: null },
      ],
    };
    const views = viewClouds(catalog, [split], []);
    expect(views.map((v) => [v.connector, v.deployments])).toEqual([
      ["aws", [{ id: "s1", connectionName: "Local AWS", exclusions: [], only: ["n2"], away: [] }]],
      ["general-openstack", [{ id: "s1", connectionName: "devstack", exclusions: [], only: null, away: ["n2"] }]],
    ]);
  });

  it("is empty with no deployments and no connections", () => {
    expect(viewClouds(catalog, [], [])).toEqual([]);
  });
});

describe("resolveView", () => {
  const views = viewClouds(catalog, [], [connection("aws")]);
  it("keeps a listed cloud and falls back to blocks otherwise", () => {
    expect(resolveView("aws", views)).toBe("aws");
    expect(resolveView("general-openstack", views)).toBe(BLOCKS_VIEW);
    expect(resolveView(null, views)).toBe(BLOCKS_VIEW);
    expect(resolveView(BLOCKS_VIEW, views)).toBe(BLOCKS_VIEW);
  });
});

describe("which blocks dim in which view", () => {
  const views = viewClouds(
    catalog,
    [
      deployment("a1", "aws", "prod", ["n1"]),
      deployment("a2", "aws", "test", null),
      deployment("o1", "general-openstack", "lab", ["n1"]),
      deployment("m1", "mock", "fake", []),
    ],
    [],
  );
  const [awsView, osView, mockView] = views;

  it("dims a block the cloud does not offer, whatever the exclusions", () => {
    expect(cloudBlockState(osView, bucket, "Bucket")).toMatchObject({ status: "not-offered", dimmed: true, label: "Not on OpenStack" });
    expect(cloudBlockState(mockView, bucket, "Bucket")).toMatchObject({ status: "not-offered", dimmed: true });
  });

  it("dims a block left out of every deployment on the cloud", () => {
    expect(cloudBlockState(osView, vm, "VM")).toMatchObject({ status: "left-out", dimmed: true, label: "Left out" });
    expect(cloudBlockState(osView, vm, "VM").detail).toBe("Left out of the deployment to lab (OpenStack).");
  });

  it("does not dim a block left out of only some deployments", () => {
    const s = cloudBlockState(awsView, vm, "VM");
    expect(s).toMatchObject({ status: "partly-left-out", dimmed: false, label: "Left out of 1 of 2" });
    expect(s.detail).toContain("prod");
  });

  it("does not dim a block the cloud builds, and names its service", () => {
    expect(cloudBlockState(awsView, bucket, "Bucket")).toEqual({ status: "builds", dimmed: false, label: null, detail: "AWS builds it as S3." });
    const connectedOnly = viewClouds(catalog, [], [connection("aws")])[0];
    expect(cloudBlockState(connectedOnly, vm, "VM").detail).toBe("AWS can build it as EC2.");
  });

  it("shows the cloud's service name and icon in its view", () => {
    expect(nodeLook("aws", views, bucket, "Bucket")).toEqual({
      subtitle: "S3",
      iconSrc: "/clouds/aws/s3.svg",
      dimmed: false,
      status: null,
      statusDetail: "AWS builds it as S3.",
      badges: [],
      viewCloud: { connector: "aws", title: views.find((v) => v.connector === "aws")!.title, shortName: "AWS", logoSrc: views.find((v) => v.connector === "aws")!.logoSrc },
    });
    expect(nodeLook(BLOCKS_VIEW, views, bucket, "Bucket").viewCloud).toBeNull();
    expect(nodeLook("general-openstack", views, bucket, "Bucket")).toMatchObject({ subtitle: "Bucket", iconSrc: null, dimmed: true, status: "Not on OpenStack" });
  });

  it("badges a block in the blocks view with every cloud it deploys to, grey where it is not built", () => {
    const look = nodeLook(BLOCKS_VIEW, views, bucket, "Bucket");
    expect(look).toMatchObject({ subtitle: "Bucket", iconSrc: null, dimmed: false, status: null });
    expect(look.badges.map((b) => [b.cloud.connector, b.on])).toEqual([
      ["aws", true],
      ["general-openstack", false],
      ["mock", false],
    ]);
  });

  it("badges only clouds with a deployment, not clouds with only a connection", () => {
    const some = viewClouds(catalog, [deployment("a1", "aws", "prod", null)], [connection("general-openstack")]);
    expect(nodeLook(BLOCKS_VIEW, some, vm, "VM").badges.map((b) => b.cloud.connector)).toEqual(["aws"]);
  });

  it("badges a block of a split deployment with the cloud it runs on, not the one it was placed away from", () => {
    const split: DeploymentListItem = {
      ...deployment("s1", "general-openstack", "devstack", null),
      placements: [{ connection_id: "c-aws", connection_name: "Local AWS", connector: "aws", region: "r", node_ids: ["n2"], settings: null }],
    };
    const v = viewClouds(catalog, [split], []);
    expect(nodeLook(BLOCKS_VIEW, v, bucket, "Bucket").badges.map((b) => [b.cloud.connector, b.on])).toEqual([["aws", true]]);
    expect(nodeLook(BLOCKS_VIEW, v, vm, "VM").badges.map((b) => [b.cloud.connector, b.on])).toEqual([["general-openstack", true]]);
    const awsView = v.find((x) => x.connector === "aws")!;
    expect(cloudBlockState(awsView, vm, "VM")).toMatchObject({ status: "elsewhere", dimmed: true, label: "On another cloud" });
    expect(viewSummary(awsView, [vm, bucket], title)).toBe("Viewing as Amazon Web Services: 1 block runs on another cloud.");
  });

  it("falls back to the blocks view for a cloud that is not listed", () => {
    expect(nodeLook("gone", views, vm, "VM").badges).toHaveLength(3);
  });

  it("sums up a view in one line", () => {
    expect(viewSummary(osView, [vm, bucket], title)).toBe("Viewing as OpenStack: 1 block not on OpenStack, 1 block left out.");
    expect(viewSummary(awsView, [vm, bucket], title)).toBe("Viewing as Amazon Web Services: 1 block left out of some deployments.");
    expect(viewSummary(awsView, [bucket], title)).toBe("Viewing as Amazon Web Services: AWS builds every block.");
    const connectedOnly = viewClouds(catalog, [], [connection("aws")])[0];
    expect(viewSummary(connectedOnly, [], title)).toBe("Viewing as Amazon Web Services (no deployment yet): no blocks yet.");
  });
});
