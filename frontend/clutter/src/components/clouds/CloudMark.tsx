"use client";

import { monogram, type CloudLook } from "@/lib/features/catalog/display";
import { X } from "lucide-react";
import Image from "next/image";
import { useState } from "react";

type Props = {
  cloud: CloudLook;
  // Width and height in pixels; fixed, so nothing moves while the logo loads.
  size?: number;
  // Faded with a small cross: the cloud does not build what this mark sits on.
  off?: boolean;
  // The mark's accessible name and tooltip. Leave it out where the cloud's
  // name is written beside the mark; the mark is then decorative.
  label?: string;
  className?: string;
};

// A cloud's logo, from its manifest's logo key (catalog/display.ts). A logo
// that is missing or fails to load shows the cloud's monogram instead.
export default function CloudMark({ cloud, size = 20, off = false, label, className = "" }: Props) {
  const [failedSrc, setFailedSrc] = useState<string | null>(null);
  const src = cloud.logoSrc && cloud.logoSrc !== failedSrc ? cloud.logoSrc : null;

  return (
    <span
      role={label ? "img" : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      title={label}
      data-cloud={cloud.connector}
      data-off={off || undefined}
      className={`relative inline-flex shrink-0 items-center justify-center ${className}`}
      style={{ width: size, height: size }}>
      {src ? (
        <Image
          src={src}
          alt=""
          width={size}
          height={size}
          unoptimized
          draggable={false}
          onError={() => setFailedSrc(src)}
          className={off ? "opacity-40 grayscale" : undefined}
        />
      ) : (
        <span
          className={`flex h-full w-full items-center justify-center rounded-[22%] border border-slate-500 bg-slate-800 font-bold leading-none text-slate-100 ${off ? "opacity-40" : ""}`}
          style={{ fontSize: Math.max(7, Math.round(size * 0.38)) }}>
          {monogram(cloud.shortName)}
        </span>
      )}
      {off && (
        <span
          aria-hidden
          className="pointer-events-none absolute -bottom-1 -right-1 flex items-center justify-center rounded-full bg-slate-500 text-slate-950 ring-1 ring-slate-950"
          style={{ width: Math.max(8, Math.round(size * 0.55)), height: Math.max(8, Math.round(size * 0.55)) }}>
          <X strokeWidth={4} style={{ width: "75%", height: "75%" }} />
        </span>
      )}
    </span>
  );
}
