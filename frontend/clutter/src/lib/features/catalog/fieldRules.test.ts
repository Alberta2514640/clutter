import { describe, expect, it } from "vitest";
import {
  blockValueError,
  cidrError,
  CIDR_ERROR,
  HANDLER_ERROR,
  INGRESS_ERROR,
  MEMORY_ERROR,
  TABLE_KEY_ERROR,
  TIMEOUT_ERROR,
  openAdminPorts,
  openToEveryoneWarning,
  parseIngress,
  RSA_BITS_ERROR,
  SSH_KEY_ERROR,
  sshKeyError,
  VOLUME_SIZE_ERROR,
} from "./fieldRules";

// The SSH wire format: each string is its length (4 bytes) and its bytes.
const wire = (...strings: number[][]) => strings.flatMap((s) => [(s.length >>> 24) & 255, (s.length >>> 16) & 255, (s.length >>> 8) & 255, s.length & 255, ...s]);
const ascii = (s: string) => [...s].map((c) => c.charCodeAt(0));
const b64 = (bytes: number[]) => btoa(String.fromCharCode(...bytes));
// An ssh-rsa key whose modulus has exactly this many bits (a multiple of 8).
const rsaKey = (bits: number) => `ssh-rsa ${b64(wire(ascii("ssh-rsa"), [1, 0, 1], [0, 0x80, ...Array(bits / 8 - 1).fill(1)]))}`;
const ED25519 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDQzsOaDh11RuTDBNbgqSImtUq3e2dNn8x3ET2n6ONTs test@example";

describe("sshKeyError", () => {
  it("takes the keys B takes, with or without a comment", () => {
    expect(sshKeyError(ED25519)).toBeNull();
    expect(sshKeyError(`  ${ED25519.split(" ").slice(0, 2).join(" ")}\n`)).toBeNull();
    expect(sshKeyError(rsaKey(2048))).toBeNull();
    expect(sshKeyError(`${rsaKey(4096)} me@laptop`)).toBeNull();
    expect(sshKeyError("")).toBeNull();
  });

  it("refuses what B refuses, in B's words", () => {
    const ecdsa =
      "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBC/sSRyTyR5fgzKAUX9QA2J7E/LikwSgQRcxw8n7PR9A5zvylNd+wQDIdZwunbYnmFJb6f+WpxXi+WcBAyFllqY=";
    const body = ED25519.split(" ")[1];
    for (const bad of [
      ecdsa,
      "ssh-ed25519",
      `${ED25519}\n${ED25519}`,
      `ssh-rsa ${body}`, // the body holds another type
      `ssh-ed25519 ${body.slice(0, -4)}`, // cut short
      `ssh-ed25519 ${body.replace(/=+$/, "")}x`, // not padded base64
      `ssh-ed25519 ${b64(wire(ascii("ssh-ed25519"), Array(31).fill(1)))}`, // a 31-byte key
      `ssh-ed25519 ${b64([...wire(ascii("ssh-ed25519"), Array(32).fill(1)), 0])}`, // bytes after the key
      "-----BEGIN OPENSSH PRIVATE KEY-----",
    ]) {
      expect(sshKeyError(bad)).toBe(SSH_KEY_ERROR);
    }
  });

  it("names the RSA size B takes", () => {
    expect(sshKeyError(rsaKey(1024))).toBe(RSA_BITS_ERROR);
    expect(sshKeyError(rsaKey(16392))).toBe(RSA_BITS_ERROR);
    expect(sshKeyError(rsaKey(16384))).toBeNull();
  });
});

describe("parseIngress", () => {
  it("reads the entries B reads", () => {
    expect(parseIngress("tcp/22, udp/1000-2000,icmp")).toEqual([
      { protocol: "tcp", from: 22, to: 22 },
      { protocol: "udp", from: 1000, to: 2000 },
      { protocol: "icmp", from: 0, to: 0 },
    ]);
    expect(parseIngress("")).toEqual([]);
    expect(parseIngress(" , ")).toEqual([]);
  });

  it("refuses what B refuses", () => {
    for (const bad of ["22", "tcp/0", "tcp/65536", "tcp/80-22", "TCP/22", "tcp/22;", "ssh", "tcp/22-"]) {
      expect(parseIngress(bad)).toBeNull();
    }
  });
});

describe("cidrError", () => {
  it("accepts IPv4 and IPv6 ranges that start at their network address", () => {
    for (const ok of ["10.0.0.0/24", "0.0.0.0/0", "192.168.1.7/32", "2001:db8::/32", "::/0", " 10.1.0.0/16 "]) {
      expect(cidrError(ok)).toBeNull();
    }
  });

  it("refuses text that is not a range", () => {
    for (const bad of ["10.0.0.0", "10.0.0/24", "10.0.0.256/24", "10.0.0.0/33", "01.0.0.0/8", "abc", "2001:db8:::/32", "::ffff:1.2.3.4/128"]) {
      expect(cidrError(bad)).toBe(CIDR_ERROR);
    }
  });

  it("names the range to use instead of changing it", () => {
    expect(cidrError("10.0.0.5/24")).toBe("The address range must start at its network address: use 10.0.0.0/24.");
    expect(cidrError("2001:db8::1/32")).toBe("The address range must start at its network address: use 2001:db8::/32.");
  });

  // The answers of Go's netip.ParsePrefix, which B's connectors.ParseCIDR
  // uses, as Go 1.26 answers them.
  it("reads an IPv6 address with a dotted IPv4 tail as Go does", () => {
    for (const ok of ["64:ff9b::192.0.2.0/120", "::1.2.3.4/128", "1:2:3:4:5:6:1.2.3.4/128"]) {
      expect(cidrError(ok)).toBeNull();
    }
    expect(cidrError("64:ff9b::1.2.3.5/120")).toBe("The address range must start at its network address: use 64:ff9b::102:300/120.");
    for (const bad of ["1::01.2.3.4/128", "::1.2.3/128", "::1.2.3.4:5/128", "1:2:3:4:5:6:7:1.2.3.4/128", "1.2.3.4::/128", ":1.2.3.4/128"]) {
      expect(cidrError(bad)).toBe(CIDR_ERROR);
    }
  });

  it("refuses an IPv4 address written as IPv6, in either notation", () => {
    for (const bad of ["::ffff:1.2.3.4/128", "::ffff:a00:0/104", "::ffff:1.2.3.4/96"]) {
      expect(cidrError(bad)).toBe(CIDR_ERROR);
    }
  });
});

describe("blockValueError", () => {
  it("checks only the fields with a grammar", () => {
    expect(blockValueError("firewall", "ingress", "tcp/22")).toBeNull();
    expect(blockValueError("firewall", "ingress", "port 22")).toBe(INGRESS_ERROR);
    expect(blockValueError("firewall", "source_cidr", "everyone")).toBe(CIDR_ERROR);
    expect(blockValueError("network", "cidr", "10.0.0.0/8")).toBeNull();
    expect(blockValueError("vm", "size", "huge")).toBeNull();
    expect(blockValueError("firewall", "ingress", undefined)).toBeNull();
  });

  it("checks a VM's SSH key and a volume's size as B does", () => {
    expect(blockValueError("vm", "ssh_public_key", "ssh-dss AAAA")).toBe(SSH_KEY_ERROR);
    expect(blockValueError("vm", "ssh_public_key", ED25519)).toBeNull();
    expect(blockValueError("vm", "ssh_public_key", "  ")).toBeNull();
    for (const bad of [0, -3, 16385, 100000]) expect(blockValueError("volume", "size_gb", bad)).toBe(VOLUME_SIZE_ERROR);
    for (const ok of [1, 10, 16384]) expect(blockValueError("volume", "size_gb", ok)).toBeNull();
    // A value of the wrong type is fieldError's to refuse.
    expect(blockValueError("volume", "size_gb", "big")).toBeNull();
  });

  it("checks a function's memory, time limit and handler, and a table's key, as B's AWS connector does", () => {
    expect(blockValueError("function", "memory_mb", 64)).toBe(MEMORY_ERROR);
    expect(blockValueError("function", "memory_mb", 10241)).toBe(MEMORY_ERROR);
    expect(blockValueError("function", "memory_mb", 128)).toBeNull();
    expect(blockValueError("function", "timeout_s", 1000)).toBe(TIMEOUT_ERROR);
    expect(blockValueError("function", "timeout_s", 0)).toBe(TIMEOUT_ERROR);
    expect(blockValueError("function", "timeout_s", 900)).toBeNull();
    for (const bad of ["-bad handler", "-x", "index handler", 'a"b', "a${b}", "x".repeat(129)]) {
      expect(blockValueError("function", "handler", bad)).toBe(HANDLER_ERROR);
    }
    for (const ok of ["index.handler", "main", "example.Handler::handleRequest", "_x/y-z"]) expect(blockValueError("function", "handler", ok)).toBeNull();
    expect(blockValueError("table", "key", "1id")).toBe(TABLE_KEY_ERROR);
    expect(blockValueError("table", "key", "order_id")).toBeNull();
  });
});

describe("openAdminPorts", () => {
  it("names SSH and remote desktop open to every address", () => {
    expect(openAdminPorts({ ingress: "tcp/22,tcp/443", source_cidr: "0.0.0.0/0" })).toEqual(["SSH"]);
    expect(openAdminPorts({ ingress: "tcp/1-65535" }, { source_cidr: "0.0.0.0/0" })).toEqual(["SSH", "remote desktop"]);
    expect(openAdminPorts({ ingress: "tcp/3389", source_cidr: "::/0" })).toEqual(["remote desktop"]);
  });

  it("stays quiet for a narrow source, other ports or udp", () => {
    expect(openAdminPorts({ ingress: "tcp/22", source_cidr: "203.0.113.0/24" })).toEqual([]);
    expect(openAdminPorts({ ingress: "tcp/80,tcp/443", source_cidr: "0.0.0.0/0" })).toEqual([]);
    expect(openAdminPorts({ ingress: "udp/22", source_cidr: "0.0.0.0/0" })).toEqual([]);
  });

  it("says what is open in one sentence", () => {
    expect(openToEveryoneWarning([])).toBeNull();
    expect(openToEveryoneWarning(["SSH"])).toContain("anyone on the internet can try SSH");
  });
});
