"use client";

import Image from "next/image";
import { useState } from "react";
import { WindowFrame } from "./Screenshot";
import { FOCUS_RING } from "./site";
import { VIEW_SHOTS } from "./shots";

// "View as" on the home page: the same three choices as the editor's switch,
// swapping between screenshots of one diagram. The pictures share one size and
// sit in one grid cell, so a switch cross-fades without moving the page. Cloud
// names are plain text, never logos (ADR-022).
export function ViewAsDemo({ sizes }: { sizes: string }) {
  const [current, setCurrent] = useState(VIEW_SHOTS[0].id);
  const active = VIEW_SHOTS.find((v) => v.id === current) ?? VIEW_SHOTS[0];

  return (
    <figure>
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <span id="view-as-label" className="text-sm text-slate-400">
          View as
        </span>
        <div role="group" aria-labelledby="view-as-label" className="inline-flex rounded-lg border border-slate-700 bg-slate-900/80 p-1">
          {VIEW_SHOTS.map((v) => {
            const on = v.id === current;
            return (
              <button
                key={v.id}
                type="button"
                aria-pressed={on}
                onClick={() => setCurrent(v.id)}
                className={`rounded-md px-3.5 py-1.5 text-sm font-medium transition-colors motion-reduce:transition-none ${FOCUS_RING} ${
                  on ? "bg-teal-400 text-black" : "text-slate-300 hover:bg-slate-800 hover:text-white"
                }`}>
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
                className={`col-start-1 row-start-1 block h-auto w-full transition-opacity duration-300 motion-reduce:transition-none ${on ? "opacity-100" : "opacity-0"}`}
              />
            );
          })}
        </div>
      </WindowFrame>
      <figcaption aria-live="polite" className="mt-3 text-sm text-slate-400">
        {active.caption}
      </figcaption>
    </figure>
  );
}
