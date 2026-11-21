import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { AlertCircle, CheckCircle2, Info } from "lucide-react";
import type { ReactNode } from "react";

// The ids a field's input points at with aria-describedby.
const hintId = (id: string) => `${id}-hint`;
const errorId = (id: string) => `${id}-error`;

// describedBy is the input's aria-describedby: its hint, then its error.
export function describedBy(id: string, hint: boolean, error: boolean): string | undefined {
  return [hint && hintId(id), error && errorId(id)].filter(Boolean).join(" ") || undefined;
}

// Field is one labelled input of the connection flow, with its hint under it
// and its error in place of nothing, right next to the input. The caller
// renders the input with id, aria-invalid and aria-describedby (describedBy).
export function Field({
  id,
  label,
  hint,
  error,
  className,
  children,
}: {
  id: string;
  label: ReactNode;
  hint?: ReactNode;
  error?: string | null;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("space-y-1.5", className)}>
      <Label htmlFor={id}>{label}</Label>
      {children}
      {error && (
        <p id={errorId(id)} className="text-danger flex items-start gap-1.5 text-xs animate-fade-in">
          <AlertCircle className="mt-px size-3.5 shrink-0" aria-hidden="true" />
          <span>{error}</span>
        </p>
      )}
      {hint && (
        <p id={hintId(id)} className="text-muted-foreground text-xs">
          {hint}
        </p>
      )}
    </div>
  );
}

// Notice is a boxed message inside the flow: an error (announced), a success
// or a plain note.
export function Notice({ tone, children, className }: { tone: "error" | "info" | "success"; children: ReactNode; className?: string }) {
  const Icon = tone === "error" ? AlertCircle : tone === "success" ? CheckCircle2 : Info;
  return (
    <div
      role={tone === "error" ? "alert" : "status"}
      className={cn(
        "flex items-start gap-2.5 rounded-lg border px-3.5 py-3 text-sm animate-fade-in",
        tone === "error" && "border-danger/30 bg-danger/5 text-foreground",
        tone === "success" && "border-success/30 bg-success/5 text-foreground",
        tone === "info" && "bg-muted/50 text-foreground",
        className,
      )}>
      <Icon
        className={cn("mt-0.5 size-4 shrink-0", tone === "error" ? "text-danger" : tone === "success" ? "text-success" : "text-muted-foreground")}
        aria-hidden="true"
      />
      <div className="min-w-0 space-y-2">{children}</div>
    </div>
  );
}
