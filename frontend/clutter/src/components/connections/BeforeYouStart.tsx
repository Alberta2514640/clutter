import { ListChecks } from "lucide-react";
import type { ReactNode } from "react";

// BeforeYouStart says what a person needs, and what happens, before they
// link or connect a cloud: the AWS link step and the OpenStack form.
export default function BeforeYouStart({ items, footer }: { items: { lead?: string; text: string }[]; footer?: ReactNode }) {
  return (
    <section aria-label="Before you start" className="bg-muted/40 space-y-2.5 rounded-lg border px-4 py-3.5 text-sm">
      <p className="text-foreground flex items-center gap-2 font-medium">
        <ListChecks className="text-link size-4 shrink-0" aria-hidden="true" />
        Before you start
      </p>
      <ul className="text-muted-foreground list-disc space-y-1 pl-5 leading-6">
        {items.map((item) => (
          <li key={item.text}>
            {item.lead && <span className="text-foreground font-medium">{item.lead} </span>}
            {item.text}
          </li>
        ))}
      </ul>
      {footer}
    </section>
  );
}
