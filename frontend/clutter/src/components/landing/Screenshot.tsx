import Image from "next/image";
import type { Shot } from "./shots";

// The plain window frame around every product picture on the home page.
export function WindowFrame({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return (
    <div
      className={`overflow-hidden rounded-xl border border-slate-700/70 bg-slate-950 shadow-[0_24px_80px_rgba(0,0,0,0.6)] transition-[border-color,box-shadow] duration-300 hover:border-teal-400/40 hover:shadow-[0_24px_90px_rgba(45,212,191,0.12)] motion-reduce:transition-none ${className}`}>
      <div aria-hidden="true" className="flex h-7 items-center gap-1.5 border-b border-slate-800 bg-slate-900/80 px-3">
        <span className="h-2.5 w-2.5 rounded-full bg-slate-600" />
        <span className="h-2.5 w-2.5 rounded-full bg-slate-600" />
        <span className="h-2.5 w-2.5 rounded-full bg-slate-600" />
      </div>
      {children}
    </div>
  );
}

// A product screenshot in the window frame. hero marks the one image that is
// the page's largest paint: it loads at once with high priority (the Next 16
// docs prefer this to preload).
export function Screenshot({ shot, sizes, hero = false, caption, className = "" }: { shot: Shot; sizes: string; hero?: boolean; caption?: string; className?: string }) {
  return (
    <figure className={className}>
      <WindowFrame>
        <Image
          src={shot.src}
          alt={shot.alt}
          width={shot.width}
          height={shot.height}
          sizes={sizes}
          loading={hero ? "eager" : "lazy"}
          fetchPriority={hero ? "high" : undefined}
          className="block h-auto w-full"
        />
      </WindowFrame>
      {caption && <figcaption className="mt-3 text-sm text-slate-400">{caption}</figcaption>}
    </figure>
  );
}
