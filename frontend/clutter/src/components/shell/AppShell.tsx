"use client";
import React from "react";
export default function AppShell({ children }: { children: React.ReactNode }) {
  return <div className="min-h-dvh flex flex-col">{children}</div>;
}
