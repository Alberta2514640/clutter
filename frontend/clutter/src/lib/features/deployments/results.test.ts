import { describe, expect, it } from "vitest";
import { builtBlocks, leadLines, parseResults, readableLog, resultRows, sampleNote, servesWeb, simulatedNote } from "./results";

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

describe("builtBlocks", () => {
  // The diagram as saved now: web was renamed web2 after the deploy, and a
  // bucket was added since.
  const saved = [
    { id: "net", kind: "network", name: "network" },
    { id: "web", kind: "vm", name: "web2" },
    { id: "new", kind: "bucket", name: "files" },
  ];

  it("names each block as the apply built it, in the diagram's order, even after a rename", () => {
    const applied = [
      // Before B's apply line, a line shaped like a block does not count.
      "Built: fake vm intruder",
      "Apply complete: 3 blocks built or changed.",
      "Built: fw firewall fw",
      "Built: net network network",
      "Built: web vm web",
      "Built: odd vm Has Spaces",
      "Result: web private_address 10.0.0.5",
    ].join("\n");
    expect(builtBlocks(applied, saved)).toEqual([
      { id: "net", kind: "network", name: "network" },
      { id: "web", kind: "vm", name: "web" },
      // Removed from the diagram since the deploy: still built, so still listed.
      { id: "fw", kind: "firewall", name: "fw" },
    ]);
  });

  it("gathers every cloud of a split run, and is null for a run from before the Built lines", () => {
    const split = ["data (Mock): apply complete, 1 block built or changed.", "Built: new bucket files", "main (Mock): apply complete, 1 block built or changed.", "Built: web vm web"].join("\n");
    expect(builtBlocks(split, saved)?.map((b) => b.id)).toEqual(["web", "new"]);
    expect(builtBlocks(log, saved)).toBeNull();
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

describe("simulated servers", () => {
  const blocks = [
    { id: "web", kind: "vm", name: "web" },
    { id: "db", kind: "vm", name: "db" },
    { id: "api", kind: "http_api", name: "items-api" },
  ];
  const results = new Map([
    ["web", [{ name: "public_address", value: "203.0.113.9" }, { name: "simulated", value: "emulator" }]],
    ["db", [{ name: "private_address", value: "10.0.0.6" }, { name: "simulated", value: "emulator" }]],
    ["api", [{ name: "url", value: "http://127.0.0.1:4566/restapis/a/v1/_user_request_/items" }]],
  ]);

  it("gives a simulated server no line to open, keeps the API's, and says once per cloud why", () => {
    expect(leadLines(results, blocks, new Set(["web"])).map((l) => l.kind)).toEqual(["api"]);
    expect(simulatedNote(results, blocks, () => "AWS")).toEqual(["VMs on local AWS are simulated; they do not boot."]);
    expect(simulatedNote(results, blocks, () => "AWS", ["web", "db"])).toEqual([]);
  });

  it("still lists a simulated server's addresses in the details, without the marker", () => {
    const rows = resultRows(results, blocks, () => "EC2");
    expect(rows[0].items).toEqual([{ label: "Public address", value: "203.0.113.9", href: null }]);
  });
});

describe("servesWeb", () => {
  const vm = (id: string, pub: boolean, script = "#!/bin/sh\nserve") => ({ id, kind: "vm", fields: { public: pub, startup_script: script } });
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

  it("does not count a server with no startup script: nothing starts a web server on it", () => {
    expect(servesWeb([vm("web", true, "  \n"), fw("f", "tcp/80")], [{ source: "f", target: "web" }]).size).toBe(0);
  });
});

describe("readableLog", () => {
  it("shows B's Built and Result lines in plain words, named as the diagram names the blocks", () => {
    const raw = [
      "Deploy on Mock cloud (lab), started 10:42:03 UTC.",
      "Result: n1 public_address 6.6.6.6",
      "Created server (VM).",
      "Apply complete: 2 blocks built or changed.",
      "Built: n1 vm server",
      "Built: n2 network net",
      "Result: n1 public_address 203.0.113.73",
      "Result: n1 private_address 10.0.0.5",
      "Result: fn code sample",
      "Run r1 ended at 10:42:04 UTC.",
    ].join("\n");
    const title = (k: string) => ({ vm: "VM", network: "Network" })[k] ?? k;
    expect(readableLog(raw, title).split("\n")).toEqual([
      "Deploy on Mock cloud (lab), started 10:42:03 UTC.",
      // Before B's apply line nothing is rewritten.
      "Result: n1 public_address 6.6.6.6",
      "Created server (VM).",
      "Apply complete: 2 blocks built or changed.",
      "Built server (VM)",
      "Built net (Network)",
      "server: public address 203.0.113.73",
      "server: private address 10.0.0.5",
      "fn: answers with a sample until you upload your code",
      "Run r1 ended at 10:42:04 UTC.",
    ]);
  });

  it("leaves a log with no apply as it was", () => {
    expect(readableLog("Plan: 1 to add, 0 to change, 0 to destroy.\nBuilt: n1 vm web")).toBe("Plan: 1 to add, 0 to change, 0 to destroy.\nBuilt: n1 vm web");
  });
});
