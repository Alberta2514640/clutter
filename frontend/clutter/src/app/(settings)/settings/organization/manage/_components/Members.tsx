"use client";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import type { UserData } from "@/lib/features/user/types";
import { plural } from "@/lib/text";

function initials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  const a = parts[0]?.[0] ?? "?";
  const b = parts.length > 1 ? parts[parts.length - 1][0] : "";
  return (a + b).toUpperCase();
}

// A's organization list gives a team's member count and member ids, not their
// names, so only the signed-in person is shown by name.
export default function Members({ me, memberCount }: { me: UserData | null; memberCount: number }) {
  const others = Math.max(memberCount - 1, 0);

  return (
    <section aria-labelledby="members-heading" className="surface-card">
      <div className="p-5 md:p-6">
        <h2 id="members-heading" className="text-foreground text-base font-semibold">
          Members
        </h2>
        <p className="text-muted-foreground mt-1 text-sm">{plural(memberCount, "person is", "people are")} in this team.</p>
      </div>

      <ul className="divide-y border-t">
        {me && (
          <li className="flex items-center gap-3 px-5 py-4 md:px-6">
            <Avatar className="size-9">
              {me.pictureUrl && <AvatarImage src={me.pictureUrl} alt="" referrerPolicy="no-referrer" />}
              <AvatarFallback className="bg-primary/10 text-link text-xs font-semibold">{initials(me.displayName || me.email)}</AvatarFallback>
            </Avatar>
            <div className="min-w-0 flex-1">
              <p className="text-foreground truncate text-sm font-medium">{me.displayName}</p>
              <p className="text-muted-foreground truncate text-xs">{me.email}</p>
            </div>
            <Badge variant="outline">You</Badge>
          </li>
        )}
        {others > 0 && (
          <li className="text-muted-foreground px-5 py-4 text-sm md:px-6">
            And {plural(others, "other member", "other members")}. Clutter does not show other members&apos; names yet.
          </li>
        )}
      </ul>

      <p className="text-muted-foreground bg-muted/40 rounded-b-xl border-t px-5 py-3 text-sm md:px-6">Inviting people to a team is not available yet.</p>
    </section>
  );
}
