"use client";

import CloudLogo from "@/components/common/CloudLogo";
import { cn } from "@/lib/utils";
import { Blocks } from "lucide-react";
import Image from "next/image";
import { useState } from "react";
import { WindowFrame } from "./Screenshot";
import { FOCUS_RING } from "./site";
import { VIEW_SHOTS } from "./shots";

// The connector each "View as" choice shows the logo of (none for plain blocks).
const VIEW_CONNECTOR: Record<string, string | undefined> = { aws: "aws", openstack: "general-openstack" };

// "View as" on the home page: the same three choices as the editor's switch,
// swapping between screenshots of one diagram. The pictures share one size and
// sit in one grid cell, so a switch cross-fades without moving the page. Each
// cloud's logo comes from the one registry (CloudLogo), beside its name.
export function ViewAsDemo({ sizes }: { sizes: string }) {
  const [current, setCurrent] = useState(VIEW_SHOTS[0].id);
  const active = VIEW_SHOTS.find((v) => v.id === current) ?? VIEW_SHOTS[0];

  return (
    <figure>
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <span id="view-as-label" className="text-muted-foreground text-sm">
          View as
        </span>
        <div role="group" aria-labelledby="view-as-label" className="border-border bg-muted inline-flex rounded-lg border p-1">
          {VIEW_SHOTS.map((v) => {
            const on = v.id === current;
            const connector = VIEW_CONNECTOR[v.id];
            return (
              <button
                key={v.id}
                type="button"
                aria-pressed={on}
                onClick={() => setCurrent(v.id)}
                className={cn(
                  "inline-flex min-h-9 items-center gap-2 rounded-md px-3.5 text-sm font-medium transition-colors",
                  FOCUS_RING,
                  on ? "bg-card text-foreground shadow-sm" : "text-muted-foreground hover:text-foreground",
                )}>
                {connector ? <CloudLogo connector={connector} size={16} /> : <Blocks className="size-4" aria-hidden="true" />}
                {v.label}
              </button>
            );
          })}
        </div>
      </div>
      <WindowFrame>
        <div className="grid">
          {VIEW_SHOTS.map((v) => {
            const on = v.id === current;
            return (
              <Image
                key={v.id}
                src={v.shot.src}
                alt={on ? v.shot.alt : ""}
                aria-hidden={on ? undefined : true}
                width={v.shot.width}
                height={v.shot.height}
                sizes={sizes}
                className={`motion-slow col-start-1 row-start-1 block h-auto w-full transition-opacity ${on ? "opacity-100" : "opacity-0"}`}
              />
            );
          })}
        </div>
      </WindowFrame>
      <figcaption aria-live="polite" className="text-muted-foreground mt-3 text-sm">
        {active.caption}
      </figcaption>
    </figure>
  );
}
