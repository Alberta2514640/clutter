// How a deployment setting is drawn when its connection's cloud can list the
// values (GET .../connections/{id}/options): a list of the cloud's own names,
// or the manifest's text box when there is no list, with the reason. A typed
// value is always still allowed: a list can fail, be cut short, or miss a
// name the user knows. B checks every value again on save and before a run.

import type { FieldSpec } from "../catalog/types";
import type { Choice, ConnectionOptions } from "./types";

// What the options query holds, in the terms the form needs.
export type OptionsState =
  | { status: "loading" }
  // unsupported: the cloud lists nothing (B's 501), which is not an error.
  | { status: "error"; message: string; unsupported: boolean }
  | { status: "ready"; options: ConnectionOptions };

export type SettingView =
  // The manifest's own input. note says why there is no list; canChoose
  // offers a way back to the list after the user chose to type.
  | { control: "field"; note: string | null; loading: boolean; canChoose: boolean }
  // The cloud's list. current is the value the select shows; the first
  // choice is the current value when the cloud does not list it.
  | { control: "list"; choices: Choice[]; current: string };

// The list entry that switches a setting to a text box.
export const TYPE_OWN = "\u0000type-own";

const field = (note: string | null, extra: Partial<{ loading: boolean; canChoose: boolean }> = {}): SettingView => ({
  control: "field",
  note,
  loading: extra.loading ?? false,
  canChoose: extra.canChoose ?? false,
});

export function settingView(f: FieldSpec, state: OptionsState, value: unknown, typing: boolean): SettingView {
  // Only free-text settings name things in the cloud; an enum already has its list.
  if (f.type !== "string") return field(null);
  if (state.status === "loading") return field("Loading choices from your cloud…", { loading: true });
  if (state.status === "error") {
    return state.unsupported ? field(null) : field(`Could not list choices from your cloud: ${state.message} Type the name instead.`);
  }
  const choices = state.options.settings?.[f.name];
  if (!choices) return field("Your cloud gave no list for this setting. Type the name.");
  if (choices.length === 0) return field("Your cloud lists none for this setting. Type the name.");
  if (typing) return field(null, { canChoose: true });

  const current = typeof value === "string" && value !== "" ? value : typeof f.default === "string" ? f.default : "";
  const listed = current === "" || choices.some((c) => c.value === current);
  return {
    control: "list",
    current,
    choices: listed ? choices : [{ value: current, label: `${current} (typed, not in your cloud's list)` }, ...choices],
  };
}

// optionsState maps the options query onto OptionsState. A query that has
// not run (no token yet) has no lists and says nothing.
export function optionsState(q: {
  isLoading: boolean;
  isError: boolean;
  data?: ConnectionOptions;
  errorCode?: string;
  errorMessage: string;
}): OptionsState {
  if (q.isLoading) return { status: "loading" };
  if (q.isError) return { status: "error", message: q.errorMessage, unsupported: q.errorCode === "no_options" };
  if (!q.data) return { status: "error", message: "", unsupported: true };
  return { status: "ready", options: q.data };
}
