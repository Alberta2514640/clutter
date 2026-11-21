// Small wording helpers B's screens share. Files that Vitest runs import this
// by relative path: Vitest has no "@/" alias.

// plural counts a noun: "1 block", "3 blocks".
export const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

// joinAnd lists names in a sentence: "a", "a and b", "a, b and c".
export const joinAnd = (names: string[]) =>
  names.length <= 1 ? (names[0] ?? "") : `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}`;
