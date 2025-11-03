"use client";
import React from "react";
import { useAuth } from "@/providers/AuthProvider";
export default function Guard({ children }: { children: React.ReactNode }) {
  const { loading } = useAuth();
  if (loading) return <div>Loading…</div>;
  return <>{children}</>;
}
