import ClutterLogo from "@/components/common/ClutterLogo";
import { cn } from "@/lib/utils";

type Props = {
  // "loop": the logo's blocks drop in, hold and fade, every 1.4 s, for a
  // page that is loading. "once": they drop in and stay. Default "loop".
  mode?: "loop" | "once";
  // Height of the logo in pixels. Default 32.
  size?: number;
  className?: string;
};

// IsoLoader is the logo assembling. It is decorative: the page around it
// says what is loading (role="status" with its own text). Under reduced
// motion it is the still logo.
export default function IsoLoader({ mode = "loop", size = 32, className }: Props) {
  return (
    <span aria-hidden="true" className={cn("inline-flex", className)}>
      <ClutterLogo size={size} motion={mode} />
    </span>
  );
}
