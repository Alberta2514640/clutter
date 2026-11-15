// Mirrors backend/adapter/connectors/manifest.go and the catalog route
// (backend/adapter/internal/api). Go writes a nil slice as null, so lists that are
// not omitempty may be null.

// A building block kind. The standard kinds are below; a connector may add its own.
export type BlockKind = string;

export const STANDARD_BLOCKS: readonly BlockKind[] = ["vm", "network", "firewall", "volume", "bucket", "function", "table", "http_api"];

export type Treatment = "as-is" | "extended" | "removed" | "new";

// "text" is multi-line text, such as a VM's startup script.
export type FieldType = "string" | "int" | "bool" | "enum" | "text";

// connectors.MaxStringField
export const MAX_STRING_FIELD = 256;

// connectors.MaxTextField, in bytes (UTF-8).
export const MAX_TEXT_FIELD = 16384;

export type FieldSpec = {
  name: string;
  title: string;
  type: FieldType;
  required: boolean;
  default?: unknown;
  allowed?: string[];
};

// connectors.BlockDisplay: the cloud's own name for the block and the key of
// its icon. No icon key means the block's neutral icon.
export type BlockDisplay = {
  name: string;
  icon?: string;
};

export type BlockSpec = {
  kind: BlockKind;
  treatment: Treatment;
  title: string;
  fields?: FieldSpec[];
  // Set on every block the cloud builds, never on a removed one.
  display?: BlockDisplay;
};

// connectors.Display: the cloud's short name and the key of its logo. Keys
// resolve to files under public/clouds/<connector>/ (catalog/display.ts).
export type ConnectorDisplay = {
  short_name: string;
  logo?: string;
};

export type EdgeSpec = {
  source: BlockKind;
  target: BlockKind;
  meaning: string;
};

export type Manifest = {
  connector: string;
  title: string;
  version: string;
  regions?: string[];
  blocks: BlockSpec[] | null;
  edges: EdgeSpec[] | null;
  settings: FieldSpec[] | null;
  display: ConnectorDisplay;
};

export type CredentialField = {
  name: string;
  label: string;
  help?: string;
  secret: boolean;
  required: boolean;
};

export type CredentialSpec = {
  kind: "borrowed" | "stored";
  fields: CredentialField[] | null;
  // True when a connection comes only from a linked account (AWS): B refuses
  // pasted credentials, so the UI never asks for them.
  linked_only: boolean;
};

export type CatalogEntry = {
  manifest: Manifest;
  credentials: CredentialSpec;
  // True when the connector offers accounts already linked outside B, which
  // the linked-accounts route lists.
  has_linked_accounts: boolean;
};

export type CatalogResponse = { items: CatalogEntry[] | null };

// connectors.LinkedAccount: an account linked outside B, by name only.
export type LinkedAccount = {
  id: string;
  name: string;
  region?: string;
};
