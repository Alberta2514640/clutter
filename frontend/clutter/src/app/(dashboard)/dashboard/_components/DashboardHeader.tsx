"use client";

import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";

interface DashboardHeaderProps {
  tenantName?: string;
}

export default function DashboardHeader({ tenantName }: DashboardHeaderProps) {
  const router = useRouter();

  return (
    <div className="flex items-center justify-between mb-8">
      <div>
        <h1 className="text-4xl font-bold mb-2">Dashboard</h1>
        <p className="text-gray-400">{tenantName || "Your Organization"}</p>
      </div>
      <Button onClick={() => router.push("/projects/new")} className="bg-teal-500 hover:bg-teal-600 flex items-center gap-2">
        <Plus className="w-4 h-4" />
        New Project
      </Button>
    </div>
  );
}
