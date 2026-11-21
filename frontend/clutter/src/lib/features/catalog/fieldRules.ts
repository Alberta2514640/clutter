// The rules for the portable fields B checks beyond their type: a firewall's
// allowed traffic and source range, a network's address range, a VM's SSH
// key, a volume's size, a function's memory, time limit and handler, and a
// table's key name. A rule only one cloud holds for a block that every cloud
// builds (AWS's /16 to /27 network range) stays with B's per-deployment
// check, since the form does not know the cloud. The manifest carries no
// range or pattern, so they
// mirror B's own checks: connectors.ParseIngress and connectors.ParseCIDR in
// backend/adapter/connectors/firewall.go, connectors.SSHPublicKey in
// backend/adapter/connectors/blocks.go, and the volume size every connector
// that builds volumes checks (maxVolumeSizeGB in
// backend/adapter/connectors/generalopenstack/model.go, the same range in
// connectors/aws/model.go). B checks again before anything runs. The
// messages are B's own.

const INGRESS_ENTRY = /^(?:(tcp|udp)\/([0-9]{1,5})(?:-([0-9]{1,5}))?|(icmp))$/;

export const INGRESS_ERROR =
  "Each allowed entry must look like tcp/22, udp/1000-2000 or icmp, separated by commas, with ports from 1 to 65535 and the lower port first.";
export const CIDR_ERROR = "The address range must look like 10.0.0.0/24 (IPv4) or 2001:db8::/32 (IPv6).";

export type IngressRule = { protocol: "tcp" | "udp" | "icmp"; from: number; to: number };

// parseIngress reads "tcp/22, tcp/8000-8100, icmp" into its rules, or null
// when an entry breaks the grammar. An empty field is no rules.
export function parseIngress(s: string): IngressRule[] | null {
  const out: IngressRule[] = [];
  for (const raw of s.split(",")) {
    const item = raw.trim();
    if (item === "") continue;
    const m = INGRESS_ENTRY.exec(item);
    if (!m) return null;
    if (m[4]) {
      out.push({ protocol: "icmp", from: 0, to: 0 });
      continue;
    }
    const from = Number(m[2]);
    const to = m[3] ? Number(m[3]) : from;
    if (from < 1 || to > 65535 || from > to) return null;
    out.push({ protocol: m[1] as "tcp" | "udp", from, to });
  }
  return out;
}

// parseIPv4 reads a dotted address into its 4 bytes, or null.
function parseIPv4(s: string): number[] | null {
  const parts = s.split(".");
  if (parts.length !== 4) return null;
  const bytes = parts.map((p) => (/^(0|[1-9][0-9]{0,2})$/.test(p) ? Number(p) : -1));
  return bytes.every((b) => b >= 0 && b <= 255) ? bytes : null;
}

// parseIPv6 reads an address into its 16 bytes, or null, as Go's
// netip.ParseAddr does: a dotted IPv4 tail ("64:ff9b::192.0.2.0") stands for
// the last two groups. A zone ("%eth0") is refused, as ParseCIDR refuses it.
function parseIPv6(s: string): number[] | null {
  if (s.includes("%")) return null;
  let text = s;
  let tail4: number[] | null = null;
  if (s.includes(".")) {
    const lastColon = s.lastIndexOf(":");
    tail4 = lastColon < 0 ? null : parseIPv4(s.slice(lastColon + 1));
    if (!tail4) return null;
    text = `${s.slice(0, lastColon + 1)}0:0`;
  }
  const bytes = parseHexGroups(text);
  return bytes && tail4 ? [...bytes.slice(0, 12), ...tail4] : bytes;
}

// isFourInSix is an IPv4 address written as IPv6 (::ffff:a.b.c.d), which Go's
// ParseCIDR refuses for a range in any notation.
const isFourInSix = (bytes: number[]) => bytes.length === 16 && bytes.slice(0, 10).every((b) => b === 0) && bytes[10] === 0xff && bytes[11] === 0xff;

// parseHexGroups reads eight colon-separated hex groups, with at most one "::".
function parseHexGroups(s: string): number[] | null {
  const halves = s.split("::");
  if (halves.length > 2) return null;
  const group = (g: string) => (/^[0-9a-fA-F]{1,4}$/.test(g) ? parseInt(g, 16) : -1);
  const head = halves[0] === "" ? [] : halves[0].split(":").map(group);
  const tail = halves.length === 2 ? (halves[1] === "" ? [] : halves[1].split(":").map(group)) : [];
  if ([...head, ...tail].some((g) => g < 0)) return null;
  const missing = 8 - head.length - tail.length;
  if (halves.length === 1 ? missing !== 0 : missing < 1) return null;
  const groups = [...head, ...Array(halves.length === 2 ? missing : 0).fill(0), ...tail];
  return groups.flatMap((g) => [g >> 8, g & 0xff]);
}

// cidrError returns why s is not an address range B accepts, or null. A range
// that does not start at its network address is refused with the one it
// should be, never changed.
export function cidrError(s: string): string | null {
  const text = s.trim();
  const slash = text.indexOf("/");
  if (slash < 0) return CIDR_ERROR;
  const addr = text.slice(0, slash);
  const bitsText = text.slice(slash + 1);
  const v4 = parseIPv4(addr);
  const bytes = v4 ?? parseIPv6(addr);
  if (!bytes || (!v4 && isFourInSix(bytes)) || !/^(0|[1-9][0-9]{0,2})$/.test(bitsText)) return CIDR_ERROR;
  const bits = Number(bitsText);
  if (bits > bytes.length * 8) return CIDR_ERROR;
  const masked = bytes.map((b, i) => {
    const keep = Math.max(0, Math.min(8, bits - i * 8));
    return b & ((0xff << (8 - keep)) & 0xff);
  });
  if (masked.every((b, i) => b === bytes[i])) return null;
  return `The address range must start at its network address: use ${formatAddress(masked, !!v4)}/${bits}.`;
}

function formatAddress(bytes: number[], v4: boolean): string {
  if (v4) return bytes.join(".");
  const groups = Array.from({ length: 8 }, (_, i) => (bytes[2 * i] << 8) | bytes[2 * i + 1]);
  // The longest run of two or more zero groups becomes "::", as Go writes it.
  let best = { at: -1, len: 0 };
  for (let i = 0; i < 8; ) {
    if (groups[i] !== 0) {
      i++;
      continue;
    }
    let j = i;
    while (j < 8 && groups[j] === 0) j++;
    if (j - i > best.len && j - i >= 2) best = { at: i, len: j - i };
    i = j;
  }
  const hex = groups.map((g) => g.toString(16));
  if (best.at < 0) return hex.join(":");
  return `${hex.slice(0, best.at).join(":")}::${hex.slice(best.at + best.len).join(":")}`;
}

// connectors.SSHPublicKey: the key types every cloud Clutter builds on
// imports, and the RSA sizes it takes.
const SSH_ED25519 = "ssh-ed25519";
const SSH_RSA = "ssh-rsa";
const MIN_RSA_BITS = 2048;
const MAX_RSA_BITS = 16384;

export const SSH_KEY_ERROR =
  "The SSH public key must be one line that starts with ssh-ed25519 or ssh-rsa, such as the contents of ~/.ssh/id_ed25519.pub.";
export const RSA_BITS_ERROR = `An RSA key must have from ${MIN_RSA_BITS} to ${MAX_RSA_BITS} bits. An ssh-ed25519 key is shorter and safer.`;

// base64Bytes decodes standard, padded base64 as Go's StdEncoding does, or null.
function base64Bytes(s: string): Uint8Array | null {
  if (s.length % 4 !== 0 || !/^[A-Za-z0-9+/]*={0,2}$/.test(s)) return null;
  try {
    return Uint8Array.from(atob(s), (c) => c.charCodeAt(0));
  } catch {
    return null;
  }
}

// sshString reads one length-prefixed string of the SSH wire format.
function sshString(b: Uint8Array): { s: Uint8Array; rest: Uint8Array } | null {
  if (b.length < 4) return null;
  const n = b[0] * 2 ** 24 + b[1] * 2 ** 16 + b[2] * 2 ** 8 + b[3];
  if (n > b.length - 4) return null;
  return { s: b.subarray(4, 4 + n), rest: b.subarray(4 + n) };
}

// sshKeyError returns why a VM's SSH public key is not one B takes, or null:
// one OpenSSH line, an ssh-ed25519 key or an ssh-rsa key of 2048 to 16384
// bits, whose base64 body holds a key of its named type.
export function sshKeyError(value: string): string | null {
  const s = value.trim();
  if (s === "") return null;
  if (/[\r\n]/.test(s)) return SSH_KEY_ERROR;
  const parts = s.split(/\s+/);
  if (parts.length < 2 || (parts[0] !== SSH_ED25519 && parts[0] !== SSH_RSA)) return SSH_KEY_ERROR;
  const blob = base64Bytes(parts[1]);
  const typ = blob && sshString(blob);
  if (!typ || new TextDecoder().decode(typ.s) !== parts[0]) return SSH_KEY_ERROR;
  if (parts[0] === SSH_ED25519) {
    const key = sshString(typ.rest);
    return key && key.s.length === 32 && key.rest.length === 0 ? null : SSH_KEY_ERROR;
  }
  const e = sshString(typ.rest);
  if (!e || e.s.length === 0) return SSH_KEY_ERROR;
  const n = sshString(e.rest);
  if (!n || n.s.length === 0 || n.rest.length !== 0) return SSH_KEY_ERROR;
  // n is a positive mpint: a leading zero byte only keeps its sign.
  let start = 0;
  while (start < n.s.length && n.s[start] === 0) start++;
  const modulus = n.s.subarray(start);
  let bits = modulus.length * 8;
  if (modulus.length > 0) {
    for (let b = modulus[0]; (b & 0x80) === 0; b = (b << 1) & 0xff) bits--;
  }
  return bits < MIN_RSA_BITS || bits > MAX_RSA_BITS ? RSA_BITS_ERROR : null;
}

// The volume sizes, in GB, every connector that builds volumes takes.
const MAX_VOLUME_GB = 16384;
export const VOLUME_SIZE_ERROR = `Size in GB must be from 1 to ${MAX_VOLUME_GB}.`;

// The function and table rules of the AWS connector's checkFields
// (backend/adapter/connectors/aws/model.go), the one cloud that builds them.
export const MEMORY_ERROR = "The memory must be from 128 to 10240 MB.";
export const TIMEOUT_ERROR = "The time limit must be from 1 to 900 seconds.";
export const HANDLER_ERROR =
  "The handler must start with a letter, digit or underscore and use only letters, digits and _ . : / - (at most 128), such as index.handler.";
export const TABLE_KEY_ERROR = "The key name must start with a letter and use only letters, digits and underscores (at most 64).";
const HANDLER = /^[A-Za-z0-9_][A-Za-z0-9_.:/-]{0,127}$/;
const TABLE_KEY = /^[A-Za-z][A-Za-z0-9_]{0,63}$/;

type Rule = (value: unknown) => string | null;
const textRule =
  (rule: (value: string) => string | null): Rule =>
  (v) =>
    typeof v === "string" ? rule(v) : null;
const patternRule = (pattern: RegExp, message: string) => textRule((v) => (pattern.test(v) ? null : message));
const rangeRule =
  (min: number, max: number, message: string): Rule =>
  (v) =>
    typeof v === "number" && (v < min || v > max) ? message : null;

// The portable fields B checks beyond their type, by block kind and field
// name. A value of the wrong type is fieldError's to refuse.
const RULES: Partial<Record<string, Partial<Record<string, Rule>>>> = {
  vm: { ssh_public_key: textRule(sshKeyError) },
  firewall: {
    ingress: textRule((v) => (parseIngress(v) ? null : INGRESS_ERROR)),
    source_cidr: textRule(cidrError),
  },
  network: { cidr: textRule(cidrError) },
  volume: { size_gb: rangeRule(1, MAX_VOLUME_GB, VOLUME_SIZE_ERROR) },
  function: {
    memory_mb: rangeRule(128, 10240, MEMORY_ERROR),
    timeout_s: rangeRule(1, 900, TIMEOUT_ERROR),
    handler: patternRule(HANDLER, HANDLER_ERROR),
  },
  table: { key: patternRule(TABLE_KEY, TABLE_KEY_ERROR) },
};

// blockValueError returns why a value breaks its field's rule, or null.
// Fields without a rule, and unset values, pass.
export function blockValueError(kind: string, field: string, value: unknown): string | null {
  const rule = RULES[kind]?.[field];
  if (!rule || value === undefined || value === null || (typeof value === "string" && value.trim() === "")) return null;
  return rule(value);
}

// The ports a scanner tries first: remote login to Linux (SSH) and Windows (RDP).
const ADMIN_PORTS = [
  { port: 22, name: "SSH" },
  { port: 3389, name: "remote desktop" },
];

const EVERYONE = new Set(["0.0.0.0/0", "::/0"]);

// openAdminPorts lists the remote login ports a firewall opens to every
// address on the internet ("SSH", "remote desktop"), or none. It only
// warns; whether to allow it is the person's call.
export function openAdminPorts(fields: Record<string, unknown>, defaults: Record<string, unknown> = {}): string[] {
  const source = typeof fields.source_cidr === "string" ? fields.source_cidr : typeof defaults.source_cidr === "string" ? defaults.source_cidr : "";
  if (!EVERYONE.has(source.trim())) return [];
  const rules = parseIngress(typeof fields.ingress === "string" ? fields.ingress : "") ?? [];
  return ADMIN_PORTS.filter(({ port }) => rules.some((r) => r.protocol === "tcp" && r.from <= port && port <= r.to)).map((p) => p.name);
}

// The warning a firewall with open login ports carries, or null.
export function openToEveryoneWarning(ports: string[]): string | null {
  if (ports.length === 0) return null;
  return `Open to everyone: anyone on the internet can try ${ports.join(" and ")} on the servers this firewall protects. Narrow the source addresses to your own range if you can.`;
}
