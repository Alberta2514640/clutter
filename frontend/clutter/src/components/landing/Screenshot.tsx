import { cn } from "@/lib/utils";
import Image from "next/image";
import type { Shot } from "./shots";

// The window frame around every product picture on the home page: a thin
// border, rounded corners and a soft shadow. The product shots are of the
// dark app, so the frame is always a dark scope's card.
export function WindowFrame({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={cn("dark border-border bg-card overflow-hidden rounded-xl border shadow-[0_24px_64px_-24px_rgb(0_0_0/0.45)]", className)}>
      <div aria-hidden="true" className="border-border flex h-7 items-center gap-1.5 border-b px-3">
        <span className="bg-muted-foreground/40 size-2.5 rounded-full" />
        <span className="bg-muted-foreground/40 size-2.5 rounded-full" />
        <span className="bg-muted-foreground/40 size-2.5 rounded-full" />
      </div>
      {children}
    </div>
  );
}

// A product screenshot in the window frame. hero marks the one image that is
// the page's largest paint: it loads at once with high priority (the Next 16
// docs prefer this to preload).
export function Screenshot({ shot, sizes, hero = false, caption, className }: { shot: Shot; sizes: string; hero?: boolean; caption?: string; className?: string }) {
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
      {caption && <figcaption className="text-muted-foreground mt-3 text-sm">{caption}</figcaption>}
    </figure>
  );
}
