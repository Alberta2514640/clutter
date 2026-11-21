"use client";

import { monogram } from "@/lib/features/catalog/display";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";
import Image from "next/image";
import { useState } from "react";

// The one place a cloud's logo is chosen; every screen that names a cloud
// renders it through CloudLogo. The files and their sources are listed in
// public/clouds/README.md.
//
// false switches every place back to the neutral tiles drawn for Clutter
// (and gives a connector this registry does not know its monogram).
export const SHOW_OFFICIAL_CLOUD_LOGOS = true;

type CloudLogoEntry = {
  shortName: string;
  // The vendor's own logo, unmodified. Absent when there is none to use.
  official?: string;
  // Clutter's neutral tile with the cloud's name.
  neutral: string;
};

// Keyed by connector id, the same id as B's manifests and public/clouds/<id>/.
export const CLOUD_LOGOS: Record<string, CloudLogoEntry> = {
  aws: { shortName: "AWS", official: "/clouds/aws/logo.svg", neutral: "/clouds/aws/neutral.svg" },
  "general-openstack": {
    shortName: "OpenStack",
    official: "/clouds/general-openstack/logo.svg",
    neutral: "/clouds/general-openstack/neutral.svg",
  },
  // The test cloud has only Clutter's own drawing.
  mock: { shortName: "Mock", neutral: "/clouds/mock/logo.svg" },
};

// cloudLogoSrc is the file to show for a connector. manifestSrc is the logo
// B's manifest names, used for a connector this registry does not list.
function cloudLogoSrc(connector: string, manifestSrc?: string | null): string | null {
  const entry = CLOUD_LOGOS[connector];
  if (!entry) return SHOW_OFFICIAL_CLOUD_LOGOS ? (manifestSrc ?? null) : null;
  return SHOW_OFFICIAL_CLOUD_LOGOS && entry.official ? entry.official : entry.neutral;
}

type Props = {
  // The connector id: "aws", "general-openstack", "mock".
  connector: string;
  // The cloud's short name, for the monogram when no file loads.
  shortName?: string;
  // The manifest's logo, for a connector not in CLOUD_LOGOS.
  src?: string | null;
  // Width and height in pixels; fixed, so nothing moves while the logo loads.
  size?: number;
  // Faded with a small cross: the cloud does not build what this mark sits on.
  off?: boolean;
  // The mark's accessible name and tooltip. Leave it out where the cloud's
  // name is written beside the mark; the mark is then decorative.
  label?: string;
  className?: string;
};

// CloudLogo shows a cloud's logo at a fixed size. A logo that is missing or
// fails to load shows the cloud's monogram instead.
export default function CloudLogo({ connector, shortName, src, size = 20, off = false, label, className }: Props) {
  const [failedSrc, setFailedSrc] = useState<string | null>(null);
  const chosen = cloudLogoSrc(connector, src);
  const shown = chosen && chosen !== failedSrc ? chosen : null;
  const name = shortName ?? CLOUD_LOGOS[connector]?.shortName ?? connector;

  return (
    <span
      role={label ? "img" : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
      title={label}
      data-cloud={connector}
      data-off={off || undefined}
      className={cn("relative inline-flex shrink-0 items-center justify-center", className)}
      style={{ width: size, height: size }}>
      {shown ? (
        <Image
          src={shown}
          alt=""
          width={size}
          height={size}
          // Both set in CSS too: Tailwind's base sets img height to auto, and
          // next/image warns when only one dimension is changed.
          style={{ width: size, height: size }}
          unoptimized
          draggable={false}
          onError={() => setFailedSrc(shown)}
          className={cn("object-contain", off && "opacity-40 grayscale")}
        />
      ) : (
        <span
          className={cn(
            "border-border bg-muted text-foreground flex h-full w-full items-center justify-center rounded-[22%] border font-bold leading-none",
            off && "opacity-40",
          )}
          style={{ fontSize: Math.max(7, Math.round(size * 0.38)) }}>
          {monogram(name)}
        </span>
      )}
      {off && (
        <span
          aria-hidden
          className="bg-muted-foreground text-background ring-background pointer-events-none absolute -right-1 -bottom-1 flex items-center justify-center rounded-full ring-1"
          style={{ width: Math.max(8, Math.round(size * 0.55)), height: Math.max(8, Math.round(size * 0.55)) }}>
          <X strokeWidth={4} style={{ width: "75%", height: "75%" }} />
        </span>
      )}
    </span>
  );
}
