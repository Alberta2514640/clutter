// src/app/(settings)/organization/layout.tsx
"use client";

import React from "react";
import HeaderSettings from "./_components/Header";
import Tabs from "./_components/Tab";

export default function OrganizationSettingsLayout({children,}: {children: React.ReactNode;}) {
  return (
    <div>
      <HeaderSettings/>

      <Tabs/>

      {children}
    </div>
  );
}
