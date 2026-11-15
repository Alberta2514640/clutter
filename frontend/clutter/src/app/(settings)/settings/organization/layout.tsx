// src/app/(settings)/organization/layout.tsx
"use client";

import { usePathname } from "next/navigation";
import React from "react";
import HeaderSettings from "./_components/Header";
import Tabs from "./_components/Tab";

export default function OrganizationSettingsLayout({children,}: {children: React.ReactNode;}) {
  const pathname = usePathname();
  // A new organization is not a setting of the current one: that page has its
  // own header and no tabs.
  if (pathname?.endsWith("/create-org")) return <div>{children}</div>;
  return (
    <div>
      <HeaderSettings/>

      <Tabs/>

      {children}
    </div>
  );
}
