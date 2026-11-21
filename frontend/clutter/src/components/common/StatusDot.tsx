import { cn } from "@/lib/utils";

export type StatusTone = "neutral" | "primary" | "success" | "warning" | "danger" | "info";

// Dots are the scale's solid step (Radix step 9).
const toneClass: Record<StatusTone, string> = {
  neutral: "bg-muted-foreground",
  primary: "bg-primary",
  success: "bg-success-solid",
  warning: "bg-warning-solid",
  danger: "bg-danger-solid",
  info: "bg-info-solid",
};

type Props = {
  tone?: StatusTone;
  // A ring that pulses outwards while something runs. It adds nothing a
  // person needs: the label says "Applying" too.
  pulse?: boolean;
  // The status in words, shown beside the dot. Without it the dot is
  // decorative and the status must be written nearby.
  label?: string;
  className?: string;
};

// StatusDot is a small coloured dot for a status, with its words beside it.
export default function StatusDot({ tone = "neutral", pulse = false, label, className }: Props) {
  const dot = (
    <span aria-hidden="true" className="relative inline-flex size-2 shrink-0">
      {pulse && <span className={cn("absolute inset-0 rounded-full opacity-0 animate-status-ping", toneClass[tone])} />}
      <span className={cn("relative inline-flex size-2 rounded-full", toneClass[tone])} />
    </span>
  );
  if (!label) return <span className={cn("inline-flex", className)}>{dot}</span>;
  return (
    <span data-tone={tone} className={cn("inline-flex items-center gap-2 text-sm", className)}>
      {dot}
      <span>{label}</span>
    </span>
  );
}
