"use client";

import CloudLogo from "@/components/common/CloudLogo";
import type { CloudLook } from "@/lib/features/catalog/display";

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

// A cloud's mark for the screens that hold a manifest's CloudLook. It draws
// through CloudLogo, the one logo registry (and its switch back to neutral
// tiles); the manifest's own logo is used only for a connector the registry
// does not list.
export default function CloudMark({ cloud, size = 20, off = false, label, className }: Props) {
  return (
    <CloudLogo connector={cloud.connector} shortName={cloud.shortName} src={cloud.logoSrc} size={size} off={off} label={label} className={className} />
  );
}
