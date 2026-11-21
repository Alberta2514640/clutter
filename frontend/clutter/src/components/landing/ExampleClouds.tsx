"use client";

import CloudLogo, { CLOUD_LOGOS } from "@/components/common/CloudLogo";

// A client component because the cloud names come from the logo registry,
// which is a client module.
const cloudName = (connector: string) => CLOUD_LOGOS[connector]?.shortName ?? connector;

// Where an example runs: on each listed cloud alone, or as one deployment over all of them.
export function ExampleClouds({ clouds, split }: { clouds: string[]; split: boolean }) {
  if (split) {
    return (
      <p className="text-muted-foreground flex flex-wrap items-center gap-2 text-sm">
        <span className="flex items-center gap-1">
          {clouds.map((c) => (
            <CloudLogo key={c} connector={c} size={18} />
          ))}
        </span>
        {clouds.map(cloudName).join(" and ")}, in one deployment
      </p>
    );
  }
  return (
    <ul aria-label="Builds on" className="flex flex-wrap gap-1.5">
      {clouds.map((c) => (
        <li key={c} className="border-border text-muted-foreground inline-flex items-center gap-1.5 rounded-full border py-0.5 pr-2.5 pl-1 text-xs font-medium">
          <CloudLogo connector={c} size={16} />
          {cloudName(c)}
        </li>
      ))}
    </ul>
  );
}
