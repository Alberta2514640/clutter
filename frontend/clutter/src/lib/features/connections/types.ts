// Mirrors connectionView in backend/adapter/internal/api/handlers.go. A connection
// never carries its credentials or their reference.
export type ConnectionStatus = "working" | "failed";

export type Connection = {
  id: string;
  organization_id: string;
  connector: string;
  name: string;
  region: string;
  status: ConnectionStatus;
  status_detail: string | null;
  created_at: string;
  updated_at: string;
  verified_at: string | null;
  // How many diagrams deploy to it; deleting it removes those deployments.
  diagrams?: number;
};

// Mirrors connectors.Choice and connectors.Options in
// backend/adapter/connectors/connectors.go: the values a connection's cloud offers
// for each deployment setting. A setting that is missing has no list; an
// empty list means the cloud has none. suggested names, for a setting whose
// manifest default the cloud lacks, one of its choices that works there.
export type Choice = { value: string; label: string };

export type ConnectionOptions = {
  settings: Record<string, Choice[] | null> | null;
  suggested?: Record<string, string> | null;
};

// POST /v1/organizations/{org}/connections: pasted credentials, or the id of
// a linked account, never both.
export type CreateConnectionInput = {
  connector: string;
  name: string;
  region: string;
} & ({ credentials: Record<string, string>; linked_id?: never } | { linked_id: string; credentials?: never });
