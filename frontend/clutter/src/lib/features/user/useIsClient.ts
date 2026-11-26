import { useSyncExternalStore } from "react";

const subscribe = () => () => {};

// useIsClient is false while the server renders and during hydration, then
// true. A page reads values kept in this browser (the preferences store) only
// once it is true, so the first client render matches the server's HTML.
export function useIsClient(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => true,
    () => false,
  );
}
