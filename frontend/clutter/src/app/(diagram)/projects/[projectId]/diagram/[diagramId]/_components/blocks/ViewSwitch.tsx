"use client";

import CloudMark from "@/components/clouds/CloudMark";
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
    <fieldset className="flex min-w-0 items-center gap-2">
      <legend className="sr-only">View as</legend>
      <span aria-hidden className="shrink-0 text-xs text-slate-400">
        View as
      </span>
      <div className="flex min-w-0 items-center gap-0.5 overflow-x-auto rounded-md border border-slate-700 bg-slate-900 p-0.5">
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
            <span className="flex items-center gap-1.5 whitespace-nowrap rounded px-2 py-1 text-xs text-slate-300 hover:text-white peer-checked:bg-slate-700 peer-checked:text-white peer-focus-visible:ring-2 peer-focus-visible:ring-teal-400">
              {o.cloud ? <CloudMark cloud={o.cloud} size={16} /> : <Boxes aria-hidden className="h-4 w-4" />}
              {o.label}
            </span>
          </label>
        ))}
      </div>
    </fieldset>
  );
}
