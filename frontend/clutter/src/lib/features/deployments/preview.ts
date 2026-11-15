// previewGroups sorts a deployment's preview files by the connection that
// builds them. A split deployment's files come named "<connection>/<file>"
// (backend/adapter/internal/api/split.go, splitPreview); a deployment on one
// cloud names its files alone, which gives one group with no name. A
// connection's name may hold a "/", a file name never does, so the split is
// at the last one.

export type PreviewFile = { path: string; name: string };
export type PreviewGroup = { connection: string | null; files: PreviewFile[] };

export function previewGroups(paths: string[]): PreviewGroup[] {
  const groups = new Map<string | null, PreviewFile[]>();
  for (const path of [...paths].sort()) {
    const cut = path.lastIndexOf("/");
    const connection = cut > 0 ? path.slice(0, cut) : null;
    const name = cut > 0 ? path.slice(cut + 1) : path;
    groups.set(connection, [...(groups.get(connection) ?? []), { path, name }]);
  }
  return [...groups].map(([connection, files]) => ({ connection, files }));
}

// joinAnd lists names in a sentence: "a", "a and b", "a, b and c".
export const joinAnd = (names: string[]) =>
  names.length <= 1 ? (names[0] ?? "") : `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
