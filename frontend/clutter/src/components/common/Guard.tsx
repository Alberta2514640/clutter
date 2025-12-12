"use client";
//commented out due to import errors

// import { useAuth } from "@/providers/AuthProvider";
import React from "react";
export default function Guard({ children }: { children: React.ReactNode }) {
  // const { loading } = useAuth();
  // if (loading) return <div>Loading…</div>;
  return <>{children}</>;
}
