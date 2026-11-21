import { ListChecks } from "lucide-react";
import type { ReactNode } from "react";

// BeforeYouStart says what a person needs, and what happens, before they
// link or connect a cloud. Used by the AWS link flow (on Connections and the
// AWS account page) and by the OpenStack credential form.
export default function BeforeYouStart({ items, footer }: { items: { lead?: string; text: string }[]; footer?: ReactNode }) {
  return (
    <section aria-label="Before you start" className="space-y-3 rounded-2xl border border-slate-700 bg-slate-950/50 px-5 py-4 text-sm text-slate-300">
      <p className="flex items-center gap-2 font-semibold text-white">
        <ListChecks className="h-4 w-4 shrink-0 text-teal-400" aria-hidden="true" />
        Before you start
      </p>
      <ul className="list-disc space-y-1.5 pl-5 leading-6">
        {items.map((item) => (
          <li key={item.text}>
            {item.lead && <span className="font-medium text-slate-100">{item.lead} </span>}
            {item.text}
          </li>
        ))}
      </ul>
      {footer}
    </section>
  );
}
