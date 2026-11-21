"use client";

import { useCurrentOrganizationStore } from "@/lib/features/organization/uiStore";
import Link from "next/link";
import type { ReactNode } from "react";

const CONNECTIONS_HREF = "/settings/organization/connections";

// A link to the Connections page for this diagram's organization. That page
// shows the organization chosen in this browser, which can differ from the
// project's own, so the link chooses the project's organization first.
export default function ConnectionsLink({
  organizationId,
  className,
  children,
}: {
  organizationId: string;
  className?: string;
  children: ReactNode;
}) {
  const select = useCurrentOrganizationStore((s) => s.select);
  return (
    <Link href={CONNECTIONS_HREF} onClick={() => select(organizationId)} className={className}>
      {children}
    </Link>
  );
}
