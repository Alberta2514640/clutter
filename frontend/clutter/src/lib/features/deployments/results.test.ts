import { describe, expect, it } from "vitest";
import { parseResults, resultRows } from "./results";

const log = [
  "Run r1: apply on deployment d1.",
  // Before B's apply line, a line shaped like a result does not count.
  "Result: web private_address 6.6.6.6",
  "aws_instance.c1_web: Creation complete after 12s [id=i-1]",
  "Apply complete! Resources: 9 added, 0 changed, 0 destroyed.",
  "Apply complete: 4 blocks built or changed.",
  "Result: api url https://abc.execute-api.ca-central-1.amazonaws.com/v1/items",
  "Result: web private_address 10.0.0.5",
  "Result: web public_address 203.0.113.9",
  "Result: bad name has spaces",
  "",
].join("\n");

describe("parseResults", () => {
  it("reads B's result lines after its apply line, by node id", () => {
    const r = parseResults(log);
    expect([...r.keys()]).toEqual(["api", "web"]);
    expect(r.get("web")).toEqual([
      { name: "private_address", value: "10.0.0.5" },
      { name: "public_address", value: "203.0.113.9" },
    ]);
  });

  it("reads each cloud's results in a split run, and nothing from a log without an apply", () => {
    const split = ["== Lab (OpenStack): apply ==", "Lab (OpenStack): apply complete, 2 blocks built or changed.", "Result: app public_address 172.24.4.10"].join("\n");
    expect(parseResults(split).get("app")).toEqual([{ name: "public_address", value: "172.24.4.10" }]);
    expect(parseResults("Plan: 3 to add, 0 to change, 0 to destroy.\nResult: web private_address 10.0.0.5").size).toBe(0);
  });
});

describe("resultRows", () => {
  const blocks = [
    { id: "web", kind: "vm", name: "web" },
    { id: "net", kind: "network", name: "app-net" },
    { id: "api", kind: "http_api", name: "items-api" },
    { id: "gone", kind: "bucket", name: "files" },
  ];
  const became = (b: { kind: string }) => ({ vm: "EC2", network: "VPC", http_api: "API Gateway", bucket: "S3" })[b.kind] ?? b.kind;

  it("lists every block with what it became and its results, public address first, and links the API's address", () => {
    const rows = resultRows(parseResults(log), blocks, became, ["gone"]);
    expect(rows).toEqual([
      {
        id: "web",
        name: "web",
        kind: "vm",
        became: "EC2",
        items: [
          { label: "Public address", value: "203.0.113.9", href: null },
          { label: "Private address", value: "10.0.0.5", href: null },
        ],
      },
      { id: "net", name: "app-net", kind: "network", became: "VPC", items: [] },
      {
        id: "api",
        name: "items-api",
        kind: "http_api",
        became: "API Gateway",
        items: [{ label: "Address", value: "https://abc.execute-api.ca-central-1.amazonaws.com/v1/items", href: "https://abc.execute-api.ca-central-1.amazonaws.com/v1/items" }],
      },
    ]);
  });

  it("never links an address that is not a web address", () => {
    const rows = resultRows(new Map([["api", [{ name: "url", value: "javascript:alert(1)" }]]]), blocks.slice(2, 3), became);
    expect(rows[0].items[0].href).toBeNull();
  });
});
