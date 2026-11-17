// When the connection form offers the local-only "Use this machine's
// DevStack" button: in a local dev build (localDevApi), on a form that takes
// pasted OpenStack application credential fields. DevStack hands out exactly
// those, so the button names no connector.

import { fitsCloudsYaml } from "./cloudsYaml";

export function offersLocalDevstack(
  localDev: boolean,
  fieldNames: string[],
  mode: "linked" | "credentials",
  linkedOnly: boolean,
): boolean {
  return localDev && !linkedOnly && mode === "credentials" && fitsCloudsYaml(fieldNames);
}
