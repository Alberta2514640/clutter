// Mirrors codeView in backend/adapter/internal/api/code.go: the zip uploaded for
// one function block of a building-block diagram.
export type FunctionCode = {
  node_id: string;
  file_name: string;
  // Bytes.
  size: number;
  // SHA-256 of the zip, in lowercase hex.
  sha256: string;
  uploaded_at: string;
};

export type FunctionCodeList = { items: FunctionCode[] | null };

// codestore.MaxSize: the largest zip B takes.
export const MAX_CODE_BYTES = 50 * 1024 * 1024;
