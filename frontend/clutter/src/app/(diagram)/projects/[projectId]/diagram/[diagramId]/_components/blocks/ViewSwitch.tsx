"use client";

import CloudLogo from "@/components/common/CloudLogo";
import { BLOCKS_VIEW, type CloudView } from "@/lib/features/deployments/views";
import { Boxes } from "lucide-react";

type Props = {
  views: CloudView[];
  value: string;
  onChange: (view: string) => void;
};

// "View as": plain building blocks, or the diagram as one cloud builds it.
// Radio inputs give arrow-key movement and a screen-reader group for free.
export default function ViewSwitch({ views, value, onChange }: Props) {
  if (views.length === 0) return null;
  const options = [
    { id: BLOCKS_VIEW, label: "Blocks", hint: "Show plain building blocks, with the clouds each one deploys to", cloud: null },
    ...views.map((v) => ({ id: v.connector, label: v.shortName, hint: `Show each block as ${v.title} builds it`, cloud: v })),
  ];
  return (
    <fieldset className="surface-raised flex min-w-0 items-center gap-2 rounded-lg border py-1 pr-1 pl-2.5">
      <legend className="sr-only">View as</legend>
      <span aria-hidden className="text-muted-foreground shrink-0 text-xs">
        View as
      </span>
      <div className="flex min-w-0 items-center gap-0.5 overflow-x-auto overflow-y-hidden">
        {options.map((o) => (
          <label key={o.id} title={o.hint} className="shrink-0 cursor-pointer">
            <input
              type="radio"
              name="blocks-view"
              value={o.id}
              checked={value === o.id}
              onChange={() => onChange(o.id)}
              className="peer sr-only"
            />
            <span className="text-muted-foreground hover:text-foreground peer-checked:bg-secondary peer-checked:text-foreground peer-focus-visible:ring-ring/50 flex h-7 items-center gap-1.5 rounded-md px-2 text-xs whitespace-nowrap transition-colors peer-checked:font-medium peer-focus-visible:ring-[3px]">
              {o.cloud ? <CloudLogo connector={o.cloud.connector} shortName={o.cloud.shortName} src={o.cloud.logoSrc} size={16} /> : <Boxes aria-hidden className="size-4" />}
              {o.label}
            </span>
          </label>
        ))}
      </div>
    </fieldset>
  );
}
