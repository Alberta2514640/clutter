"use client";

import type { UserData } from "@/lib/features/user/types";

function initials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  const a = parts[0]?.[0] ?? "?";
  const b = parts.length > 1 ? parts[parts.length - 1][0] : "";
  return (a + b).toUpperCase();
}

// The backend gives an organization's member count and member ids, not their
// names, so only the signed-in user is shown by name.
export default function Members({ me, memberCount }: { me: UserData | null; memberCount: number }) {
  const others = Math.max(memberCount - 1, 0);

  return (
    <section className="space-y-3" aria-labelledby="members-heading">
      <h2 id="members-heading" className="text-sm font-medium text-slate-200">
        Members
      </h2>

      <div className="overflow-hidden rounded-xl border border-slate-800 bg-slate-950/40">
        <ul className="divide-y divide-slate-800">
          {me && (
            <li className="flex items-center gap-3 px-4 py-3">
              <div className="h-9 w-9 rounded-full bg-slate-800/70 border border-slate-700 flex items-center justify-center text-xs font-semibold text-slate-100 shrink-0" aria-hidden="true">
                {initials(me.displayName || me.email)}
              </div>
              <div className="min-w-0">
                <div className="text-sm text-slate-100 truncate">{me.displayName} (you)</div>
                <div className="text-xs text-slate-400 truncate">{me.email}</div>
              </div>
            </li>
          )}
          {others > 0 && (
            <li className="px-4 py-3 text-sm text-slate-300">
              {others === 1 ? "1 other member" : `${others} other members`}
            </li>
          )}
        </ul>
      </div>

      <p className="text-xs text-slate-500">Invites are not available yet.</p>
    </section>
  );
}
