import { describe, expect, it } from "vitest";
import { leadLines, parseResults, resultRows, sampleNote, servesWeb } from "./results";

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

  it("does not list a function's sample marker as a value", () => {
    const rows = resultRows(new Map([["fn", [{ name: "code", value: "sample" }]]]), [{ id: "fn", kind: "function", name: "items" }], became);
    expect(rows[0].items).toEqual([]);
  });
});

describe("leadLines", () => {
  const blocks = [
    { id: "web", kind: "vm", name: "web" },
    { id: "db", kind: "vm", name: "db" },
    { id: "api", kind: "http_api", name: "items-api" },
    { id: "files", kind: "bucket", name: "files" },
  ];
  const results = new Map([
    ["web", [{ name: "private_address", value: "10.0.0.5" }, { name: "public_address", value: "203.0.113.9" }]],
    ["db", [{ name: "private_address", value: "10.0.0.6" }]],
    ["api", [{ name: "url", value: "https://abc.execute-api.ca-central-1.amazonaws.com/v1/items" }]],
    ["files", [{ name: "bucket", value: "c1-files" }]],
  ]);

  it("leads with a website for a public server that serves the web, and the API's full address", () => {
    expect(leadLines(results, blocks, new Set(["web"]))).toEqual([
      { kind: "website", id: "web", label: "Your website", url: "http://203.0.113.9" },
      { kind: "api", id: "api", label: "Your API", url: "https://abc.execute-api.ca-central-1.amazonaws.com/v1/items" },
    ]);
  });

  it("calls a public server that does not serve the web a server, and never leads with a private address", () => {
    expect(leadLines(results, blocks, new Set())[0]).toEqual({ kind: "server", id: "web", label: "Your server", address: "203.0.113.9" });
    expect(leadLines(results, blocks, new Set()).some((l) => l.id === "db")).toBe(false);
  });

  it("names each block when two lines share a label, and skips left-out blocks", () => {
    const two = new Map([...results, ["db", [{ name: "public_address", value: "203.0.113.10" }]]]);
    expect(leadLines(two, blocks, new Set(["web", "db"])).map((l) => l.label)).toEqual(["Your website (web)", "Your website (db)", "Your API"]);
    expect(leadLines(results, blocks, new Set(["web"]), ["web", "api"])).toEqual([]);
  });

  it("does not lead with an address that is not a web address", () => {
    expect(leadLines(new Map([["api", [{ name: "url", value: "javascript:alert(1)" }]]]), blocks, new Set())).toEqual([]);
  });
});

describe("sampleNote", () => {
  const fns = [
    { id: "a", kind: "function", name: "items" },
    { id: "b", kind: "function", name: "orders" },
  ];
  const sample = [{ name: "code", value: "sample" }];

  it("says once that the function answers with a sample, by name when it stands alone", () => {
    expect(sampleNote(new Map([["a", sample]]), fns, true)).toBe("Answers with a sample until you upload your code.");
    expect(sampleNote(new Map([["a", sample]]), fns, false)).toBe("items answers with a sample until you upload your code.");
    expect(sampleNote(new Map([["a", sample], ["b", sample]]), fns, true)).toBe("items and orders answer with a sample until you upload your code.");
  });

  it("says nothing once the code is uploaded or the function is left out", () => {
    expect(sampleNote(new Map(), fns, true)).toBeNull();
    expect(sampleNote(new Map([["a", sample]]), fns, true, ["a"])).toBeNull();
  });
});

describe("servesWeb", () => {
  const vm = (id: string, pub: boolean) => ({ id, kind: "vm", fields: { public: pub } });
  const fw = (id: string, ingress: string) => ({ id, kind: "firewall", fields: { ingress } });

  it("is a public server with a firewall that lets the web in on port 80", () => {
    const blocks = [vm("web", true), vm("db", false), vm("ssh", true), vm("range", true), fw("web-fw", "tcp/22, tcp/80"), fw("ssh-fw", "tcp/22, tcp/443"), fw("all", "tcp/1-1024")];
    const lines = [
      { source: "web-fw", target: "web" },
      { source: "web-fw", target: "db" },
      { source: "ssh-fw", target: "ssh" },
      { source: "all", target: "range" },
    ];
    expect([...servesWeb(blocks, lines)].sort()).toEqual(["range", "web"]);
  });

  it("does not count udp, a server without a firewall, or a line the other way", () => {
    expect(servesWeb([vm("web", true), fw("f", "udp/80")], [{ source: "f", target: "web" }]).size).toBe(0);
    expect(servesWeb([vm("web", true), fw("f", "tcp/80")], [{ source: "web", target: "f" }]).size).toBe(0);
    expect(servesWeb([vm("web", true)], []).size).toBe(0);
  });
});
