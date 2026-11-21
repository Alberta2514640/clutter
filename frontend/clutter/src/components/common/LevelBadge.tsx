import { Badge } from "@/components/ui/badge";
import { levelLook } from "@/lib/features/examples/types";

// An example's level (Starter, Intermediate, Advanced), the one way every
// screen shows it: the landing page, onboarding and the New diagram dialog.
export default function LevelBadge({ level, className }: { level: string; className?: string }) {
  const { label, tone } = levelLook(level);
  return (
    <Badge variant={tone} className={className}>
      {label}
    </Badge>
  );
}
