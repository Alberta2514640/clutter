"use client";

import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";

interface DashboardHeaderProps {
  organizationName?: string;
}

export default function DashboardHeader({ organizationName }: DashboardHeaderProps) {
  const router = useRouter();

  return (
    <div className="mb-8 flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 className="text-4xl font-bold mb-2">Dashboard</h1>
        {organizationName && (
          <p className="text-gray-400">
            Organization: <span className="text-gray-200">{organizationName}</span>
          </p>
        )}
      </div>
      <Button onClick={() => router.push("/projects/new")} className="bg-teal-500 hover:bg-teal-600 flex items-center gap-2">
        <Plus className="w-4 h-4" aria-hidden="true" />
        New Project
      </Button>
    </div>
  );
}
