import { create } from "zustand";
import { mergeRuns } from "./runState";
import type { Run } from "./types";

// The runs this page has seen per deployment, for the deploy panel's run
// history. Memory only: a reload starts again from what B returns (the
// latest run and the last successful apply).
type RunHistoryState = {
  byDeployment: Record<string, Run[]>;
  record: (deploymentId: string, runs: (Run | null | undefined)[]) => void;
};

export const useRunHistoryStore = create<RunHistoryState>((set) => ({
  byDeployment: {},
  record: (deploymentId, runs) =>
    set((s) => {
      const known = s.byDeployment[deploymentId] ?? [];
      const next = mergeRuns(known, runs);
      const same = next.length === known.length && next.every((r, i) => r === known[i]);
      return same ? s : { byDeployment: { ...s.byDeployment, [deploymentId]: next } };
    }),
}));
