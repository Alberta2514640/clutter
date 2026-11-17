"use client";

import Image from "next/image";
import { useState } from "react";
import KindIcon from "./KindIcon";

// A block's icon: the cloud's service icon when the current view has one,
// else the block's neutral icon. Decorative: the block's name and service
// name are written beside it.
export default function ServiceIcon({ src, kind, className }: { src: string | null; kind: string; className?: string }) {
  const [failedSrc, setFailedSrc] = useState<string | null>(null);
  if (src && src !== failedSrc) {
    return (
      <Image
        src={src}
        alt=""
        width={20}
        height={20}
        unoptimized
        draggable={false}
        onError={() => setFailedSrc(src)}
        className="h-5 w-5 shrink-0 rounded-[4px]"
      />
    );
  }
  return <KindIcon kind={kind} className={className} />;
}
